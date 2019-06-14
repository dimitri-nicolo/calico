// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"crypto"
	"crypto/x509"
	"os"

	"github.com/pkg/errors"
)

// Option is a common format for New() options
type Option func(*Server) error

// WithDefaultAddr changes the default address where the server accepts
// connections when Listener is not provided.
func WithDefaultAddr(addr string) Option {
	return func(s *Server) error {
		s.http.Addr = addr
		return nil
	}
}

// ProxyTarget represents a target for WithProxyTargets. It defines where a
// request should be redirected based on patter that matches its path.
type ProxyTarget struct {
	Pattern string
	Dest    string
}

// WithCredsFiles sets the default cert and key to be used for the TLS
// connections and for the tunnel.
func WithCredsFiles(cert, key string) Option {
	return func(s *Server) error {
		// Check if files exist
		if _, err := os.Stat(cert); os.IsNotExist(err) {
			return errors.Errorf("cert file: %s", err)
		}

		if _, err := os.Stat(key); os.IsNotExist(err) {
			return errors.Errorf("cert file: %s", err)
		}

		s.certFile = cert
		s.keyFile = key

		return nil
	}
}

// WithTunnelCreds sets the credentials to be used for the tunnel server and to
// be used to generate creds for guardians. If not populated WithCredsFiles
// creds will be used.
func WithTunnelCreds(cert *x509.Certificate, key crypto.Signer) Option {
	return func(s *Server) error {
		s.tunnelCert = cert
		s.tunnelKey = key
		return nil
	}
}

// WithKeepClusterKeys allows the server to keep the generated private keys.
// This is to be used only for debugging and testing
func WithKeepClusterKeys() Option {
	return func(s *Server) error {
		s.clusters.keepKeys = true
		return nil
	}
}
