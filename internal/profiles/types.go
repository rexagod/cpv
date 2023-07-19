package profiles

type (
	CollectionProfile  string
	CollectionProfiles []CollectionProfile
)

const (
	CollectionProfileOptInLabel                   = "monitoring.openshift.io/collection-profile"
	FullCollectionProfile       CollectionProfile = "full"
	MinimalCollectionProfile    CollectionProfile = "minimal"
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
