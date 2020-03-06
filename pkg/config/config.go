package config

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MetricsPort    int `envconfig:"LICENSE_METRICS_PORT" default:"9200"`
	MetricPollTime int `envconfig:"LICENSE_POLL_MINUTES" default:"2"`
}

func MustLoadConfig() *Config {
	c, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v", err)
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
