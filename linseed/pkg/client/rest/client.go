// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package rest

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
)

// RESTClient is a helper for building HTTP requests for the Linseed API.
type RESTClient interface {
	BaseURL() string
	Tenant() string
	HTTPClient() *http.Client
	Verb(string) Request
	Post() Request
}

type restClient struct {
	config Config
	client *http.Client

	tenantID string
}

type Config struct {
	// The base URL of the server
	URL string

	// TLS configuration.
	CACertPath      string
	ClientCertPath  string
	ClientKeyPath   string
	FIPSModeEnabled bool
}

// NewClient returns a new restClient.
func NewClient(tenantID string, cfg Config) (RESTClient, error) {
	httpClient, err := newHTTPClient(cfg)
	if err != nil {
		return nil, err
	}

	return &restClient{
		config:   cfg,
		tenantID: tenantID,
		client:   httpClient,
	}, nil
}

func newHTTPClient(cfg Config) (*http.Client, error) {
	tlsConfig := calicotls.NewTLSConfig(cfg.FIPSModeEnabled)
	if cfg.CACertPath != "" {
		caCertPool := x509.NewCertPool()
		caCert, err := os.ReadFile(cfg.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("error reading CA file: %s", err)
		}
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return nil, fmt.Errorf("failed to parse root certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}
	httpTransport := &http.Transport{TLSClientConfig: tlsConfig}

	if cfg.ClientKeyPath != "" && cfg.ClientCertPath != "" {
		clientCert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("error load cert key pair for linseed client: %s", err)
		}
		httpTransport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}
	}
	return &http.Client{
		Transport: httpTransport,
	}, nil
}

func (c *restClient) Verb(verb string) Request {
	return NewRequest(c).Verb(verb)
}

func (c *restClient) Post() Request {
	return c.Verb("POST")
}

func (c *restClient) BaseURL() string {
	return c.config.URL
}

func (c *restClient) Tenant() string {
	return c.tenantID
}

func (c *restClient) HTTPClient() *http.Client {
	return c.client
}
