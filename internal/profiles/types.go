package profiles

type CollectionProfile string
type CollectionProfiles []CollectionProfile

const (
	CollectionProfileOptInLabel = "monitoring.openshift.io/collection-profile"
	FullCollectionProfile       = "full"
	MinimalCollectionProfile    = "minimal"
)

var SupportedCollectionProfiles = CollectionProfiles{FullCollectionProfile, MinimalCollectionProfile}
var SupportedNonDefaultCollectionProfiles = SupportedCollectionProfiles[1:]

func IsSupportedCollectionProfile(profile CollectionProfile) bool {
	for _, p := range SupportedCollectionProfiles {
		if p == profile {
			return true
		}
	}
	return false
}
