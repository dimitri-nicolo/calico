package server

import (
	"crypto/tls"
	"io/ioutil"

	"github.com/tigera/es-gateway/pkg/elastic"
)

// Option is a common format for New() options
type Option func(*Server) error

// WithAddr changes the address where the server accepts
// connections when Listener is not provided.
func WithAddr(addr string) Option {
	return func(s *Server) error {
		s.addr = addr
		return nil
	}
}

// WithInternalTLSFiles sets the cert and key to be used for the TLS
// connections for internal traffic (this includes in-cluster requests or
// ones coming from Voltron tunnel).
func WithInternalTLSFiles(certFile, keyFile string) Option {
	return func(s *Server) error {
		var err error

		certPEMBlock, err := ioutil.ReadFile(certFile)
		if err != nil {
			return err
		}
		keyPEMBlock, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return err
		}

		return WithInternalCreds(certPEMBlock, keyPEMBlock)(s)
	}
}

// WithInternalCreds creates a tls.Certificate chain from the given key pair bytes.
// This certificate chain is used for TLS connections for all external client requests.
func WithInternalCreds(certBytes []byte, keyBytes []byte) Option {
	return func(s *Server) error {
		var err error
		s.internalCert, err = tls.X509KeyPair(certBytes, keyBytes)
		return err
	}
}

// WithESClient sets the Elasticsearch client for the server (needed for Elasticsearch
// API calls like authentication checking).
func WithESClient(client elastic.Client) Option {
	return func(s *Server) error {
		s.esClient = client
		return nil
	}
}
