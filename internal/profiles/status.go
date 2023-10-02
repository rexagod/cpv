package profiles

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"os"
	"text/tabwriter"
	"time"
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

	// Write the implementation status to a file.
	file, err := os.Create("/tmp/" + fmt.Sprintf("implementation-status-%s-recoder.txt", time.Now().Format("2006-01-02T15:04:05")))
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

	// We assume that the default profile is always implemented.
	defaultProfileServiceMonitorsSet := mServiceMonitors[FullCollectionProfile]
	for serviceMonitor := range defaultProfileServiceMonitorsSet {
		for _, profile := range SupportedNonDefaultCollectionProfiles {
			// We assume monitors will adhere to a naming standard as defined in the original implementation.
			// Refer: https://github.com/openshift/cluster-monitoring-operator/pull/1785/files#diff-229e84547c808580dd069005f5467c35c491380b90690771b1f1d44454067e02R10.
			if !mServiceMonitors[profile].Has(serviceMonitor + "-" + string(profile)) {
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
			if !mPodMonitors[profile].Has(podMonitor + "-" + string(profile)) {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", profile, "", podMonitor, ErrImplemented)
			}
		}
	}

	_ = w.Flush()
	// Delete the file if there are no implementation issues.
	if *recorder.implementationIssues > 0 {
		klog.Errorf(ErrNonNilIssues, *recorder.implementationIssues, file.Name())
	} else {
		_ = os.Remove(file.Name())
	}

	return nil
}
