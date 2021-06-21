// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package config

import (
	"encoding/json"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "PACKETCAPTURE_API"
)

// Config is a configuration used for PacketCapture API
type Config struct {
	Port     int `default:"8444" split_words:"true"`
	Host     string
	LogLevel string `default:"INFO" split_words:"true"`

	// Dex settings for authentication.
	DexEnabled        bool   `default:"false" split_words:"true"`
	DexIssuer         string `default:"https://127.0.0.1:5556/dex" split_words:"true"`
	DexClientID       string `default:"tigera-manager" split_words:"true"`
	DexJwksUrl        string `default:"https://tigera-dex.tigera-dex.svc.cluster.local:5556/dex/keys" split_words:"true"`
	DexUsernameClaim  string `default:"email" split_words:"true"`
	DexGroupsClaim    string `split_words:"true"`
	DexUsernamePrefix string `split_words:"true"`
	DexGroupsPrefix   string `split_words:"true"`

	// Multi-cluster settings
	MultiClusterForwardingCA       string `default:"/manager-tls/cert"`
	MultiClusterForwardingEndpoint string `default:"https://localhost:9443"`
}

// Return a string representation on the Config instance.
func (cfg *Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}
