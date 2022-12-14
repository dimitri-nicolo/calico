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
	Port      int `default:"8444" split_words:"true"`
	Host      string
	HTTPSCert string `default:"/certs/https/tls.crt" split_words:"true"`
	HTTPSKey  string `default:"/certs/https/tls.key" split_words:"true"`
	LogLevel  string `default:"INFO" split_words:"true"`

	// Dex settings
	DexEnabled bool `default:"false" split_words:"true"`

	// OIDC Authentication settings.
	OIDCAuthJWKSURL        string `default:"https://tigera-dex.tigera-dex.svc.cluster.local:5556/dex/keys" split_words:"true"`
	OIDCAuthIssuer         string `default:"https://127.0.0.1:5556/dex" split_words:"true"`
	OIDCAuthClientID       string `default:"tigera-manager" split_words:"true"`
	OIDCAuthUsernameClaim  string `default:"email" split_words:"true"`
	OIDCAuthUsernamePrefix string `split_words:"true"`
	OIDCAuthGroupsClaim    string `default:"groups" split_words:"true"`
	OIDCAuthGroupsPrefix   string `split_words:"true"`

	CalicoCloudRequireTenantClaim bool   `envconfig:"CALICO_CLOUD_REQUIRE_TENANT_CLAIM" default:"false"`
	CalicoCloudTenantClaim        string `envconfig:"CALICO_CLOUD_TENANT_CLAIM"`

	// Multi-cluster settings
	MultiClusterForwardingCA       string `default:"/manager-tls/cert"`
	MultiClusterForwardingEndpoint string `default:"https://localhost:9443"`

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
