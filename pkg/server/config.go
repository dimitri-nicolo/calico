// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"errors"
	"fmt"
	"net/url"
	"os"
)

const (
	listenAddrEnv      = "LISTEN_ADDR"
	accessModeEnv      = "ACCESS_MODE"
	elasticSchemeEnv   = "ELASTIC_SCHEME"
	elasticHostEnv     = "ELASTIC_HOST"
	elasticPortEnv     = "ELASTIC_PORT"
	elasticUsernameEnv = "ELASTIC_USERNAME"
	elasticPasswordEnv = "ELASTIC_PASSWORD"
)

type ElasticAccessMode string

const (
	// In PassThroughMode users are managed in Elasticsearch
	// and the proxy will pass this information over.
	PassThroughMode ElasticAccessMode = "passthrough"

	// In ServiceUserMode the users are authorized and the
	// Elasticsearch is accessed on behalf of the user using
	// the service's Elasticsearch credentials.
	ServiceUserMode = "serviceuser"

	// In InsecureMode access to Elasticsearch is not password
	// protected.
	InsecureMode = "insecure"
)

// Config stores various configuration information for the es-proxy
// server.
type Config struct {
	// ListeAddr is the address and port that the server will listen
	// on for proxying requests. The format is similar to the address
	// parameter of net.Listen
	ListenAddr string

	// AccessMode controls how we access es-proxy is configured to enforce
	// Elasticsearch access.
	AccessMode ElasticAccessMode

	// The URL that we should proxy requests to.
	ElasticURL *url.URL

	// The usename and password to inject when in Singleuser mode.
	// Unused otherwise.
	ElasticUsername string
	ElasticPassword string
}

func NewConfigFromEnv() (*Config, error) {
	listenAddr := os.Getenv(listenAddrEnv)
	accessMode := parseAccessMode(os.Getenv(accessModeEnv))
	elasticScheme := os.Getenv(elasticSchemeEnv)
	elasticHost := os.Getenv(elasticHostEnv)
	elasticPort := os.Getenv(elasticPortEnv)
	elasticURL := &url.URL{
		Scheme: elasticScheme,
		Host:   fmt.Sprintf("%s:%s", elasticHost, elasticPort),
	}
	elasticUsername := os.Getenv(elasticUsernameEnv)
	elasticPassword := os.Getenv(elasticPasswordEnv)
	config := &Config{
		ListenAddr:      listenAddr,
		AccessMode:      accessMode,
		ElasticURL:      elasticURL,
		ElasticUsername: elasticUsername,
		ElasticPassword: elasticPassword,
	}
	err := validateConfig(config)
	return config, err
}

func parseAccessMode(am string) ElasticAccessMode {
	switch am {
	case "serviceuser":
		return ServiceUserMode
	case "passthrough":
		return PassThroughMode
	default:
		return InsecureMode
	}
}

func validateConfig(config *Config) error {
	if config.AccessMode == PassThroughMode &&
		config.ElasticUsername != "" && config.ElasticPassword != "" {
		return errors.New("Cannot set Elasticsearch credentials in Passthrough mode")

	}

	return nil
}
