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
		bearerToken    string
		address        string
		kubeconfigPath string
		profile        string
		implStats      bool
	)
	flag.StringVar(&bearerToken, "bearer-token", "", "Bearer token for authentication.")
	flag.StringVar(&address, "address", "http://localhost:9090", "Address of the Prometheus instance.")
	flag.StringVar(&kubeconfigPath, "kubeconfigPath", os.Getenv("KUBECONFIG"), "Path to kubeconfig file.")
	flag.StringVar(&profile, "profile", "", "Collection profile to run the validation against.")
	flag.BoolVar(&implStats, "impl-stats", false, "Report collection profiles implementation status.")
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
		err = profiles.ProfileOperators[collectionProfile](ctx, collectionProfile, dc, c)
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
}
