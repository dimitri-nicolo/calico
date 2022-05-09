// Copyright (c) 2022 Tigera All rights reserved.
package config

import (
	"net/url"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// this service will be hosted on this addresss
	ListenAddr string `envconfig:"LISTEN_ADDR" default:":8080"`

	ServiceEndpoint string `envconfig:"ENDPOINT_URL" default:"http://localhost:8080"`
	ServiceURL      *url.URL

	StoragePath string `envconfig:"STORAGE_PATH" default:"/store"`

	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	TLSCert string `envconfig:"TLS_CERT" default:"/tls/tls.crt"`
	TLSKey  string `envconfig:"TLS_KEY" default:"/tls/tls.key"`

	// fv setting
	DebugRunWithRBACDisabled bool `envconfig:"DEBUG_RBAC_DISABLED" default:"false"`
}

func NewConfigFromEnv() (*Config, error) {
	config := &Config{}

	// Load config from environments.
	err := envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	// loads the url to host the API on from ServiceEndpoint
	config.ServiceURL, err = url.Parse(config.ServiceEndpoint)

	if err != nil {
		return nil, err
	}

	return config, nil
}
