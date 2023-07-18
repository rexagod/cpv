package profiles

import (
	"context"
	"github.com/rexagod/cpv/internal/client"
	"k8s.io/client-go/dynamic"
)

var ProfileOperators = map[CollectionProfile]func(
	ctx context.Context,
	profile CollectionProfile,
	dc *dynamic.DynamicClient,
	c *client.Client,
) error{
	FullCollectionProfile: nil,
	// A minimal collection profile is a collection profile that only collects metrics necessary for:
	//  * alerts,
	//  * dashboards,
	//  * recording rules, and,
	//  * telemetry.
	MinimalCollectionProfile: MinimalCollectionProfileOperator,
}
