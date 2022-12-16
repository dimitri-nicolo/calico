// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package config

import (
	"encoding/json"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "LINSEED"
)

// Config defines the parameters of the application
type Config struct {
	Port      int `default:"443" split_words:"true"`
	Host      string
	HTTPSCert string `default:"/certs/https/tls.crt" split_words:"true"`
	HTTPSKey  string `default:"/certs/https/tls.key" split_words:"true"`
	LogLevel  string `default:"INFO" split_words:"true"`

	// FIPSModeEnabled Enables FIPS 140-2 verified crypto mode.
	FIPSModeEnabled bool `default:"false" split_words:"true"`
}

// Return a string representation on the Config instance.
func (cfg *Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}
