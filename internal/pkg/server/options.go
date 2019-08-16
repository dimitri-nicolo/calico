// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"crypto"
	"crypto/x509"
	"io/ioutil"
	"os"
	"time"

	"k8s.io/client-go/rest"

	"github.com/tigera/voltron/internal/pkg/auth"
	"github.com/tigera/voltron/internal/pkg/proxy"

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

// WithTemplate adds the path to the manifest template
func WithTemplate(templatePath string) Option {
	return func(s *Server) error {
		templateContent, err := ioutil.ReadFile(templatePath)
		if err != nil {
			return errors.Errorf("Could not read template from path %v", err)
		}

		s.template = string(templateContent)
		return nil
	}
}

// WithPublicAddr assigns a public address
func WithPublicAddr(address string) Option {
	return func(s *Server) error {
		s.publicAddress = address
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

// WithAuthentication sets the kubernetes client that will be used to interact with its api
func WithAuthentication(k8sConfig *rest.Config) Option {
	return func(s *Server) error {
		if s.k8s == nil {
			return errors.Errorf("WithAuthentication requires the k8s.Interface to be set")
		}
		s.auth = auth.NewIdentity(s.k8s, k8sConfig)
		return nil
	}
}

// WithKeepAliveSettings sets the Keep Alive settings for the tunnel.
func WithKeepAliveSettings(enable bool, intervalMs int) Option {
	return func(s *Server) error {
		s.tunnelEnableKeepAlive = enable
		s.tunnelKeepAliveInterval = time.Duration(intervalMs) * time.Millisecond
		return nil
	}
}

// WithDefaultProxy set the default proxy if no x-cluster-id header is present.
// it is optional. If not set, server returns 400 error if a request does not
// have the x-cluster-id set.
func WithDefaultProxy(p *proxy.Proxy) Option {
	return func(s *Server) error {
		s.defaultProxy = p
		return nil
	}
}

// WithAutoRegister accept tunnels with genuine cert even if there is no cluster
// registerd for it. This is a poor man's crash recovery and mainly for
// debugging, should not be used in production.
func WithAutoRegister() Option {
	return func(s *Server) error {
		s.toggles.autoRegister = true
		return nil
	}
}

// WithWatchAdded lets server to watch k8s for added clusters and register them
// this way. This is useful for testing.
func WithWatchAdded() Option {
	return func(s *Server) error {
		s.clusters.watchAdded = true
		return nil
	}
}
