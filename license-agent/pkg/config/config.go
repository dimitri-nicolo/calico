// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	MetricsPort        int           `envconfig:"LICENSE_METRICS_PORT"      default:"9081"`
	MetricPollInterval time.Duration `envconfig:"LICENSE_POLL_INTERVAL"     default:"2m"`
	MetricsCertFile    string        `envconfig:"LICENSE_METRICS_CERT_FILE" default:""`
	MetricsKeyFile     string        `envconfig:"LICENSE_METRICS_KEY_FILE"  default:""`
	MetricsCaFile      string        `envconfig:"LICENSE_METRICS_CA_FILE"   default:""`
	MetricsHost        string        `envconfig:"LICENSE_METRICS_HOST"      default:""`
}

func MustLoadConfig() *Config {
	c, err := LoadConfig()
	if err != nil {
		log.Panicf("Error loading configuration: %v", err)
	}
	return c
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
