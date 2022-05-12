// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package config

import (
	"encoding/json"
	"time"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "DPI"
)

// Config is a configuration used for PacketCapture API
type Config struct {
	LogLevel                       string        `split_words:"true" default:"INFO"`
	HealthEnabled                  bool          `split_words:"true" default:"true"`
	HealthPort                     int           `split_words:"true" default:"9097"`
	HealthHost                     string        `split_words:"true" default:"0.0.0.0"`
	HealthTimeout                  time.Duration `split_words:"true" default:"30s"`
	SnortAlertFileBasePath         string        `split_words:"true" default:"/var/log/calico/snort-alerts"`
	SnortAlertFileSize             int           `split_words:"true" default:"5"`
	SnortCommunityRulesFile        string        `split_words:"true" default:"/usr/local/etc/rules/snort3-community-rules/snort3-community.rules"`
	ElasticIndexSuffix             string        `split_words:"true" default:"cluster"`
	ElasticScheme                  string        `split_words:"true" default:"https"`
	ElasticHost                    string        `split_words:"true" default:"tigera-secure-es-gateway-http.tigera-elasticsearch.svc"`
	ElasticPort                    string        `split_words:"true" default:"9200"`
	ElasticCA                      string        `split_words:"true"`
	ElasticUsername                string        `split_words:"true"`
	ElasticPassword                string        `split_words:"true"`
	ElasticEnableTrace             bool          `split_words:"true" default:"false"`
	ElasticConnectionRetries       int           `split_words:"true" default:"30"`
	ElasticConnectionRetryInterval time.Duration `split_words:"true" default:"500ms"`

	// For Debugging
	ElasticInsecureSkipVerify bool `split_words:"true" default:"false"`
	ElasticDebug              bool `split_words:"true" default:"false"`

	// All the below config variables are used by typha to establish connection and they should not use split_words
	NodeName            string
	TyphaK8sNamespace   string
	TyphaK8sServiceName string

	// Client-side TLS config for tigera-dpi's communication with Typha.  If any of these are
	// specified, they _all_ must be - except that either TyphaCN or TyphaURISAN may be left
	// unset.  Tigera-dpi will then initiate a secure (TLS) connection to Typha.  Typha must present
	// a certificate signed by a CA in TyphaCAFile, and with CN matching TyphaCN or URI SAN
	// matching TyphaURISAN.
	TyphaKeyFile  string
	TyphaCertFile string
	TyphaCAFile   string
	TyphaCN       string
	TyphaURISAN   string
}

// Return a string representation on the Config instance.
func (cfg *Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
}
