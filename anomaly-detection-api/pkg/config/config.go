// Copyright (c) 2022 Tigera All rights reserved.
package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// this service will be hosted on this addresss
	ListenAddr string `envconfig:"LISTEN_ADDR" default:":8080"`

	// ServiceEndpoint string `envconfig:"ENDPOINT_URL" default:"http://localhost:8080"`
	// ServiceURL      *url.URL
	HostedNamespace string `envconfig:"NAMESPACE" default:"tigera-intrusion-detection"`
	StoragePath     string `envconfig:"STORAGE_PATH" default:"/store"`

	TLSCert string `envconfig:"TLS_CERT" default:"/tls/tls.crt"`
	TLSKey  string `envconfig:"TLS_KEY" default:"/tls/tls.key"`

	// debug settings
	DebugRunWithRBACDisabled bool   `envconfig:"DEBUG_RBAC_DISABLED" default:"false"`
	LogLevel                 string `envconfig:"LOG_LEVEL" default:"info"`

	// FIPSModeEnabled Enables FIPS 140-2 verified crypto mode.
	FIPSModeEnabled bool `envconfig:"FIPS_MODE_ENABLED" default:"false"`
}

func NewConfigFromEnv() (*Config, error) {
	config := &Config{}

	// Load config from environments.
	err := envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
