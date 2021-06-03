package server

import (
	"crypto/tls"
	"io/ioutil"
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

// WithInternalCreds creates the cert and key from the given pem bytes to be used for the TLS connections for
// external traffic (UI).
func WithInternalCreds(certBytes []byte, keyBytes []byte) Option {
	return func(s *Server) error {
		var err error
		s.internalCert, err = tls.X509KeyPair(certBytes, keyBytes)
		return err
	}
}
