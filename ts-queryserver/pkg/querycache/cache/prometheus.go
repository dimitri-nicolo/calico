// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package cache

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

func NewPrometheusClient(address, token string, tlsConfig *tls.Config) *PrometheusClient {
	client, err := api.NewClient(api.Config{
		Address: address,
		RoundTripper: config.NewAuthorizationCredentialsRoundTripper(
			"Bearer", config.Secret(token),
			&http.Transport{TLSClientConfig: tlsConfig},
		),
	})

	if err != nil {
		log.WithError(err).Warn("failed to create prometheus client")
		return nil
	}
	return &PrometheusClient{client: client}
}

type PrometheusClient struct {
	client api.Client
}

func (c *PrometheusClient) Query(query string, ts time.Time) (model.Value, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v1api := v1.NewAPI(c.client)
	res, _, err := v1api.Query(ctx, query, ts, v1.WithTimeout(5*time.Second))
	return res, err
}
