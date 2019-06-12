// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

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

// WithCredsFiles sets the default cert and key to be used for TLS connections
func WithCredsFiles(cert, key string) Option {
	return func(s *Server) error {
		s.certFile = cert
		s.keyFile = key
		// XXX perhaps check if the files exist?
		return nil
	}
}
