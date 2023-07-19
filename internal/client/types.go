package client

import (
	"context"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type Client struct {
	ctx         context.Context
	address     string
	bearerToken string
	v1.API
}
