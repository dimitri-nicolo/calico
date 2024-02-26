// Copyright (c) 2019, 2023 Tigera, Inc. All rights reserved.

package tunnel

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/voltron/internal/pkg/utils"
)

// Server is a connection server that accepts connections from the provided
// Listener and provides the data streams for the tunnel.
type Server struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	streamC chan *ServerStream

	// serverCerts contains the certificate chains voltron should present to connecting guardians.
	serverCerts []tls.Certificate

	// clientCertPool defines the set of root certificate authorities this server can use to verify a client certificate.
	// This should be filled with certs generated by the root CA bundle.
	clientCertPool *x509.CertPool

	tlsHandshakeTimeout time.Duration
}

// ServerOption is option for NewServer
type ServerOption func(*Server) error

// WithServerCert adds to the certificate chain which voltron should present to connecting guardians.
// This can either be a cert issued by a public CA, or the same cert used as part of the certificate bundle.
func WithServerCert(certs tls.Certificate) ServerOption {
	return func(s *Server) error {
		s.serverCerts = append(s.serverCerts, certs)

		log.Debug("tunnel.Server: TLS creds set")
		return nil
	}
}

// WithClientCert adds to the set of root certificate authorities this server can use to verify a client certificate.
// This should be filled with certs generated by the root CA bundle.
func WithClientCert(cert *x509.Certificate) ServerOption {
	return func(s *Server) error {
		certPem := utils.CertPEMEncode(cert)
		for {
			block, rest := pem.Decode(certPem)
			if block == nil {
				break
			}

			xCert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return errors.Errorf("parsing cert PEM failed: %s", err)
			}
			// s.certManager.AddCert(fmt.Sprintf("%x", sha256.Sum256(xCert.Raw)), xCert)
			s.clientCertPool.AddCert(xCert)
			certPem = rest
		}
		//nolint:staticcheck // Ignore SA1019 deprecated
		if len(s.clientCertPool.Subjects()) < 1 {
			return errors.Errorf("no block in cert key")
		}

		log.Debug("tunnel.Server: TLS client creds set")
		return nil
	}
}

// WithTLSHandshakeTimeout overrides the default 1s timeout for TLS handshake
func WithTLSHandshakeTimeout(to time.Duration) ServerOption {
	return func(s *Server) error {
		s.tlsHandshakeTimeout = to
		return nil
	}
}

// NewServer returns a new server
func NewServer(opts ...ServerOption) (*Server, error) {
	s := &Server{
		streamC:             make(chan *ServerStream),
		clientCertPool:      x509.NewCertPool(),
		tlsHandshakeTimeout: time.Second,
	}

	for _, opt := range opts {
		err := opt(s)
		if err != nil {
			return nil, errors.WithMessage(err, "tunnel.Server.New")
		}
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	return s, nil
}

// Serve starts serving connections on the given Listener
func (s *Server) Serve(lis net.Listener) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-s.ctx.Done()
		lis.Close()
	}()

	for {
		c, err := lis.Accept()
		if err != nil {
			s.cancel()
			return errors.WithMessage(err, "lis.Accept")
		}

		log.Debugf("tunnel.Server: new connection from %s", c.RemoteAddr().String())

		ss := &ServerStream{Conn: c}
		ss.ctx, ss.cancel = context.WithCancel(s.ctx)

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ss.watchServerStop()
		}()

		select {
		case s.streamC <- ss:
		case <-s.ctx.Done():
			return errors.Errorf("server stopped")
		}
	}
}

// ServeTLS starts serving TLS connections using the provided listener and the
// configured certs
func (s *Server) ServeTLS(lis net.Listener) error {
	config := calicotls.NewTLSConfig()
	config.Certificates = s.serverCerts
	config.ClientAuth = tls.RequireAndVerifyClientCert
	config.ClientCAs = s.clientCertPool

	return s.Serve(tls.NewListener(lis, config))
}

