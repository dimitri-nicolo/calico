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

	FIPSMode bool `envconfig:"FIPS_MODE" default:"false"`
}

func GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
