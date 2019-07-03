// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package client

import (
	"crypto/x509"

	"github.com/pkg/errors"

	"github.com/tigera/voltron/internal/pkg/proxy"
)

// Option is a common format for New() options
type Option func(*Client) error

// WithProxyTargets sets the proxying targets, can be used multiple times to add
// to a union of target.
func WithProxyTargets(tgts []proxy.Target) Option {
	return func(c *Client) error {
		c.targets = tgts
		return nil
	}
}

// WithDefaultAddr changes the default address where the server accepts
// connections when Listener is not provided.
func WithDefaultAddr(addr string) Option {
	return func(c *Client) error {
		c.http.Addr = addr
		return nil
	}
}

// WithTunnelCreds sets the credential to be used when establishing the tunnel
func WithTunnelCreds(certPEM []byte, keyPEM []byte, ca *x509.CertPool) Option {
	return func(c *Client) error {
		if certPEM == nil || keyPEM == nil {
			return errors.Errorf("WithTunnelCreds: cert and key are required")
		}
		c.tunnelCertPEM = certPEM
		c.tunnelKeyPEM = keyPEM
		c.tunnelRootCAs = ca
		return nil
	}
}
