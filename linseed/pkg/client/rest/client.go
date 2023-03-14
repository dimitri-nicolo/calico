// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package rest

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
)

// RESTClient is a helper for building HTTP requests for the Linseed API.
type RESTClient interface {
	BaseURL() string
	Tenant() string
	HTTPClient() *http.Client
	Verb(string) Request
	Post() Request
	Put() Request
	Delete() Request
}

type restClient struct {
	config Config
	client *http.Client

	tenantID string
}

type Config struct {
	// The base URL of the server
	URL string

	// CACertPath is the path to the CA cert for verifying
	// the server certificate provided by Linseed.
	CACertPath string

	// ClientCertPath is the path to the client certificate this client
	// should present to Linseed for mTLS authentication.
	ClientCertPath string

	// ClientKeyPath is the path to the client key used for mTLS.
	ClientKeyPath string

	// Whether or not TLS should be limited to FIPS compatible ciphers
	// and TLS versions.
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

	// Create a custom dialer so that we can configure a dial timeout.
	// If we can't connect to Linseed within 10 seconds, something is up.
	// Note: this is not the same as the request timeout, which is handled via the
	// provided context on a per-request basis.
	dialWithTimeout := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, 10*time.Second)
	}
	httpTransport := &http.Transport{
		Dial:            dialWithTimeout,
		TLSClientConfig: tlsConfig,
	}

	if cfg.ClientKeyPath != "" && cfg.ClientCertPath != "" {
		clientCert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("error load cert key pair for linseed client: %s", err)
		}
		httpTransport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}
		logrus.Info("Using provided client certificates for mTLS")
	}
	return &http.Client{
		Transport: httpTransport,
	}, nil
}

func (c *restClient) Verb(verb string) Request {
	return NewRequest(c).Verb(verb)
}

func (c *restClient) Post() Request {
	return c.Verb(http.MethodPost)
}

func (c *restClient) Put() Request {
	return c.Verb(http.MethodPut)
}

func (c *restClient) Delete() Request {
	return c.Verb(http.MethodDelete)
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
