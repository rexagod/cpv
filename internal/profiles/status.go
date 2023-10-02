package profiles

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
)

// ReportImplementationStatus reports the implementation status w.r.t. all supported collection profiles, and points out
// the monitors that are absent (partial implementations).
// NOTE: The general assumption for a monitor not implementing a particular profile translates to the fact that the end
// user simply do not want to keep ANY metrics when operating under that profile.
func ReportImplementationStatus(ctx context.Context, dc *dynamic.DynamicClient) error {
	mServiceMonitors := make(map[CollectionProfile]sets.Set[string])
	mPodMonitors := make(map[CollectionProfile]sets.Set[string])
	for _, profile := range SupportedCollectionProfiles {
		mServiceMonitors[profile] = sets.Set[string]{}
		podMonitors, serviceMonitors, err := fetchMonitorsForProfile(ctx, dc, profile)
		if err != nil {
			return fmt.Errorf("failed to fetch service monitors for profile %s: %w", profile, err)
		}
		for _, serviceMonitor := range serviceMonitors.Items {
			mServiceMonitors[profile].Insert(serviceMonitor.GetName())
		}
		for _, podMonitor := range podMonitors.Items {
			mPodMonitors[profile].Insert(podMonitor.GetName())
		}
	}

	// We assume that the default profile is always implemented.
	defaultProfileServiceMonitorsSet := mServiceMonitors[FullCollectionProfile]
	for serviceMonitor := range defaultProfileServiceMonitorsSet {
		for _, profile := range SupportedNonDefaultCollectionProfiles {
			// We assume monitors will adhere to a naming standard as defined in the original implementation.
			// Refer: https://github.com/openshift/cluster-monitoring-operator/pull/1785/files#diff-229e84547c808580dd069005f5467c35c491380b90690771b1f1d44454067e02R10.
			if !mServiceMonitors[profile].Has(serviceMonitor + "-" + string(profile)) {
				fmt.Printf("Service monitor %s is not implemented for profile %s\n", serviceMonitor, profile)
			}
		}
	}

	// We assume that the default profile is always implemented.
	defaultProfilePodMonitorsSet := mPodMonitors[FullCollectionProfile]
	for podMonitor := range defaultProfilePodMonitorsSet {
		for _, profile := range SupportedNonDefaultCollectionProfiles {
			// We assume monitors will adhere to a naming standard as defined in the original implementation.
			// Refer: https://github.com/openshift/cluster-monitoring-operator/pull/1785/files#diff-229e84547c808580dd069005f5467c35c491380b90690771b1f1d44454067e02R10.
			if !mPodMonitors[profile].Has(podMonitor + "-" + string(profile)) {
				fmt.Printf("Pod monitor %s is not implemented for profile %s\n", podMonitor, profile)
			}
		}
	}

	return nil
}
