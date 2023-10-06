package main

import (
	"context"
	"flag"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/rexagod/cpv/internal/client"
	"github.com/rexagod/cpv/internal/options"
	"github.com/rexagod/cpv/internal/profiles"
)

const (
	contextTimeout    = 5 * time.Minute
	invalidProfileErr = "invalid profile: %s"
)

func main() {

	// Get options.
	o := options.NewOptions()

	// Check if the endpoint at -address is up.
	err := o.IsUp()
	if err != nil {
		klog.Fatal(err)
	}

	// Create a new client.
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	c := client.NewClient(ctx, o.Address, o.BearerToken)
	if err := c.Init(); err != nil {
		klog.Fatal(err)
	}

	// Create a new Kube client.
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", o.KubeconfigPath)
	if err != nil {
		klog.Fatal(err)
	}
	dc, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	// Track if any operation was performed based on the given inputs.
	didOp := false

	// Call profile-specific operator to validate the respective profile.
	if o.Profile != "" && o.Validate {
		didOp = true
		p := profiles.CollectionProfile(o.Profile)
		if !profiles.IsSupportedCollectionProfile(p) {
			klog.Fatalf(invalidProfileErr, p)
		}
		err = profiles.ProfileOperators[p].Operator(
			ctx,
			dc,
			c,
			o.Noisy,
		)
		if err != nil {
			klog.Error(err)
		}
	}

	// Call profile-specific extractor to extract the metrics needed to implement the respective profile.
	if o.Profile != "" && o.HasExtractor() {
		didOp = true
		p := profiles.CollectionProfile(o.Profile)
		if !profiles.IsSupportedCollectionProfile(p) {
			klog.Fatalf(invalidProfileErr, p)
		}
		err = profiles.ProfileExtractors[p].Extract(
			ctx,
			c,
			o.AllowListFile,
			o.RuleFile,
			o.TargetSelectors,
			o.OutputCardinality,
		)
		if err != nil {
			klog.Error(err)
		}
	}

	// Report implementation status for all supported profiles.
	if o.Status {
		didOp = true
		p := profiles.CollectionProfile(o.Profile)
		if !profiles.IsSupportedCollectionProfile(p) && p != "" {
			klog.Fatalf(invalidProfileErr, p)
		}
		err = profiles.ReportImplementationStatus(
			ctx,
			dc,
			p,
			o.Noisy,
		)
		if err != nil {
			klog.Error(err)
		}
	}

	// If no operation was performed, print usage.
	if !didOp {
		flag.Usage()
	}
}
