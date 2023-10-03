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
		bearerToken            string
		address                string
		kubeconfigPath         string
		profile                string
		status                 bool
		extractForProfile      string
		extractForProfileParam string
		outputCardinality      bool
	)
	flag.StringVar(&bearerToken, "bearer-token", "", "Bearer token for authentication.")
	flag.StringVar(&address, "address", "http://localhost:9090", "Address of the Prometheus instance.")
	flag.StringVar(&kubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to kubeconfig file.")
	flag.StringVar(&profile, "profile", "", "Collection profile to run the validation against.")
	flag.BoolVar(&status, "status", false, "Report collection profiles implementation status.")
	flag.StringVar(&extractForProfile, "extract-for-profile", "", "Extract the metrics needed to implement the given collection profile.")

	// Specifying targets: https://github.com/prometheus/client_golang/blob/644c80d1360fb1409a3fe8dfc5bad4228f282f3b/api/prometheus/v1/api_test.go#L1007
	flag.StringVar(&extractForProfileParam, "extract-for-profile-param", "", "Path to rule file, or targets to be used to extract the metrics needed to implement the -extract-for-profile.")
	flag.BoolVar(&outputCardinality, "output-cardinality", false, "Output cardinality of all extracted metrics (while using -extract-for-profile-*).")
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
		p := profiles.CollectionProfile(profile)
		if !profiles.IsSupportedCollectionProfile(p) {
			klog.Fatalf(invalidProfileErr, p)
		}
		err = profiles.ProfileOperators[p].Operator(ctx, dc, c)
		if err != nil {
			klog.Error(err)
		}
	}

	// Call profile-specific extractor to extract the metrics needed to implement the respective profile.
	if extractForProfile != "" {
		if extractForProfileParam == "" {
			klog.Fatal("extract-for-profile-param must be set when using --extract-for-profile")
		}
		extractProfile := profiles.CollectionProfile(extractForProfile)
		if !profiles.IsSupportedCollectionProfile(extractProfile) {
			klog.Fatalf(invalidProfileErr, extractProfile)
		}
		err = profiles.ProfileExtractors[extractProfile].Extract(ctx, dc, c, extractForProfileParam, outputCardinality)
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
