// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// Used only when overriding config tests.
var getEnv = os.Getenv

// Environment variables that we read.
const (
	listenAddrEnv   = "LISTEN_ADDR"
	certFilePathEnv = "CERT_FILE_PATH"
	keyFilePathEnv  = "KEY_FILE_PATH"

	keyCertGenPathEnv = "KEY_CERT_GEN_PATH"

	elasticSchemeEnv             = "ELASTIC_SCHEME"
	elasticHostEnv               = "ELASTIC_HOST"
	elasticPortEnv               = "ELASTIC_PORT"
	elasticCAPathEnv             = "ELASTIC_CA"
	elasticInsecureSkipVerifyEnv = "ELASTIC_INSECURE_SKIP_VERIFY"
	elasticUsernameEnv           = "ELASTIC_USERNAME"
	elasticPasswordEnv           = "ELASTIC_PASSWORD"
	elasticIndexSuffixEnv        = "ELASTIC_INDEX_SUFFIX"
	elasticConnRetriesEnv        = "ELASTIC_CONN_RETRIES"
	elasticConnRetryIntervalEnv  = "ELASTIC_CONN_RETRY_INTERVAL"
	elasticEnableTraceEnv        = "ELASTIC_ENABLE_TRACE"
	elasticLicenseTypeEnv        = "ELASTIC_LICENSE_TYPE"
	elasticVersionEnv            = "ELASTIC_VERSION"
	elasticKibanaEndpointEnv     = "ELASTIC_KIBANA_ENDPOINT"

	voltronCAPathEnv = "VOLTRON_CA_PATH"

	// Dex settings for authentication.
	oidcAuthEnabledEnv        = "OIDC_AUTH_ENABLED"
	oidcAuthIssuerEnv         = "OIDC_AUTH_ISSUER"
	oidcAuthClientIDEnv       = "OIDC_AUTH_CLIENT_ID"
	oidcAuthJWKSURLEnv        = "OIDC_AUTH_JWKSURL"
	oidcAuthUsernameClaimEnv  = "OIDC_AUTH_USERNAME_CLAIM"
	oidcAuthGroupsClaimEnv    = "OIDC_AUTH_GROUPS_CLAIM"
	oidcAuthUsernamePrefixEnv = "OIDC_AUTH_USERNAME_PREFIX"
	oidcAuthGroupsPrefixEnv   = "OIDC_AUTH_GROUPS_PREFIX"
)

const (
	defaultListenAddr      = "127.0.0.1:8443"
	defaultConnectTimeout  = 30 * time.Second
	defaultKeepAlivePeriod = 30 * time.Second
	defaultIdleConnTimeout = 90 * time.Second

	defaultIndexSuffix       = "cluster"
	defaultConnRetryInterval = 500 * time.Millisecond
	defaultConnRetries       = 30
	defaultEnableTrace       = false

	defaultUsernameClaim = "email"
	defaultJWSKURL       = "https://tigera-dex.tigera-dex.svc.cluster.local:5556/dex/keys"

	defaultKibanaEndpoint = "https://tigera-secure-kb-http.tigera-kibana.svc:5601"
)

const (
	// Certificate file paths. If explicit certificates aren't provided
	// then self-signed certificates are generated and stored on these
	// paths.
	defaultKeyCertGenPath = "/etc/es-proxy/ssl/"
	defaultCertFileName   = "cert"
	defaultKeyFileName    = "key"

	// We will use HTTPS if the env variable ELASTIC_SCHEME is not set.
	defaultElasticScheme = "https"

	defaultVoltronCAPath = "/manager-tls/cert"
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

	// The URL that we should proxy requests to.
	ElasticURL                *url.URL
	ElasticCAPath             string
	ElasticInsecureSkipVerify bool

	// The username and password to inject when in ServiceUser mode.
	// Unused otherwise.
	ElasticUsername string
	ElasticPassword string

	ElasticIndexSuffix       string
	ElasticConnRetries       int
	ElasticConnRetryInterval time.Duration
	ElasticEnableTrace       bool
	ElasticLicenseType       string
	ElasticVersion           string
	ElasticKibanaEndpoint    string

	// Various proxy timeouts. Used when creating a http.Transport RoundTripper.
	ProxyConnectTimeout  time.Duration
	ProxyKeepAlivePeriod time.Duration
	ProxyIdleConnTimeout time.Duration

	// If multi-cluster management is used inside the cluster, this CA
	// is necessary for establishing a connection with Voltron, when
	// accessing other clusters.
	VoltronCAPath string

	// Dex settings for authentication.
	OIDCAuthEnabled        bool
	OIDCAuthIssuer         string
	OIDCAuthClientID       string
	OIDCAuthJWKSURL        string
	OIDCAuthUsernameClaim  string
	OIDCAuthGroupsClaim    string
	OIDCAuthUsernamePrefix string
	OIDCAuthGroupsPrefix   string
}