// Accept returns the next available stream or returns an error
func (s *Server) Accept() (io.ReadWriteCloser, error) {
	select {
	case ss := <-s.streamC:
		ctyp := ""
		if tlsc, ok := ss.Conn.(*tls.Conn); ok {
			if !tlsc.ConnectionState().HandshakeComplete {
				// Set timeout not to hang for ever
				_ = tlsc.SetReadDeadline(time.Now().Add(s.tlsHandshakeTimeout))
				err := tlsc.Handshake()
				if err != nil {
					msg := fmt.Sprintf("tunnel.Server TLS handshake error from %s: %s",
						tlsc.RemoteAddr().String(), err)
					_ = ss.Close()
					return nil, errors.Errorf(msg)
				}
				// reset the deadline to no timeout
				_ = tlsc.SetReadDeadline(time.Time{})
				log.Debugf("TLS HandshakeComplete %t certs %d",
					tlsc.ConnectionState().HandshakeComplete,
					len(tlsc.ConnectionState().PeerCertificates))
			}
			ctyp = "tls "
		}

		log.Debugf("tunnel.Server accepted %s connection from %s", ctyp, ss.Conn.RemoteAddr().String())
		return ss, nil
	case <-s.ctx.Done():
		return nil, errors.Errorf("server is exiting")
	}
}

// AcceptTunnel accepts a new connection as a tunnel
func (s *Server) AcceptTunnel(opts ...Option) (*Tunnel, error) {
	c, err := s.Accept()
	if err != nil {
		return nil, err
	}

	return NewServerTunnel(c, opts...)
}

func (s *Server) GetClientCertificatePool() *x509.CertPool {
	return s.clientCertPool
}

// Stop stops the server and terminates all connections.
func (s *Server) Stop() {
	s.cancel()
	s.wg.Wait()
}

type atomicBool struct {
	sync.RWMutex
	v bool
}

func (b *atomicBool) set(v bool) {
	b.Lock()
	defer b.Unlock()
	b.v = v
}

func (b *atomicBool) get() bool {
	b.RLock()
	defer b.RUnlock()
	return b.v
}

// ServerStream represents the server side of the tcp stream
type ServerStream struct {
	net.Conn

	closed        atomicBool
	closeConnOnce sync.Once
	ctx           context.Context
	cancel        context.CancelFunc
}

// Identity returns net.Addr of the remote end
func (ss *ServerStream) Identity() Identity {
	if tlsc, ok := ss.Conn.(*tls.Conn); ok {
		if len(tlsc.ConnectionState().PeerCertificates) > 0 {
			return tlsc.ConnectionState().PeerCertificates[0]
		}
	}
	return ss.Conn.RemoteAddr()
}

// Read blocks until some bytes are received or an error happens. It is OK to
// call Read and Write from different threads, but it is in general not ok to
// call Read simultaneously from different threads.
func (ss *ServerStream) Read(dst []byte) (int, error) {
	if ss.closed.get() {
		return 0, errors.Errorf("Read on a closed stream")
	}

	return ss.Conn.Read(dst)
}

// Write sends data unless an error happens.
//
// It is OK to call Read and Write from different threads, but it is in general
// not ok to call Write simultaneously from different threads.
func (ss *ServerStream) Write(data []byte) (int, error) {
	if ss.closed.get() {
		return 0, errors.Errorf("Write on a closed stream")
	}

	return ss.Conn.Write(data)
}

// Close terminates the connection
func (ss *ServerStream) Close() error {
	log.Debugf("ServerStream: Close")
	ss.cancel()
	return ss.closeConn()
}

// watchServerStop monitors the server and if it stops, it closes the
// ServerStream
func (ss *ServerStream) watchServerStop() {
	<-ss.ctx.Done()
	log.Debugf("ServerStream: watchServerStop fired")
	ss.closed.set(true)
	_ = ss.closeConn()
}

// closeConn makes sure that the ServerStream is closed only once
func (ss *ServerStream) closeConn() error {
	var err error
	ss.closeConnOnce.Do(func() {
		err = ss.Conn.Close()
		log.Debugf("ServerStream: closing connection")
	})

	return err
}
