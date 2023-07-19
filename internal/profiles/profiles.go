package profiles

import (
	"context"
	"github.com/rexagod/cpv/internal/client"
	"k8s.io/client-go/dynamic"
)

var ProfileOperators = map[CollectionProfile]func(
	context.Context,
	*dynamic.DynamicClient,
	*client.Client,
) error{
	FullCollectionProfile: nil,
	// A minimal collection profile is a collection profile that only collects metrics necessary for:
	//  * alerts,
	//  * dashboards,
	//  * recording rules, and,
	//  * telemetry.
	MinimalCollectionProfile: MinimalCollectionProfileOperator,
}

var ProfileGuessers = map[CollectionProfile]func(
	context.Context,
	*dynamic.DynamicClient,
	*client.Client,
	...interface{},
) error{
	FullCollectionProfile: nil,
	// A minimal collection profile Guesser estimates the metrics needed to implement minimal collection profile from a
	// given set of constraints (targets).
	MinimalCollectionProfile: GuessMinimalProfile,
}
