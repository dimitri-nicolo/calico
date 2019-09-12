package elastic

import (
	"fmt"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/logutils"
)

// Config contain environment based configuration for all compliance components. Although not all configuration is
// required for all components, it is useful having everything defined in one location.
type Config struct {
	// LogLevel
	LogLevel string `envconfig:"LOG_LEVEL"`

	// Elastic parameters
	ElasticURI               string        `envconfig:"ELASTIC_URI"`
	ElasticScheme            string        `envconfig:"ELASTIC_SCHEME" default:"http"`
	ElasticHost              string        `envconfig:"ELASTIC_HOST" default:"elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local"`
	ElasticPort              int           `envconfig:"ELASTIC_PORT" default:"9200"`
	ElasticUser              string        `envconfig:"ELASTIC_USER" default:"elastic"`
	ElasticPassword          string        `envconfig:"ELASTIC_PASSWORD"`
	ElasticCA                string        `envconfig:"ELASTIC_CA"`
	ElasticIndexSuffix       string        `envconfig:"ELASTIC_INDEX_SUFFIX" default:"cluster"`
	ElasticConnRetries       int           `envconfig:"ELASTIC_CONN_RETRIES" default:"5"`
	ElasticConnRetryInterval time.Duration `envconfig:"ELASTIC_CONN_RETRY_INTERVAL" default:"500ms"`

	// Parsed values.
	ParsedElasticURL *url.URL
	ParsedLogLevel   log.Level
}

func MustLoadConfig() *Config {
	c, err := LoadConfig()
	if err != nil {
		log.Panicf("Error loading configuration: %v", err)
	}
	return c
}

func LoadConfig() (*Config, error) {
	var err error
	config := &Config{}
	err = envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	// Parse elastic parms
	if config.ElasticURI != "" {
		config.ParsedElasticURL, err = url.Parse(config.ElasticURI)
		if err != nil {
			return nil, err
		}
	} else {
		config.ParsedElasticURL = &url.URL{
			Scheme: config.ElasticScheme,
			Host:   fmt.Sprintf("%s:%d", config.ElasticHost, config.ElasticPort),
		}
	}
	log.WithField("url", config.ParsedElasticURL).Debug("Parsed elastic url")

	// Parse log level.
	config.ParsedLogLevel = logutils.SafeParseLogLevel(config.LogLevel)

	return config, nil
}
