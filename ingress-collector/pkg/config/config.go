// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package config

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

type Config struct {
	// LogLevel
	LogLevel string `envconfig:"LOG_LEVEL"`

	// Socket to dial
	DialTarget string `envconfig:"FELIX_DIAL_TARGET"`
	// Location of the ingress log files to read from
	IngressLogPath string `envconfig:"INGRESS_LOG_PATH"`
	// How long the collector will wait in seconds to collect
	// logs before sending them as a batch.
	IngressLogIntervalSecs int `envconfig:"INGRESS_LOG_INTERVAL_SECONDS"`
	// Number requests sent in the batch of logs from the collector.
	// A negative number will return as many requests as possible.
	IngressRequestsPerInterval int `envconfig:"INGRESS_LOG_REQUESTS_PER_INTERVAL"`

	// Configuration for tests
	// Sets where the log file should be read from.
	// Defaulted to 2 (end of the file).
	TailWhence int

	// Parsed values
	ParsedLogLevel log.Level
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

	// Parse log level.
	config.ParsedLogLevel = logutils.SafeParseLogLevel(config.LogLevel)

	// Default the IngressLogPath to /var/log/calico/ingress/ingress.log
	if config.IngressLogPath == "" {
		config.IngressLogPath = "/var/log/calico/ingress/ingress.log"
	}

	// Default the IngressLogInterval to 5 seconds
	if config.IngressLogIntervalSecs == 0 {
		config.IngressLogIntervalSecs = 5
	}

	// Default the INgressLogBatchSize to 10
	if config.IngressRequestsPerInterval == 0 {
		config.IngressRequestsPerInterval = 10
	}

	// Make sure that the tail reads from the end of the ingress log.
	config.TailWhence = 2

	return config, nil
}

func (c *Config) InitializeLogging() {
	logutils.ConfigureFormatter("ingresscol")
	log.SetLevel(c.ParsedLogLevel)
}
