// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package server

import (
	"net/url"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// this service will be hosted on this addresss
	ListenAddr string `envconfig:"LISTEN_ADDR" default:":9090"`

	PrometheusEndpoint string   `envconfig:"PROMETHEUS_ENDPOINT_URL" default:"http://localhost:9090"`
	PrometheusUrl      *url.URL `envconfig:"-"`

	TLSCert string `envconfig:"TLS_CERT" default:"/tls/tls.crt"`
	TLSKey  string `envconfig:"TLS_KEY" default:"/tls/tls.key"`

	// Meant for fv only.
	AuthenticationEnabled bool `default:"true" split_words:"true"`

	// Dex settings
	DexEnabled bool `default:"false" split_words:"true"`

	// OIDC Authentication settings.
	OIDCAuthJWKSURL        string `default:"https://tigera-dex.tigera-dex.svc.cluster.local:5556/dex/keys" split_words:"true"`
	OIDCAuthIssuer         string `default:"https://127.0.0.1:5556/dex" split_words:"true"`
	OIDCAuthClientID       string `default:"tigera-manager" split_words:"true"`
	OIDCAuthUsernameClaim  string `default:"email" split_words:"true"`
	OIDCAuthUsernamePrefix string `split_words:"true"`
	OIDCAuthGroupsClaim    string `default:"groups" split_words:"true"`
	OIDCAuthGroupsPrefix   string `split_words:"true"`

	// FIPSModeEnabled Enables FIPS 140-2 verified crypto mode.
	FIPSModeEnabled bool `default:"false" split_words:"true"`
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
