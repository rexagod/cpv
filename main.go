package main

import (
	"context"
	"flag"
	"github.com/rexagod/cpv/internal/client"
	"github.com/rexagod/cpv/internal/profiles"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"os"
	"time"
)

func main() {
	// Initialize and validate flags.
	var (
		bearerToken         string
		address             string
		kubeconfigPath      string
		profile             string
		implStats           bool
		guessMinimalProfile string
	)
	flag.StringVar(&bearerToken, "bearer-token", "", "Bearer token for authentication.")
	flag.StringVar(&address, "address", "http://localhost:9090", "Address of the Prometheus instance.")
	flag.StringVar(&kubeconfigPath, "kubeconfigPath", os.Getenv("KUBECONFIG"), "Path to kubeconfig file.")
	flag.StringVar(&profile, "profile", "", "Collection profile to run the validation against.")
	flag.BoolVar(&implStats, "impl-stats", false, "Report collection profiles implementation status.")
	// Usage: https://github.com/prometheus/client_golang/blob/644c80d1360fb1409a3fe8dfc5bad4228f282f3b/api/prometheus/v1/api_test.go#L1007
	flag.StringVar(&guessMinimalProfile, "guess-minimal-profile", "", "Guess the metrics needed to implement minimal collection profile, can be a path to a rule file, or a set of constraints (targets) to fetch metrics from.")
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	collectionProfile := profiles.CollectionProfile(profile)
	if profile != "" && profiles.IsSupportedCollectionProfile(collectionProfile) {
		err = profiles.ProfileOperators[collectionProfile](ctx, dc, c)
		if err != nil {
			klog.Error(err)
		}
	} else {
		klog.Fatalf("Invalid profile %s", profile)
	}

	// Report implementation status for all supported profiles.
	if implStats {
		err = profiles.ReportImplementationStatus(ctx, dc)
		if err != nil {
			klog.Error(err)
		}
	}

	if guessMinimalProfile != "" {
		err = profiles.ProfileGuessers[profiles.MinimalCollectionProfile](ctx, dc, c, guessMinimalProfile)
		if err != nil {
			klog.Error(err)
		}
	}
}
