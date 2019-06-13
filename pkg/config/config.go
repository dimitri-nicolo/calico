// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package config

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/logutils"
)

type Config struct {
	// LogLevel
	LogLevel string `envconfig:"LOG_LEVEL"`

	// Socket to dial
	DialTarget string `envconfig:"FELIX_DIAL_TARGET"`

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

	return config, nil
}

func (c *Config) InitializeLogging() {
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetLevel(c.ParsedLogLevel)
}
