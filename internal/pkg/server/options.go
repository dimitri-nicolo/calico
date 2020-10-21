// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package server

import (
	"crypto"
	"crypto/x509"
	"os"
	"regexp"
	"time"

	"github.com/tigera/voltron/internal/pkg/proxy"

	"github.com/pkg/errors"
)

// Option is a common format for New() options
type Option func(*Server) error

// WithDefaultAddr changes the default address where the server accepts
// connections when Listener is not provided.
func WithDefaultAddr(addr string) Option {
	return func(s *Server) error {
		s.addr = addr
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

var missingFileErrorMessage = "file %s not found : %s"

// WithExternalCredsFiles sets the default cert and key to be used for the TLS
// connections for external traffic (UI).
func WithExternalCredsFiles(cert, key string) Option {
	return func(s *Server) error {
		// Check if files exist
		if _, err := os.Stat(cert); os.IsNotExist(err) {
			return errors.Errorf(missingFileErrorMessage, cert, err)
		}

		if _, err := os.Stat(key); os.IsNotExist(err) {
			return errors.Errorf(missingFileErrorMessage, key, err)
		}

		s.certFile = cert
		s.keyFile = key

		return nil
	}
}

// WithInternalCredsFiles sets the default cert and key to be used for the TLS
// connections within the management cluster.
func WithInternalCredFiles(cert, key string) Option {
	return func(s *Server) error {
		// Check if files exist
		if _, err := os.Stat(cert); os.IsNotExist(err) {
			return errors.Errorf(missingFileErrorMessage, cert, err)
		}

		if _, err := os.Stat(key); os.IsNotExist(err) {
			return errors.Errorf(missingFileErrorMessage, key, err)
		}

		s.internalCertFile = cert
		s.internalKeyFile = key

		return nil
	}
}

// WithTunnelCreds sets the credentials to be used for the tunnel server and to
// be used to generate creds for guardians.
func WithTunnelCreds(cert *x509.Certificate, key crypto.Signer) Option {
	return func(s *Server) error {
		s.tunnelCert = cert
		s.tunnelKey = key
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
// it is optional. If this option is not set, then the server will returns a 400
// error when a request does not have the x-cluster-id header set.
func WithDefaultProxy(p *proxy.Proxy) Option {
	return func(s *Server) error {
		s.defaultProxy = p
		return nil
	}
}

// WithTunnelTargetWhitelist sets a whitelist of regex representing potential target paths
// that should go through tunnel proxying in the server. This happens within the server
// clusterMux handler.
func WithTunnelTargetWhitelist(tgts []regexp.Regexp) Option {
	return func(s *Server) error {
		s.tunnelTargetWhitelist = tgts
		return nil
	}
}

// WithForwardingEnabled sets if we should allow forwarding to another server
func WithForwardingEnabled(forwardingEnabled bool) Option {
	return func(s *Server) error {
		s.clusters.forwardingEnabled = forwardingEnabled
		return nil
	}
}

// WithDefaultForwardServer sets the server that requests from guardian should be sent to by default
func WithDefaultForwardServer(serverName string, dialRetryAttempts int, dialRetryInterval time.Duration) Option {
	return func(s *Server) error {
		s.clusters.defaultForwardServerName = serverName
		s.clusters.defaultForwardDialRetryAttempts = dialRetryAttempts
		s.clusters.defaultForwardDialRetryInterval = dialRetryInterval
		return nil
	}
}

// WithKubernetesAPITargets sets a whitelist of regex representing target paths
// that target the kube (a)api server
func WithKubernetesAPITargets(tgts []regexp.Regexp) Option {
	return func(s *Server) error {
		s.kubernetesAPITargets = tgts
		return nil
	}
}
