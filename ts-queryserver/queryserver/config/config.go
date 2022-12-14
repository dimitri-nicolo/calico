// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package config

type Config struct {
	// this service will be hosted on this address
	ListenAddr string `envconfig:"LISTEN_ADDR" default:":8080"`

	TLSCert string `envconfig:"TLS_CERT" default:"/tigera-apiserver-certs/tls.crt"`
	TLSKey  string `envconfig:"TLS_KEY" default:"/tigera-apiserver-certs/tls.key"`

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

	// FIPSModeEnabled uses images and features only that are using FIPS 140-2 validated cryptographic modules and standards.
	FIPSModeEnabled bool `default:"false" split_words:"true"`
}
