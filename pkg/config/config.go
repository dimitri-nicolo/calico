// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package config

import (
	"encoding/json"
	"time"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "DPI"
)

// Config is a configuration used for PacketCapture API
type Config struct {
	LogLevel                string        `split_words:"true" default:"INFO"`
	HealthEnabled           bool          `split_words:"true" default:"true"`
	HealthPort              int           `split_words:"true" default:"9097"`
	HealthHost              string        `split_words:"true" default:"0.0.0.0"`
	HealthTimeout           time.Duration `split_words:"true" default:"30s"`
	SnortAlertFileBasePath  string        `split_words:"true" default:"/var/log/snort-alerts"`
	SnortAlertFileSize      int           `split_words:"true" default:"5"`
	SnortCommunityRulesFile string        `split_words:"true" default:"/usr/local/etc/rules/snort3-community-rules/snort3-community.rules"`
}

// Return a string representation on the Config instance.
func (cfg *Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}
