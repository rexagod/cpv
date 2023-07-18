package profiles

type CollectionProfile string
type CollectionProfiles []CollectionProfile

const (
	FullCollectionProfile    = "full"
	MinimalCollectionProfile = "minimal"
)

var SupportedCollectionProfiles = CollectionProfiles{FullCollectionProfile, MinimalCollectionProfile}
var SupportedNonDefaultCollectionProfiles = SupportedCollectionProfiles[1:]
