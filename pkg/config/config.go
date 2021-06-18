package config

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/logutils"
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

	ElasticEndpoint      string `default:"https://tigera-secure-internal-es-http.tigera-elasticsearch.svc:9200" split_words:"true"`
	ElasticCatchAllRoute string `default:"/" split_words:"true"`
	ElasticCABundlePath  string `default:"/certs/elasticsearch/tls.crt" split_words:"true"`
	ElasticUsername      string `default:"" split_words:"true"`
	ElasticPassword      string `default:"" split_words:"true"`

	KibanaEndpoint      string `default:"https://tigera-secure-internal-kb-http.tigera-kibana.svc:5601" split_words:"true"`
	KibanaCatchAllRoute string `default:"/tigera-kibana/" split_words:"true"` // Note: The ending "/" is important for prefix matching.
	KibanaCABundlePath  string `default:"/certs/kibana/tls.crt" split_words:"true"`
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
	log.AddHook(&logutils.ContextHook{})
	log.SetFormatter(&logutils.Formatter{})
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
