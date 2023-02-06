// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	defaultCertFileName = "cert"
	defaultKeyFileName  = "key"
)

// Config stores various configuration information for the es-proxy
// server.
type Config struct {
	// ListenAddr is the address and port that the server will listen
	// on for proxying requests. The format is similar to the address
	// parameter of net.Listen
	ListenAddr string `envconfig:"LISTEN_ADDR" default:"127.0.0.1:8443"`

	// Paths to files containing certificate and matching private key
	// for serving requests over TLS.
	CertFile string `envconfig:"CERT_FILE_PATH"`
	KeyFile  string `envconfig:"KEY_FILE_PATH"`

	// If specific a CertFile and KeyFile are not provided this is the
	// location to autogenerate the files
	DefaultSSLPath string `envconfig:"KEY_CERT_GEN_PATH" default:"/etc/es-proxy/ssl/"`
	// Default cert and key file paths calculated from the DefaultSSLPath
	DefaultCertFile string `envconfig:"-"`
	DefaultKeyFile  string `envconfig:"-"`

	// The URL that we should proxy requests to.
	ElasticScheme             string   `envconfig:"ELASTIC_SCHEME" default:"https"`
	ElasticHost               string   `envconfig:"ELASTIC_HOST"`
	ElasticPort               int      `envconfig:"ELASTIC_PORT"`
	ElasticURL                *url.URL `envconfig:"-"`
	ElasticCAPath             string   `envconfig:"ELASTIC_CA"`
	ElasticInsecureSkipVerify bool     `envconfig:"ELASTIC_INSECURE_SKIP_VERIFY" default:"false"`

	// The username and password to inject.
	ElasticUsername string `envconfig:"ELASTIC_USERNAME"`
	ElasticPassword string `envconfig:"ELASTIC_PASSWORD"`

	ElasticIndexSuffix       string        `envconfig:"ELASTIC_INDEX_SUFFIX" default:"cluster"`
	ElasticConnRetries       int           `envconfig:"ELASTIC_CONN_RETRIES" default:"30"`
	ElasticConnRetryInterval time.Duration `envconfig:"ELASTIC_CONN_RETRY_INTERVAL" default:"500ms"`
	ElasticEnableTrace       bool          `envconfig:"ELASTIC_ENABLE_TRACE" default:"false"`
	ElasticLicenseType       string        `envconfig:"ELASTIC_LICENSE_TYPE"`
	ElasticKibanaEndpoint    string        `envconfig:"ELASTIC_KIBANA_ENDPOINT" default:"https://tigera-secure-kb-http.tigera-kibana.svc:5601"`

	// Various proxy timeouts. Used when creating a http.Transport RoundTripper.
	ProxyConnectTimeout  time.Duration `envconfig:"PROXY_CONNECT_TIMEOUT" default:"30s"`
	ProxyKeepAlivePeriod time.Duration `envconfig:"PROXY_KEEPALIVE_PERIOD" default:"30s"`
	ProxyIdleConnTimeout time.Duration `envconfig:"PROXY_IDLECONN_TIMEOUT" default:"90s"`

	// If multi-cluster management is used inside the cluster, this CA
	// is necessary for establishing a connection with Voltron, when
	// accessing other clusters.
	VoltronCAPath string `envconfig:"VOLTRON_CA_PATH" default:"/manager-tls/cert"`

	// Location of the Voltron service.
	VoltronURL string `envconfig:"VOLTRON_URL" default:"https://localhost:9443"`

	// Dex settings for authentication.
	OIDCAuthEnabled        bool   `envconfig:"OIDC_AUTH_ENABLED" default:"false"`
	OIDCAuthIssuer         string `envconfig:"OIDC_AUTH_ISSUER"`
	OIDCAuthClientID       string `envconfig:"OIDC_AUTH_CLIENT_ID"`
	OIDCAuthJWKSURL        string `envconfig:"OIDC_AUTH_JWKSURL" default:"https://tigera-dex.tigera-dex.svc.cluster.local:5556/dex/keys"`
	OIDCAuthUsernameClaim  string `envconfig:"OIDC_AUTH_USERNAME_CLAIM" default:"email"`
	OIDCAuthGroupsClaim    string `envconfig:"OIDC_AUTH_GROUPS_CLAIM"`
	OIDCAuthUsernamePrefix string `envconfig:"OIDC_AUTH_USERNAME_PREFIX"`
	OIDCAuthGroupsPrefix   string `envconfig:"OIDC_AUTH_GROUPS_PREFIX"`

	// Service graph settings.  See servicegraph.Config for details.
	ServiceGraphCacheMaxEntries           int           `envconfig:"SERVICE_GRAPH_CACHE_MAX_ENTRIES" default:"10"`
	ServiceGraphCacheMaxBucketsPerQuery   int           `envconfig:"SERVICE_GRAPH_CACHE_MAX_BUCKETS_PER_QUERY" default:"1000"`
	ServiceGraphCacheMaxAggregatedRecords int           `envconfig:"SERVICE_GRAPH_CACHE_MAX_AGGREGATED_RECORDS" default:"100000"`
	ServiceGraphCachePolledEntryAgeOut    time.Duration `envconfig:"SERVICE_GRAPH_CACHE_POLLED_ENTRY_AGE_OUT" default:"1h"`
	ServiceGraphCacheSlowQueryEntryAgeOut time.Duration `envconfig:"SERVICE_GRAPH_CACHE_SLOW_QUERY_ENTRY_AGE_OUT" default:"5m"`
	ServiceGraphCachePollLoopInterval     time.Duration `envconfig:"SERVICE_GRAPH_CACHE_POLL_LOOP_INTERVAL" default:"2m"`
	ServiceGraphCachePollQueryInterval    time.Duration `envconfig:"SERVICE_GRAPH_CACHE_POLL_QUERY_INTERVAL" default:"2s"`
	ServiceGraphCacheDataSettleTime       time.Duration `envconfig:"SERVICE_GRAPH_CACHE_DATA_SETTLE_TIME" default:"15m"`

	// FIPSModeEnabled Enables FIPS 140-2 verified crypto mode.
	FIPSModeEnabled bool `envconfig:"FIPS_MODE_ENABLED" default:"false"`
}

func NewConfigFromEnv() (*Config, error) {
	config := &Config{}

	// Load config from environments.
	err := envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	// Calculate the elastic URl from other config values.
	config.ElasticURL = &url.URL{
		Scheme: config.ElasticScheme,
		Host:   fmt.Sprintf("%s:%d", config.ElasticHost, config.ElasticPort),
	}

	// Calculate the default cert and key file from the directory.
	config.DefaultKeyFile = config.DefaultSSLPath + defaultKeyFileName
	config.DefaultCertFile = config.DefaultSSLPath + defaultCertFileName

	err = validateConfig(config)
	return config, err
}

func validateConfig(config *Config) error {
	if config.ElasticURL.Scheme == "" || config.ElasticURL.Host == "" {
		return errors.New("Invalid Elasticsearch backend URL specified")
	}
	if config.ElasticUsername == "" || config.ElasticPassword == "" {
		return errors.New("Elasticsearch credentials not provided")
	}
	if config.ElasticURL.Scheme == "https" && config.ElasticCAPath == "" {
		return errors.New("Elasticsearch CA not provided")
	}
	if config.ElasticURL.Scheme == "http" && config.ElasticCAPath != "" {
		return errors.New("Elasticsearch CA provided but scheme is set to http")
	}
	return nil
}
