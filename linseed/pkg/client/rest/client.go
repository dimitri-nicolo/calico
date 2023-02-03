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
type RESTClient struct {
	config Config
	client *http.Client

	clusterID string
	tenantID  string
}

type Config struct {
	URL             string
	CACertPath      string
	ClientCertPath  string
	ClientKeyPath   string
	FipsModeEnabled bool
}

// NewClient returns a new RESTClient.
func NewClient(clusterID, tenantID string, cfg Config) (*RESTClient, error) {
	httpClient, err := newHTTPClient(cfg)
	if err != nil {
		return nil, err
	}

	return &RESTClient{
		config:    cfg,
		clusterID: clusterID,
		tenantID:  tenantID,
		client:    httpClient,
	}, nil
}

func newHTTPClient(cfg Config) (*http.Client, error) {
	caCert, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("error reading CA file: %s", err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCert)
	if !ok {
		return nil, fmt.Errorf("failed to parse root certificate")
	}
	tlsConfig := calicotls.NewTLSConfig(cfg.FipsModeEnabled)
	tlsConfig.RootCAs = caCertPool
	httpTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	clientCert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error load cert key pair for linseed client: %s", err)
	}
	httpTransport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}
	return &http.Client{
		Transport: httpTransport,
	}, nil
}

func (c *RESTClient) Verb(verb string) *Request {
	return NewRequest(c).Verb(verb)
}

func (c *RESTClient) Post() *Request {
	return c.Verb("POST")
}
