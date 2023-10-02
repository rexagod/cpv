package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/rexagod/cpv/internal/client"
	"github.com/rexagod/cpv/internal/profiles"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

const (
	invalidProfileErr = "invalid profile: %s"
	contextTimout     = 10 * time.Second
)

func main() {

	// Initialize and validate flags.
	var (
		bearerToken       string
		address           string
		kubeconfigPath    string
		profile           string
		status            bool
		guessProfile      string
		guessProfileParam string
	)
	flag.StringVar(&bearerToken, "bearer-token", "", "Bearer token for authentication.")
	flag.StringVar(&address, "address", "http://localhost:9090", "Address of the Prometheus instance.")
	flag.StringVar(&kubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to kubeconfig file.")
	flag.StringVar(&profile, "profile", "", "Collection profile to run the validation against.")
	flag.BoolVar(&status, "status", false, "Report collection profiles implementation status.")
	flag.StringVar(&guessProfile, "guess-profile", "", "Guess the metrics needed to implement the given collection profile.")

	// Specifying targets: https://github.com/prometheus/client_golang/blob/644c80d1360fb1409a3fe8dfc5bad4228f282f3b/api/prometheus/v1/api_test.go#L1007
	flag.StringVar(&guessProfileParam, "guess-profile-param", "", "Path to rule file, or targets to be used to guess the metrics needed to implement the --guess-profile.")
	flag.Parse()
	if len(bearerToken) == 0 {
		klog.Fatal("Bearer token must be set")
	}
	if len(address) == 0 {
		klog.Fatal("Address must be set")
	}
	if len(kubeconfigPath) == 0 {
		klog.Fatal("KUBECONFIG must be set")
	}

	// Create a new client.
	ctx, cancel := context.WithTimeout(context.Background(), contextTimout)
	defer cancel()
	c := client.NewClient(ctx, address, bearerToken)
	if err := c.Init(); err != nil {
		klog.Fatal(err)
	}

	// Create a new Kube client.
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		klog.Fatal(err)
	}
	dc, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	// Call profile-specific operator to validate the respective profile.
	if profile != "" {
		profile := profiles.CollectionProfile(profile)
		if !profiles.IsSupportedCollectionProfile(profile) {
			klog.Fatalf(invalidProfileErr, profile)
		}
		err = profiles.ProfileOperators[profile].Operator(ctx, dc, c)
		if err != nil {
			klog.Error(err)
		}
	}

	// Call profile-specific guesser to guess the metrics needed to implement the respective profile.
	if guessProfile != "" {
		if guessProfileParam == "" {
			klog.Fatal("guess-profile-param must be set when using --guess-profile")
		}
		guessProfile := profiles.CollectionProfile(guessProfile)
		if !profiles.IsSupportedCollectionProfile(guessProfile) {
			klog.Fatalf(invalidProfileErr, guessProfile)
		}
		err = profiles.ProfileGuessers[guessProfile].Guess(ctx, dc, c, guessProfileParam)
		if err != nil {
			klog.Error(err)
		}
	}

	// Report implementation status for all supported profiles.
	if status {
		err = profiles.ReportImplementationStatus(ctx, dc)
		if err != nil {
			klog.Error(err)
		}
	}
}
