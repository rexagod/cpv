// Package profiles contains the collection profiles supported by the metrics-anomaly-detector.
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
		bool,
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

// extractor is an interface that defines the Extract method, which must be implemented by all profile extractors.
type extractor interface {
	Extract(
		context.Context,
		*client.Client,
		...interface{},
	) error
}

// ProfileExtractors is a map of all the profile extractor.
var ProfileExtractors = map[CollectionProfile]extractor{
	FullCollectionProfile: nil,

	// A minimal collection profile Extractor estimates the metrics needed to implement minimal collection profile from a
	// given set of constraints (targets).
	MinimalCollectionProfile: &minimalProfileExtractor{},
}
