package client

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"time"
)

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
	return c.API.Query(c.ctx, query, time.Now())
}
