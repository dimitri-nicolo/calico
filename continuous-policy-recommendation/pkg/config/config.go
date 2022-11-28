// Copyright (c) 2022 Tigera Inc. All rights reserved.

package config

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

type Config struct {
	LogLevel log.Level `envconfig:"LOG_LEVEL" default:"INFO"`

	// This setting is required for es proxy that performs the authentication and authorization for a user.
	EnableMultiClusterClient       bool   `envconfig:"ENABLE_MULTI_CLUSTER_CLIENT" default:"false"`
	MultiClusterForwardingCA       string `envconfig:"MULTI_CLUSTER_FORWARDING_CA" default:"/manager-tls/cert"`
	MultiClusterForwardingEndpoint string `envconfig:"MULTI_CLUSTER_FORWARDING_ENDPOINT" default:"https://tigera-manager.tigera-manager.svc:9443"`
}

func LoadConfig() (*Config, error) {
	var err error
	config := &Config{}
	err = envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) InitializeLogging() {
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetLevel(c.LogLevel)
}
