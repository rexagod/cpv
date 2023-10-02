package profiles

import (
	"context"
	"github.com/rexagod/cpv/internal/client"
	"k8s.io/client-go/dynamic"
)

// operator is an interface that defines the Operator method, which must be implemented by all profile operators.
type operator interface {
	Operator(
		context.Context,
		*dynamic.DynamicClient,
		*client.Client,
	) error
}

// ProfileOperators is a map of all the profile operators.
var ProfileOperators = map[CollectionProfile]operator{
	FullCollectionProfile: nil,
	// A minimal collection profile is a collection profile that only collects metrics necessary for:
	//  * alerts,
	//  * dashboards,
	//  * recording rules, and,
	//  * telemetry.
	MinimalCollectionProfile: &minimalProfileOperator{},
}

// guesser is an interface that defines the Guess method, which must be implemented by all profile guessers.
type guesser interface {
	Guess(
		context.Context,
		*dynamic.DynamicClient,
		*client.Client,
		...interface{},
	) error
}

// ProfileGuessers is a map of all the profile guessers.
var ProfileGuessers = map[CollectionProfile]guesser{
	FullCollectionProfile: nil,
	// A minimal collection profile Guesser estimates the metrics needed to implement minimal collection profile from a
	// given set of constraints (targets).
	MinimalCollectionProfile: &minimalProfileGuesser{},
}
