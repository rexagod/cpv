package profiles

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// ReportImplementationStatus reports the implementation status w.r.t. all supported collection profiles, and points out
// the monitors that are absent (partial implementations).
// NOTE: The general assumption for a monitor not implementing a particular profile translates to the fact that the end
// user simply do not want to keep ANY metrics when operating under that profile.
func ReportImplementationStatus(ctx context.Context, dc *dynamic.DynamicClient, profile CollectionProfile, noisy bool) error {
	profilesRange := SupportedCollectionProfiles

	// Restrict the range of profiles to the one specified by the user.
	if len(profile) > 0 {
		profilesRange = CollectionProfiles{
			profile,
			FullCollectionProfile, // required within the profile range to compare the given profile with the default profile.
		}
	}
	mServiceMonitors := make(map[CollectionProfile]sets.Set[string])
	mPodMonitors := make(map[CollectionProfile]sets.Set[string])
	for _, p := range profilesRange {
		mServiceMonitors[p] = sets.Set[string]{}
		podMonitors, serviceMonitors, err := fetchMonitorsForProfile(ctx, dc, p, noisy)
		if err != nil {
			return fmt.Errorf("failed to fetch service monitors for profile %s: %w", p, err)
		}
		for _, serviceMonitor := range serviceMonitors.Items {
			mServiceMonitors[p].Insert(serviceMonitor.GetName())
		}
		for _, podMonitor := range podMonitors.Items {
			mPodMonitors[p].Insert(podMonitor.GetName())
		}
	}

	// Write the implementation status to a file.
	file, err := os.CreateTemp("/tmp", "implementation-status-*.log")
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()
	recorder := &Recorder{file: file, implementationIssues: new(uint)}
	w := tabwriter.NewWriter(recorder, 0, 0, 2, ' ', 0)
	columns := fmt.Sprintf("PROFILE\tSERVICE MONITOR\tPOD MONITOR\tERROR")
	_, _ = fmt.Fprintln(w, columns)

	// No need to check for non-default profiles when comparing the base (default) profile with the default profile.
	if profile != FullCollectionProfile {
		// We assume that the default profile is always implemented.
		defaultProfileServiceMonitorsSet := mServiceMonitors[FullCollectionProfile]
		for serviceMonitor := range defaultProfileServiceMonitorsSet {
			for _, profile := range SupportedNonDefaultCollectionProfiles {
				// We assume monitors will adhere to a naming standard as defined in the original implementation.
				// Refer: https://github.com/openshift/cluster-monitoring-operator/pull/1785/files#diff-229e84547c808580dd069005f5467c35c491380b90690771b1f1d44454067e02R10.
				if !strings.HasSuffix(serviceMonitor, string(profile)) && !mServiceMonitors[profile].Has(serviceMonitor+"-"+string(profile)) {
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", profile, serviceMonitor, "", ErrImplemented)
				}
			}
		}

		// We assume that the default profile is always implemented.
		defaultProfilePodMonitorsSet := mPodMonitors[FullCollectionProfile]
		for podMonitor := range defaultProfilePodMonitorsSet {
			for _, profile := range SupportedNonDefaultCollectionProfiles {
				// We assume monitors will adhere to a naming standard as defined in the original implementation.
				// Refer: https://github.com/openshift/cluster-monitoring-operator/pull/1785/files#diff-229e84547c808580dd069005f5467c35c491380b90690771b1f1d44454067e02R10.
				if !strings.HasSuffix(podMonitor, string(profile)) && !mPodMonitors[profile].Has(podMonitor+"-"+string(profile)) {
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", profile, "", podMonitor, ErrImplemented)
				}
			}
		}
	}

	_ = w.Flush()
	// Delete the file if there are no implementation issues.
	if *recorder.implementationIssues > 0 {
		klog.Errorf("encountered %d issues, refer: %s", *recorder.implementationIssues, file.Name())
	} else {
		_ = os.Remove(file.Name())
	}

	return nil
}
