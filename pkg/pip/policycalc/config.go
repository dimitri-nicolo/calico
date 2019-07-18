package policycalc

import (
	"github.com/kelseyhightower/envconfig"

	log "github.com/sirupsen/logrus"
)

// Config contain environment based configuration for the policy calculator.
type Config struct {
	// Whether or not the original action should be calculated. If this is false, the Action in the flow data will
	// be unchanged from the original value. If this is true, the Action will be calculated by passing the flow through
	// the current set of policy resources.
	CalculateOriginalAction bool `envconfig:"TIGERA_PIP_CALCULATE_ORIGINAL_ACTION"`
}

// MustLoadConfig loads the configuration from the environment variables. It panics if there is an error in the config.
func MustLoadConfig() *Config {
	c, err := LoadConfig()
	if err != nil {
		log.Panicf("Error loading configuration: %v", err)
	}
	return c
}

// LoadConfig loads the configuration from the environment variables.
func LoadConfig() (*Config, error) {
	var err error
	config := &Config{}
	err = envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
