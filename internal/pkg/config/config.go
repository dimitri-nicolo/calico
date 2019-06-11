// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package config

type Config struct {
	Port     int    `default:"5555"`
	Host     string `default:"localhost"`
	LogLevel string `default:"DEBUG"`
	CertPath string `default:"certs"`
}
