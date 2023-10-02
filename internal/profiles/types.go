package profiles

type (
	CollectionProfile  string
	CollectionProfiles []CollectionProfile
)

const (

	// CollectionProfileOptInLabel is the label that is used to opted-in a monitor for a particular collection profile.
	// Its absence DOES NOT currently imply that the monitor has opted-in for the FullCollectionProfile.
	CollectionProfileOptInLabel = "monitoring.openshift.io/collection-profile"

	// FullCollectionProfile is the default collection profile that collects all the metrics.
	FullCollectionProfile CollectionProfile = "full"

	// MinimalCollectionProfile is the collection profile that collects only the metrics that are strictly required for
	// the cluster to function.
	MinimalCollectionProfile CollectionProfile = "minimal"
)

var (
	SupportedCollectionProfiles           = CollectionProfiles{FullCollectionProfile, MinimalCollectionProfile}
	SupportedNonDefaultCollectionProfiles = SupportedCollectionProfiles[1:]
)

func IsSupportedCollectionProfile(profile CollectionProfile) bool {
	for _, p := range SupportedCollectionProfiles {
		if p == profile {
			return true
		}
	}

	return false
}
