// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package config contains the structures that define an application configuration
package config

// Config is a configuration used by Guardian
type Config struct {
	Port     int    `default:"5555"`
	Host     string `default:"localhost"`
	LogLevel string `default:"DEBUG"`
	CertPath string `default:"certs"`
}
