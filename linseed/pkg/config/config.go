// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package config

import (
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

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

	Kubeconfig string `envconfig:"KUBECONFIG"`

	// Certificate presented to the client for TLS verification.
	HTTPSCert string `default:"/certs/https/tls.crt" split_words:"true"`
	HTTPSKey  string `default:"/certs/https/tls.key" split_words:"true"`

	// Metrics endpoint configurations.
	EnableMetrics bool `default:"false" split_words:"true"`
	MetricsPort   int  `default:"9095" split_words:"true"`

	// Certificates used to secure metrics endpoint via TLS
	MetricsCert string `default:"/certs/https/tls.crt" split_words:"true"`
	MetricsKey  string `default:"/certs/https/tls.key" split_words:"true"`

	// Key used for generation and verification of access tokens.
	TokenKey string `default:"/certs/https/tokens.key" split_words:"true"`

	// Used to verify client certificates for mTLS.
	CACert string `default:"/certs/https/client.crt" split_words:"true"`

	// FIPSModeEnabled Enables FIPS 140-2 verified crypto mode.
	FIPSModeEnabled bool `default:"false" split_words:"true"`

	// ExpectedTenantID will be verified against x-tenant-id header for all API calls
	// in a multi-tenant environment
	// If left empty, x-tenant-id header is not required as Linseed will run in a
	// single-tenant environment
	ExpectedTenantID string `default:"" split_words:"true"`

	ManagementOperatorNamespace string `envconfig:"MANAGEMENT_OPERATOR_NS" default:""`

	// Whether or not to run the token controller. This must be true for management clusters.
	TokenControllerEnabled bool `envconfig:"TOKEN_CONTROLLER_ENABLED" default:"false"`

	// Configuration for Voltron access.
	MultiClusterForwardingEndpoint string `default:"https://tigera-manager.tigera-manager.svc:9443" split_words:"true"`
	MultiClusterForwardingCA       string `default:"/etc/pki/tls/certs/tigera-ca-bundle.crt" split_words:"true"`

	// Configuration for health port.
	HealthPort int `default:"8080" split_words:"true"`

	// Elastic configuration
	ElasticScheme               string `envconfig:"ELASTIC_SCHEME" default:"https"`
	ElasticHost                 string `envconfig:"ELASTIC_HOST" default:"tigera-secure-es-http.tigera-elasticsearch.svc"`
	ElasticPort                 string `envconfig:"ELASTIC_PORT" default:"9200"`
	ElasticUsername             string `envconfig:"ELASTIC_USERNAME" default:""`
	ElasticPassword             string `envconfig:"ELASTIC_PASSWORD" default:"" json:",omitempty"`
	ElasticCA                   string `envconfig:"ELASTIC_CA" default:"/certs/elasticsearch/tls.crt"`
	ElasticClientKey            string `envconfig:"ELASTIC_CLIENT_KEY" default:"/certs/elasticsearch/client.key"`
	ElasticClientCert           string `envconfig:"ELASTIC_CLIENT_CERT" default:"/certs/elasticsearch/client.crt"`
	ElasticGZIPEnabled          bool   `envconfig:"ELASTIC_GZIP_ENABLED" default:"false"`
	ElasticMTLSEnabled          bool   `envconfig:"ELASTIC_MTLS_ENABLED" default:"false"`
	ElasticSniffingEnabled      bool   `envconfig:"ELASTIC_SNIFFING_ENABLED" default:"false"`
	ElasticIndexMaxResultWindow int64  `envconfig:"ELASTIC_INDEX_MAX_RESULT_WINDOW" default:"10000"`

	// Default value for replicas and shards
	ElasticReplicas int `envconfig:"ELASTIC_REPLICAS" default:"0"`
	ElasticShards   int `envconfig:"ELASTIC_SHARDS" default:"1"`

	// Replicas and flows for flows
	ElasticFlowReplicas int `envconfig:"ELASTIC_FLOWS_INDEX_REPLICAS" default:"0"`
	ElasticFlowShards   int `envconfig:"ELASTIC_FLOWS_INDEX_SHARDS" default:"1"`

	// Replicas and flows for DNS
	ElasticDNSReplicas int `envconfig:"ELASTIC_DNS_INDEX_REPLICAS" default:"0"`
	ElasticDNSShards   int `envconfig:"ELASTIC_DNS_INDEX_SHARDS" default:"1"`

	// Replicas and flows for Audit
	ElasticAuditReplicas int `envconfig:"ELASTIC_AUDIT_INDEX_REPLICAS" default:"0"`
	ElasticAuditShards   int `envconfig:"ELASTIC_AUDIT_INDEX_SHARDS" default:"1"`

	// Replicas and flows for BGP
	ElasticBGPReplicas int `envconfig:"ELASTIC_BGP_INDEX_REPLICAS" default:"0"`
	ElasticBGPShards   int `envconfig:"ELASTIC_BGP_INDEX_SHARDS" default:"1"`

	// Replicas and flows for WAF
	ElasticWAFReplicas int `envconfig:"ELASTIC_WAF_INDEX_REPLICAS" default:"0"`
	ElasticWAFShards   int `envconfig:"ELASTIC_WAF_INDEX_SHARDS" default:"1"`

	// Replicas and flows for L7
	ElasticL7Replicas int `envconfig:"ELASTIC_L7_INDEX_REPLICAS" default:"0"`
	ElasticL7Shards   int `envconfig:"ELASTIC_L7_INDEX_SHARDS" default:"1"`

	// Replicas and flows for Runtime
	ElasticRuntimeReplicas int `envconfig:"ELASTIC_RUNTIME_INDEX_REPLICAS" default:"0"`
	ElasticRuntimeShards   int `envconfig:"ELASTIC_RUNTIME_INDEX_SHARDS" default:"1"`

	// Configures which backend mode to use.
	Backend BackendType `envconfig:"BACKEND" default:"elastic-multi-index"`

	TenantNamespace string `envconfig:"TENANT_NAMESPACE" default:""`
}

type BackendType string

const (
	// BackendTypeMultiIndex is the legacy backend that stores different cluster and tenant data in separate indices.
	BackendTypeMultiIndex BackendType = "elastic-multi-index"

	// BackendTypeSingleIndex is the backend that stores all cluster and tenant data in a single index.
	BackendTypeSingleIndex BackendType = "elastic-single-index"
)

// Return a string representation on the Config instance.
func (cfg *Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func LoadConfig() (*Config, error) {
	var err error
	config := &Config{}
	if err = envconfig.Process(EnvConfigPrefix, config); err != nil {
		logrus.WithError(err).Fatal("Unable to load envconfig %w", err)
	}

	// Get TenantNamespace in MultiTenant Mode.
	if len(config.ExpectedTenantID) > 0 && config.TenantNamespace == "" {
		ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			logrus.WithError(err).Fatal("unable to get the tenant namespace: %w", err)
		}
		config.TenantNamespace = strings.TrimSpace(string(ns))
	}
	return config, nil
}
