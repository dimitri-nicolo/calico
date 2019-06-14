// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package client

import (
	"os"

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

// WithCredsFiles sets the default cert and key to be used for TLS connections
func WithCredsFiles(cert, key string) Option {
	return func(c *Client) error {
		// Check if files exist
		if _, err := os.Stat(cert); os.IsNotExist(err) {
			return errors.Errorf("cert file: %s", err)
		}

		if _, err := os.Stat(key); os.IsNotExist(err) {
			return errors.Errorf("key file: %s", err)
		}

		c.certFile = cert
		c.keyFile = key

		return nil
	}
}
