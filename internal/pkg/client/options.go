// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package client

import (
	"crypto/x509"

	"github.com/pkg/errors"
)

// Option is a common format for New() options
type Option func(*Client) error

// ProxyTarget represents a target for WithProxyTargets. It defines where a
// request should be redirected based on patter that matches its path.
type ProxyTarget struct {
	Pattern string
	Dest    string
}

// WithProxyTargets sets the proxying targets, can be used multiple times to add
// to a union of target.
func WithProxyTargets(tgts []ProxyTarget) Option {
	return func(c *Client) error {
		for _, t := range tgts {
			if err := c.targets.Add(t.Pattern, t.Dest); err != nil {
				return err
			}
		}

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

// WithAuthBearerToken sets the bearer token to be used when proxying
func WithAuthBearerToken(token string) Option {
	return func(c *Client) error {
		c.authBearerToken = token
		return nil
	}
}
