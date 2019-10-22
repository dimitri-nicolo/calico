// Copyright 2019 Tigera Inc. All rights reserved.
package main

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// Credentials
	AWSRegion          string `envconfig:"AWS_REGION"`
	AWSAccessKeyId     string `envconfig:"AWS_ACCESS_KEY_ID"`
	AWSSecretAccessKey string `envconfig:"AWS_SECRET_ACCESS_KEY"`

	// Cloudwatch config
	EKSCloudwatchLogGroup        string `envconfig:"EKS_CLOUDWATCH_LOG_GROUP"`
	EKSCloudwatchLogStreamPrefix string `envconfig:"EKS_CLOUDWATCH_LOG_STREAM_PREFIX" default:"kube-apiserver-audit-"`
	EKSStateFileDir              string `envconfig:"EKS_CLOUDWATCH_STATE_FILE_PFX" default:"/fluentd/cloudwatch-logs/"`

	// Elastic Config
	ESURI               string        `envconfig:"ELASTIC_URI"`
	ESScheme            string        `envconfig:"ELASTIC_SCHEME" default:"https"`
	ESHost              string        `envconfig:"ELASTIC_HOST" default:"tigera-secure-es-http.tigera-elasticsearch.svc"`
	ESPort              int           `envconfig:"ELASTIC_PORT" default:"9200"`
	ESUser              string        `envconfig:"ELASTIC_USER" default:"elastic"`
	ESPassword          string        `envconfig:"ELASTIC_PASSWORD"`
	ESSSLVerify         bool          `envconfig:"ELASTIC_SSL_VERIFY" default:"true"`
	ESCA                string        `envconfig:"ELASTIC_CA"`
	ESIndexSuffix       string        `envconfig:"ELASTIC_INDEX_SUFFIX" default:"cluster"`
	ESConnRetries       int           `envconfig:"ELASTIC_CONN_RETRIES" default:"30"`
	ESConnRetryInterval time.Duration `envconfig:"ELASTIC_CONN_RETRY_INTERVAL" default:"500ms"`
}

func LoadConfig() (*Config, error) {
	var err error

	config := &Config{}
	err = envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	// Validate credentials
	if config.AWSRegion == "" || config.AWSAccessKeyId == "" || config.AWSSecretAccessKey == "" {
		return nil, fmt.Errorf("missing AWS credentials. make sure AWS_REGION, AWS_ACCESS_KEY_ID, and AWS_SECRET_ACCESS_KEY are available.")
	}

	if config.EKSCloudwatchLogGroup == "" {
		return nil, fmt.Errorf("missing EKS logs information. make sure EKS_CLOUDWATCH_LOG_GROUP, EKS_CLOUDWATCH_LOG_STREAM_PREFIX variables are available")
	}

	return config, nil
}
