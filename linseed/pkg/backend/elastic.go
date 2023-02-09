// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package backend

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"strings"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// ElasticConfig is the configuration that selects the flavour
// of the client that can communicate with an Elastic Cluster
type ElasticConfig struct {
	// URL is endpoint of the Elastic cluster
	URL string

	// Username is the username to authenticate with Elastic
	// If left empty, no authentication will be enabled
	Username string

	// Password is the password to authenticate with Elastic
	// If left empty, no authentication will be enabled
	Password string

	// CACertPath is the absolute path of the CA certificate
	// of Elastic
	CACertPath string

	// MTLSEnabled will determine whether communication with
	// Elastic enabled mutual authentication or not
	MTLSEnabled bool

	// ClientCertPath is the absolute path of the client certificate
	// generated for the Elastic client; It will be used only if
	// MTLSEnabled is set to true
	ClientCertPath string

	// ClientCertKeyPath is the absolute path of the client key
	// generated for the Elastic client; It will be used only if
	// MTLSEnabled is set to true
	ClientCertKeyPath string

	// GZIPEnabled enables GZIP communication between Elastic
	// and the client
	GZIPEnabled bool

	// LogLevel will configure the log level enabled for the
	// Elastic client
	LogLevel string

	// FIPSModeEnabled will configure TLS to match FIPS requirements
	FIPSModeEnabled bool

	// Scheme determines the protocol used to sniff Elastic nodes
	// Can be HTTP or HTTPS
	Scheme string

	// EnableSniffing will enable sniffing (or node discovery)
	// and connect only with the available nodes and keep in
	// memory a state of the cluster. Since ES is run as K8S
	// service, and we cannot connect directly to the nodes
	// sniffing will be disabled
	EnableSniffing bool
}

// MustGetElasticClient will create an elastic client or stop execution
// if configurations like certificate paths are invalid
func MustGetElasticClient(config ElasticConfig) lmaelastic.Client {
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(config.URL),
		elastic.SetScheme(config.Scheme),
		elastic.SetGzip(config.GZIPEnabled),
		elastic.SetSniff(config.EnableSniffing),
	}

	if config.Username != "" && config.Password != "" {
		options = append(options, elastic.SetBasicAuth(config.Username, config.Password))
	}

	// Use the standard logger to inherit configuration.
	log := logrus.StandardLogger()

	switch strings.ToLower(config.LogLevel) {
	case "error":
		options = append(options, elastic.SetErrorLog(log))
	case "info", "debug", "warning":
		options = append(options, elastic.SetInfoLog(log))
	case "trace":
		options = append(options, elastic.SetTraceLog(log))
	}

	options = append(options, elastic.SetHttpClient(mustGetHTTPClient(config)))
	esClient, err := elastic.NewClient(options...)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create Elastic client")
	}

	return lmaelastic.NewWithClient(esClient)
}

func mustGetHTTPClient(config ElasticConfig) *http.Client {
	if config.CACertPath == "" || config.ClientCertPath == "" || config.ClientCertKeyPath == "" {
		logrus.Warn("No certificates were passed in for Elastic. Will use a default HTTP client")
		return http.DefaultClient
	}

	// Configure TLS
	tlsConfig := calicotls.NewTLSConfig(config.FIPSModeEnabled)

	// Configure CA certificates
	caCertPool := mustGetCACertPool(config)
	tlsConfig.RootCAs = caCertPool

	// Configure clients certificate if needed
	if config.MTLSEnabled {
		clientCert := mustGetClientCert(config)
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
}

func mustGetClientCert(config ElasticConfig) tls.Certificate {
	// Read client certificate
	clientCert, err := tls.LoadX509KeyPair(config.ClientCertPath, config.ClientCertKeyPath)
	if err != nil {
		logrus.WithError(err).Fatal("Failed load client x509 certificates")
	}
	return clientCert
}

func mustGetCACertPool(config ElasticConfig) *x509.CertPool {
	// Read CA cert file
	caCert, err := os.ReadFile(config.CACertPath)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to read CA certificate")
	}

	// Append CA to cert pool
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCert)
	if !ok {
		logrus.Fatal("Failed to parse root certificate")
	}
	return caCertPool
}
