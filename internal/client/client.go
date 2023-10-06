// Package client contains the client for Prometheus.
package client

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/promql/parser"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

type Client struct {
	ctx         context.Context
	address     string
	bearerToken string
	v1.API
}

func NewClient(ctx context.Context, address, bearerToken string) *Client {
	return &Client{
		ctx:         ctx,
		address:     address,
		bearerToken: bearerToken,
	}
}

func (c *Client) Init() error {
	client, err := api.NewClient(api.Config{
		Address:      c.address,
		RoundTripper: config.NewAuthorizationCredentialsRoundTripper("Bearer", config.Secret(c.bearerToken), api.DefaultRoundTripper),
	})
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}
	c.API = v1.NewAPI(client)

	return nil
}

func (c *Client) Query(query string) (model.Value, v1.Warnings, error) {

	//nolint:wrapcheck
	return c.API.Query(c.ctx, query, time.Now())
}

func (c *Client) getCardinalityForMetric(metric *parser.VectorSelector) uint {
	query := fmt.Sprintf("count(%s)", metric)
	r, _, err := c.Query(query)
	if err != nil {
		return uint(0)
	}
	v, ok := r.(model.Vector)
	if !ok {
		return uint(0)
	}
	cardinality := uint(0)
	for _, sample := range v {
		cardinality += uint(sample.Value)
	}

	return cardinality
}

type CardinalValue struct {
	Metric string
	Value  uint
}

func (c *Client) EvaluateCardinalities(ctx context.Context, metricSet *sets.Set[string]) []CardinalValue {
	cardinalChan := make(chan CardinalValue, len(*metricSet))
	var cardinalities []CardinalValue
	var wg sync.WaitGroup
	wg.Add(len(*metricSet))
	for m := range *metricSet {
		go func(m string) {
			if ctx.Err() != nil {
				klog.Warningf("context cancelled, skipping metric %s", m)

				return
			}
			cardinalChan <- CardinalValue{
				Metric: m,
				Value:  c.getCardinalityForMetric(&parser.VectorSelector{Name: m}),
			}
			wg.Done()
		}(m)
	}
	wg.Wait()
	close(cardinalChan)

	for c := range cardinalChan {
		cardinalities = append(cardinalities, c)
	}
	sort.Slice(cardinalities, func(i, j int) bool {
		return cardinalities[i].Value > cardinalities[j].Value
	})

	return cardinalities
}
