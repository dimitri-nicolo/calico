// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package server

import (
	"crypto/tls"
	"io/ioutil"

	"github.com/tigera/es-gateway/pkg/metrics"

	"github.com/tigera/es-gateway/pkg/clients/elastic"
	"github.com/tigera/es-gateway/pkg/clients/kibana"
	"github.com/tigera/es-gateway/pkg/clients/kubernetes"
	"github.com/tigera/es-gateway/pkg/proxy"
)

// Option is a common format for New() options
type Option func(*Server) error

// WithCollector adds a collector for Prometheus metrics.
func WithCollector(collector metrics.Collector) Option {
	return func(s *Server) error {
		s.collector = collector
		return nil
	}
}

// WithAddr changes the address where the server accepts
// connections when Listener is not provided.
func WithAddr(addr string) Option {
	return func(s *Server) error {
		s.addr = addr
		return nil
	}
}

// WithESTarget sets the ES target for the server.
func WithESTarget(t *proxy.Target) Option {
	return func(s *Server) error {
		s.esTarget = t
		return nil
	}
}

// WithKibanaTarget sets the Kibana target for the server.
func WithKibanaTarget(t *proxy.Target) Option {
	return func(s *Server) error {
		s.kibanaTarget = t
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

// WithKibanaClient sets the Kibana client for the server (needed for Kibana API call
// to perform health checking).
func WithKibanaClient(client kibana.Client) Option {
	return func(s *Server) error {
		s.kbClient = client
		return nil
	}
}

// WithK8sClient sets the K8s clientset for the server (needed for retrieving K8s resources
// related to ES users).
func WithK8sClient(client kubernetes.Client) Option {
	return func(s *Server) error {
		s.k8sClient = client
		return nil
	}
}

// WithAdminUser sets the username and password of the real ES admin user for the server
// (needed during ES credential swapping to handle special scenarios where valid requests need
// to use the Elastic admin user).
func WithAdminUser(u, p string) Option {
	return func(s *Server) error {
		s.adminESUsername = u
		s.adminESPassword = p
		return nil
	}
}
