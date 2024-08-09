// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package config

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "ES_GATEWAY"
)

// Config is a configuration used for ES Gateway
type Config struct {
	Port     int `default:"5554" split_words:"true"`
	Host     string
	LogLevel string `default:"INFO" split_words:"true"`

	// HTTPSCert, HTTPSKey - path to an x509 certificate and its private key used
	// for internal communication (between ES Gateway and any in-cluster component,
	// including requests from Voltron tunnel)
	HTTPSCert string `default:"/certs/https/cert" split_words:"true" json:"-"`
	HTTPSKey  string `default:"/certs/https/key" split_words:"true" json:"-"`

	K8sConfigPath string `split_words:"true"`

	// ES Gateway requires Elastic credentials for an ES user with real permissions in order
	// to call both the Elasticsearch and Kibana APIs. Since it needs access, before any other
	// component, the credentials should be for the ES admin user.
	ElasticUsername string `default:"" split_words:"true"`
	ElasticPassword string `default:"" split_words:"true" json:",omitempty"`

	ElasticEndpoint        string `default:"https://tigera-secure-internal-es-http.tigera-elasticsearch.svc:9200" split_words:"true"`
	ElasticCatchAllRoute   string `default:"/" split_words:"true"`
	ElasticCABundlePath    string `default:"/certs/elasticsearch/tls.crt" split_words:"true"`
	ElasticClientKeyPath   string `default:"/certs/elasticsearch/client.key" split_words:"true"`
	ElasticClientCertPath  string `default:"/certs/elasticsearch/client.crt" split_words:"true"`
	EnableElasticMutualTLS bool   `default:"false" split_words:"true"`

	KibanaEndpoint        string `default:"https://tigera-secure-internal-kb-http.tigera-kibana.svc:5601" split_words:"true"`
	KibanaCatchAllRoute   string `default:"/tigera-kibana/" split_words:"true"` // Note: The ending "/" is important for prefix matching.
	KibanaCABundlePath    string `default:"/certs/kibana/tls.crt" split_words:"true"`
	KibanaClientKeyPath   string `default:"/certs/kibana/client.key" split_words:"true"`
	KibanaClientCertPath  string `default:"/certs/kibana/client.crt" split_words:"true"`
	EnableKibanaMutualTLS bool   `default:"false" split_words:"true"`
	ChallengerPort        int    `default:"8080" split_words:"true"`
	TenantID              string `envconfig:"TENANT_ID"`

	// When enabled, any ILM endpoint PUTs or POSTs will be ignored and return success
	ILMDummyRouteEnabled bool `default:"false" split_words:"true"`

	// Prometheus metrics are exposed on this port.
	MetricsEnabled bool `default:"false" split_words:"true"`
	MetricsPort    int  `default:"9091" split_words:"true"`
}

// Return a string representation on the Config instance.
func (cfg *Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// SetupLogging configures the logging framework. The logging level that will
// be used defined by cfg.LogLevel. Otherwise, it will default to WARN.
// The output will be set to STDOUT and the format is TextFormat
func (cfg *Config) SetupLogging() {
	// Install a hook that adds file/line number information.
	logutils.ConfigureFormatter("es-gateway")
	log.SetOutput(os.Stdout)

	// Override with desired log level
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Error("Invalid logging level passed in. Will use default level set to WARN")
		// Setting default to WARN
		level = log.WarnLevel
	}

	log.SetLevel(level)
}
