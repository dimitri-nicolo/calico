// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package server

import (
	"net/url"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// this service will be hosted on this addresss
	ListenAddr string `envconfig:"LISTEN_ADDR" default:"127.0.0.1:9090"`

	PrometheusEndpoint string   `envconfig:"PROMETHEUS_ENDPOINT_URL" default:"http://localhost:9090"`
	PrometheusUrl      *url.URL `envconfig:"-"`
}

func NewConfigFromEnv() (*Config, error) {
	config := &Config{}

	// Load config from environments.
	err := envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	// Calculate the prometheus URl from other config values.
	config.PrometheusUrl, err = url.Parse(config.PrometheusEndpoint)

	if err != nil {
		return nil, err
	}

	return config, nil
}