func NewConfigFromEnv() (*Config, error) {
	listenAddr := getEnvOrDefaultString(listenAddrEnv, defaultListenAddr)
	certFilePath := getEnv(certFilePathEnv)
	keyFilePath := getEnv(keyFilePathEnv)
	keyCertGenPath := getEnvOrDefaultString(keyCertGenPathEnv, defaultKeyCertGenPath)

	defaultCertFile := keyCertGenPath + defaultCertFileName
	defaultKeyFile := keyCertGenPath + defaultKeyFileName

	elasticScheme := getEnvOrDefaultString(elasticSchemeEnv, defaultElasticScheme)
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

	elasticIndexSuffix := getEnvOrDefaultString(elasticIndexSuffixEnv, defaultIndexSuffix)
	elasticConnRetries, err := getEnvOrDefaultInt(elasticConnRetriesEnv, defaultConnRetries)
	if err != nil {
		return nil, err
	}
	elasticConnRetryInterval, err := getEnvOrDefaultDuration(elasticConnRetryIntervalEnv, defaultConnRetryInterval)
	if err != nil {
		return nil, err
	}
	elasticEnableTrace, err := getEnvOrDefaultBool(elasticEnableTraceEnv, defaultEnableTrace)
	if err != nil {
		log.WithError(err).Error("Failed to parse " + elasticEnableTraceEnv)
		elasticEnableTrace = false
	}

	elasticLicenseType := getEnv(elasticLicenseTypeEnv)
	elasticVersion := getEnv(elasticVersionEnv)

	elasticKibanaEndpoint := getEnvOrDefaultString(elasticKibanaEndpointEnv, defaultKibanaEndpoint)

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
	voltronCAPath := getEnvOrDefaultString(voltronCAPathEnv, defaultVoltronCAPath)

	oidcAuthEnabled, err := getEnvOrDefaultBool(oidcAuthEnabledEnv, false)
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
		ElasticURL:                elasticURL,
		ElasticCAPath:             elasticCAPath,
		ElasticInsecureSkipVerify: elasticInsecureSkipVerify,
		ElasticUsername:           elasticUsername,
		ElasticPassword:           elasticPassword,
		ElasticIndexSuffix:        elasticIndexSuffix,
		ElasticConnRetryInterval:  elasticConnRetryInterval,
		ElasticEnableTrace:        elasticEnableTrace,
		ElasticLicenseType:        elasticLicenseType,
		ElasticVersion:            elasticVersion,
		ElasticKibanaEndpoint:     elasticKibanaEndpoint,
		ElasticConnRetries:        int(elasticConnRetries),
		ProxyConnectTimeout:       connectTimeout,
		ProxyKeepAlivePeriod:      keepAlivePeriod,
		ProxyIdleConnTimeout:      idleConnTimeout,
		VoltronCAPath:             voltronCAPath,
		OIDCAuthEnabled:           oidcAuthEnabled,
		OIDCAuthIssuer:            getEnv(oidcAuthIssuerEnv),
		OIDCAuthClientID:          getEnv(oidcAuthClientIDEnv),
		OIDCAuthJWKSURL:           getEnvOrDefaultString(oidcAuthJWKSURLEnv, defaultJWSKURL),
		OIDCAuthUsernameClaim:     getEnvOrDefaultString(oidcAuthUsernameClaimEnv, defaultUsernameClaim),
		OIDCAuthGroupsClaim:       getEnv(oidcAuthGroupsClaimEnv),
		OIDCAuthUsernamePrefix:    getEnv(oidcAuthUsernamePrefixEnv),
		OIDCAuthGroupsPrefix:      getEnv(oidcAuthGroupsPrefixEnv),
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

func getEnvOrDefaultInt(key string, defaultValue int) (int, error) {
	val := getEnv(key)
	if val == "" {
		return defaultValue, nil
	}

	i, err := strconv.ParseInt(getEnv(val), 10, 0)
	if err != nil {
		return 0, err
	}

	return int(i), nil
}

func getEnvOrDefaultBool(key string, defaultValue bool) (bool, error) {
	log.Debug(key + " " + getEnv(key))
	if val := getEnv(key); val != "" {
		return strconv.ParseBool(val)
	}
	return defaultValue, nil
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
