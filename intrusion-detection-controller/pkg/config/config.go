package config

import (
	"fmt"
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
	LinseedToken      string `envconfig:"LINSEED_TOKEN" default:"/var/run/secrets/kubernetes.io/serviceaccount/token"`

	// TenantID configuration for Calico Cloud.
	TenantID string `envconfig:"TENANT_ID"`

	// MCM configuration
	MultiClusterForwardingCA       string `envconfig:"MULTI_CLUSTER_FORWARDING_CA" default:"/manager-tls/cert"`
	MultiClusterForwardingEndpoint string `envconfig:"MULTI_CLUSTER_FORWARDING_ENDPOINT" default:"https://tigera-manager.tigera-manager.svc:9443"`

	// MultiTenant configuration
	TenantNamespace string `envconfig:"TENANT_NAMESPACE" default:""`
}

func GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}
	if cfg.TenantNamespace != "" && cfg.TenantID == "" {
		return nil, fmt.Errorf("Tenant namespace was provided but TenantID was not")
	}
	return cfg, nil
}
