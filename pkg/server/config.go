// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Used only when overriding config tests.
var getEnv = os.Getenv

// Environment variables that we read.
const (
	listenAddrEnv   = "LISTEN_ADDR"
	certFilePathEnv = "CERT_FILE_PATH"
	keyFilePathEnv  = "KEY_FILE_PATH"

	keyCertGenPathEnv = "KEY_CERT_GEN_PATH"

	elasticAccessModeEnv         = "ELASTIC_ACCESS_MODE"
	elasticSchemeEnv             = "ELASTIC_SCHEME"
	elasticHostEnv               = "ELASTIC_HOST"
	elasticPortEnv               = "ELASTIC_PORT"
	elasticCAPathEnv             = "ELASTIC_CA"
	elasticInsecureSkipVerifyEnv = "ELASTIC_INSECURE_SKIP_VERIFY"
	elasticUsernameEnv           = "ELASTIC_USERNAME"
	elasticPasswordEnv           = "ELASTIC_PASSWORD"
)

const (
	defaultListenAddr      = "127.0.0.1:8443"
	defaultConnectTimeout  = 30 * time.Second
	defaultKeepAlivePeriod = 30 * time.Second
	defaultIdleConnTimeout = 90 * time.Second
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

// Certificate file paths. If explicit certificates aren't provided
// then self-signed certificates are generated and stored on these
// paths.
const (
	defaultKeyCertGenPath = "/etc/es-proxy/ssl/"
	defaultCertFileName   = "cert"
	defaultKeyFileName    = "key"
)

// Config stores various configuration information for the es-proxy
// server.
type Config struct {
	// ListenAddr is the address and port that the server will listen
	// on for proxying requests. The format is similar to the address
	// parameter of net.Listen
	ListenAddr string

	// Paths to files containing certificate and matching private key
	// for serving requests over TLS.
	CertFile string
	KeyFile  string

	// If specific a CertFile and KeyFile are not provided this is the
	// location to autogenerate the files
	DefaultSSLPath string
	// Default cert and key file paths calculated from the DefaultSSLPath
	DefaultCertFile string
	DefaultKeyFile  string

	// AccessMode controls how we access es-proxy is configured to enforce
	// Elasticsearch access.
	AccessMode ElasticAccessMode

	// The URL that we should proxy requests to.
	ElasticURL                *url.URL
	ElasticCAPath             string
	ElasticInsecureSkipVerify bool

	// The username and password to inject when in ServiceUser mode.
	// Unused otherwise.
	ElasticUsername string
	ElasticPassword string

	// Various proxy timeouts. Used when creating a http.Transport RoundTripper.
	ProxyConnectTimeout  time.Duration
	ProxyKeepAlivePeriod time.Duration
	ProxyIdleConnTimeout time.Duration
}

func NewConfigFromEnv() (*Config, error) {
	listenAddr := getEnvOrDefaultString(listenAddrEnv, defaultListenAddr)
	certFilePath := getEnv(certFilePathEnv)
	keyFilePath := getEnv(keyFilePathEnv)
	keyCertGenPath := getEnvOrDefaultString(keyCertGenPathEnv, defaultKeyCertGenPath)

	defaultCertFile := keyCertGenPath + defaultCertFileName
	defaultKeyFile := keyCertGenPath + defaultKeyFileName

	accessMode, err := parseAccessMode(getEnv(elasticAccessModeEnv))
	if err != nil {
		return nil, err
	}
	elasticScheme := getEnv(elasticSchemeEnv)
	elasticHost := getEnv(elasticHostEnv)
	elasticPort := getEnv(elasticPortEnv)
	elasticURL := &url.URL{
		Scheme: elasticScheme,
		Host:   fmt.Sprintf("%s:%s", elasticHost, elasticPort),
	}
	elasticCAPath := getEnv(elasticCAPathEnv)
	elasticInsecureSkipVerify, err := strconv.ParseBool(getEnv(elasticInsecureSkipVerifyEnv))
	if err != nil {
		elasticInsecureSkipVerify = false
	}
	elasticUsername := getEnv(elasticUsernameEnv)
	elasticPassword := getEnv(elasticPasswordEnv)

	connectTimeout, err := getEnvOrDefaultDuration("PROXY_CONNECT_TIMEOUT", defaultConnectTimeout)
	if err != nil {
		return nil, err
	}
	keepAlivePeriod, err := getEnvOrDefaultDuration("PROXY_KEEPALIVE_PERIOD", defaultKeepAlivePeriod)
	if err != nil {
		return nil, err
	}
	idleConnTimeout, err := getEnvOrDefaultDuration("PROXY_IDLECONN_TIMEOUT", defaultIdleConnTimeout)
	if err != nil {
		return nil, err
	}
	config := &Config{
		ListenAddr:                listenAddr,
		CertFile:                  certFilePath,
		KeyFile:                   keyFilePath,
		DefaultSSLPath:            keyCertGenPath,
		DefaultCertFile:           defaultCertFile,
		DefaultKeyFile:            defaultKeyFile,
		AccessMode:                accessMode,
		ElasticURL:                elasticURL,
		ElasticCAPath:             elasticCAPath,
		ElasticInsecureSkipVerify: elasticInsecureSkipVerify,
		ElasticUsername:           elasticUsername,
		ElasticPassword:           elasticPassword,
		ProxyConnectTimeout:       connectTimeout,
		ProxyKeepAlivePeriod:      keepAlivePeriod,
		ProxyIdleConnTimeout:      idleConnTimeout,
	}
	err = validateConfig(config)
	return config, err
}

func getEnvOrDefaultString(key string, defaultValue string) string {
	val := getEnv(key)
	if val == "" {
		return defaultValue
	} else {
		return val
	}
}

func getEnvOrDefaultDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	val := getEnv(key)
	if val == "" {
		return defaultValue, nil
	} else {
		return time.ParseDuration(val)
	}
}

func parseAccessMode(am string) (ElasticAccessMode, error) {
	switch am {
	case "serviceuser":
		return ServiceUserMode, nil
	case "passthrough":
		return PassThroughMode, nil
	case "insecure":
		return InsecureMode, nil
	default:
		return ElasticAccessMode(""), fmt.Errorf("Indeterminate access mode %v", am)
	}
}

func validateConfig(config *Config) error {
	if config.ElasticURL.Scheme == "" || config.ElasticURL.Host == "" {
		return errors.New("Invalid Elasticsearch backend URL specified")
	}
	if (config.AccessMode == PassThroughMode || config.AccessMode == InsecureMode) &&
		(config.ElasticUsername != "" || config.ElasticPassword != "") {
		return errors.New("Cannot set Elasticsearch credentials in Passthrough or Insecure mode")
	}
	if config.AccessMode == ServiceUserMode &&
		(config.ElasticUsername == "" || config.ElasticPassword == "") {
		return errors.New("Elasticsearch credentials not provided for Service user mode")
	}
	if config.ElasticURL.Scheme == "https" && config.ElasticCAPath == "" {
		return errors.New("Elasticsearch CA not provided")
	}
	if config.ElasticURL.Scheme == "http" && config.ElasticCAPath != "" {
		return errors.New("Elasticsearch CA provided but scheme is set to http")

	}
	return nil
}
