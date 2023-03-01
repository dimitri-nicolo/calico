// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package config

import (
	"github.com/projectcalico/calico/libcalico-go/lib/json"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "LINSEED"
)

// Config defines the parameters of the application
type Config struct {
	Port     int `default:"8444" split_words:"true"`
	Host     string
	LogLevel string `default:"INFO" split_words:"true"`

	// Certificate presented to the client for TLS verification.
	HTTPSCert string `default:"/certs/https/tls.crt" split_words:"true"`
	HTTPSKey  string `default:"/certs/https/tls.key" split_words:"true"`

	// Used to verify client certificates for mTLS.
	CACert string `default:"/certs/https/client.crt" split_words:"true"`

	// FIPSModeEnabled Enables FIPS 140-2 verified crypto mode.
	FIPSModeEnabled bool `default:"false" split_words:"true"`

	// ExpectedTenantID will be verified against x-tenant-id header for all API calls
	// in a multi-tenant environment
	// If left empty, x-tenant-id header is not required as Linseed will run in a
	// single-tenant environment
	ExpectedTenantID string `default:"" split_words:"true"`

	// Elastic configuration
	ElasticEndpoint        string `default:"https://tigera-secure-es-http.tigera-elasticsearch.svc:9200" split_words:"true"`
	ElasticUsername        string `default:"" split_words:"true"`
	ElasticPassword        string `default:"" split_words:"true" json:",omitempty"`
	ElasticCABundlePath    string `default:"/certs/elasticsearch/tls.crt" split_words:"true"`
	ElasticClientKeyPath   string `default:"/certs/elasticsearch/client.key" split_words:"true"`
	ElasticClientCertPath  string `default:"/certs/elasticsearch/client.crt" split_words:"true"`
	ElasticGZIPEnabled     bool   `default:"false" split_words:"true"`
	ElasticMTLSEnabled     bool   `default:"false" split_words:"true"`
	ElasticScheme          string `default:"https" split_words:"true"`
	ElasticSniffingEnabled bool   `default:"false" split_words:"true"`
	ElasticReplicas        int    `envconfig:"ELASTIC_REPLICAS" default:"0"`
	ElasticShards          int    `envconfig:"ELASTIC_SHARDS" default:"1"`
}

// Return a string representation on the Config instance.
func (cfg *Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}
