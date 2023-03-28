package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	KibanaScheme  string `envconfig:"KIBANA_SCHEME" default:"https"`
	KibanaHost    string `envconfig:"KIBANA_HOST"`
	KibanaPort    string `envconfig:"KIBANA_PORT" default:"5601"`
	KibanaCAPath  string `envconfig:"KB_CA_CERT" default:"/etc/pki/tls/certs/tigera-ca-bundle.crt"`
	KibanaSpaceID string `envconfig:"KIBANA_SPACE_ID"`

	ElasticUsername string `envconfig:"ELASTIC_USER"`
	ElasticPassword string `envconfig:"ELASTIC_PASSWORD"`

	FIPSMode bool `envconfig:"FIPS_MODE_ENABLED" default:"false"`

	// Linseed configuration
	LinseedURL        string `envconfig:"LINSEED_URL" default:"https://tigera-linseed.tigera-elasticsearch.svc"`
	LinseedCA         string `envconfig:"LINSEED_CA" default:"/etc/pki/tls/certs/tigera-ca-bundle.crt"`
	LinseedClientCert string `envconfig:"LINSEED_CLIENT_CERT" default:"/etc/pki/tls/certs/tigera-ca-bundle.crt"`
	LinseedClientKey  string `envconfig:"LINSEED_CLIENT_KEY"`

	// Tenant configuration for Calico Cloud.
	// Note the environment variable doesn't match the field name. This is for backwards compat
	// with the cloud operator, and will be udpated in the future.
	Tenant string `envconfig:"ELASTIC_INDEX_MIDFIX"`

	// MCM configuration
	ClusterName              string `envconfig:"CLUSTER_NAME" default:"cluster"`
	MultiClusterForwardingCA string `envconfig:"MULTI_CLUSTER_FORWARDING_CA" default:"/manager-tls/cert"`
}

func GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
