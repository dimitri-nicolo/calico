// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package tunnel defines an authenticated tunnel API, that allows creating byte
// pipes in both directions, initiated from either side of the tunnel.
package tunnel

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Tunnel represents either side of the tunnel that allows waiting for,
// accepting and initiating creation of new BytePipes.
type Tunnel struct {
	stream io.ReadWriteCloser
	mux    *yamux.Session

	errOnce sync.Once
	errCh   chan struct{}
	err     error

	keepAliveEnable   bool
	keepAliveInterval time.Duration
}

func newTunnel(stream io.ReadWriteCloser, isServer bool, opts ...Option) (*Tunnel, error) {
	t := &Tunnel{
		stream: stream,
		errCh:  make(chan struct{}),

		// Defaults
		keepAliveEnable:   true,
		keepAliveInterval: 100 * time.Millisecond,
	}

	var mux *yamux.Session
	var err error

	for _, o := range opts {
		if err := o(t); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	// XXX all the config options should probably become options taken by New()
	// XXX that can override the defaults set here
	config := yamux.DefaultConfig()
	config.AcceptBacklog = 1
	config.EnableKeepAlive = t.keepAliveEnable
	config.KeepAliveInterval = t.keepAliveInterval

	if isServer {
		mux, err = yamux.Server(&serverCloser{
			ReadWriteCloser: stream,
			t:               t,
		},
			config)
	} else {
		mux, err = yamux.Client(stream, config)
	}

	if err != nil {
		return nil, errors.Errorf("New failed creating muxer: %+v", err)
	}

	t.mux = mux

	return t, nil
}

// NewServerTunnel returns a new tunnel that uses the provided stream as the
// carrier. The stream must be the server side of the stream
func NewServerTunnel(stream io.ReadWriteCloser, opts ...Option) (*Tunnel, error) {
	return newTunnel(stream, true, opts...)
}

// NewClientTunnel returns a new tunnel that uses the provided stream as the
// carrier. The stream must be the client side of the stream
func NewClientTunnel(stream io.ReadWriteCloser, opts ...Option) (*Tunnel, error) {
	return newTunnel(stream, false, opts...)
}

// Identity represents remote peer identity
// XXX the exact type TBD
type Identity = interface{}

type hasIdentity interface {
	Identity() Identity
}

// Close closes this end of the tunnel and so all existing connections
func (t *Tunnel) Close() error {
	defer log.Debugf("Tunnel: Closed")
	return t.mux.Close()
}

// Accept waits for a new connection, returns net.Conn or an error
func (t *Tunnel) Accept() (net.Conn, error) {
	log.Debugf("Tunnel: Accepting connections")
	defer log.Debugf("Tunnel: Accepted connection")
	return t.mux.Accept()
}

// AcceptStream waits for a new connection, returns io.ReadWriteCloser or an error
func (t *Tunnel) AcceptStream() (io.ReadWriteCloser, error) {
	log.Debugf("Tunnel: Accepting stream")
	defer log.Debugf("Tunnel: Accepted stream")
	return t.mux.AcceptStream()
}

// Addr returns the address of this tunnel sides endpoint.
func (t *Tunnel) Addr() net.Addr {
	a := addr{
		net: "voltron-tunnel",
	}

	if n, ok := t.stream.(net.Conn); ok {
		a.addr = n.LocalAddr().String()
	}

	return a
}

// Open opens a new net.Conn to the other side of the tunnel. Returns when
// the the new connection is set up
func (t *Tunnel) Open() (net.Conn, error) {
	c, err := t.mux.Open()
	t.checkErr(err)
	return c, err
}

// OpenStream returns, unlike NewConn, an io.ReadWriteCloser
func (t *Tunnel) OpenStream() (io.ReadWriteCloser, error) {
	s, err := t.mux.OpenStream()
	t.checkErr(err)
	return s, err
}

// Identity provides the identity of the remote side that initiated the tunnel
func (t *Tunnel) Identity() Identity {
	if id, ok := t.stream.(hasIdentity); ok {
		return id.Identity()
	}

	return nil
}

// WaitForError blocks as long as the tunnel exists and will return the reason
// why the tunnel exited
func (t *Tunnel) WaitForError() error {
	<-t.errCh
	return t.err
}

func (t *Tunnel) checkErr(err error) {
	if err != nil {
		t.errOnce.Do(func() {
			t.err = err
			close(t.errCh)
		})
	}
}

type serverCloser struct {
	io.ReadWriteCloser
	t *Tunnel
}

func (sc *serverCloser) Close() error {
	sc.t.checkErr(errors.Errorf("closed by multiplexer"))
	return sc.ReadWriteCloser.Close()
}

type addr struct {
	net  string
	addr string
}

func (a addr) Network() string {
	return a.net
}

func (a addr) String() string {
	return a.addr
}

// Dial returns a client side Tunnel or an error
func Dial(target string, opts ...Option) (*Tunnel, error) {
	c, err := net.Dial("tcp", target)
	if err != nil {
		return nil, errors.Errorf("tcp.Dial failed: %s", err)
	}

	return NewClientTunnel(c, opts...)
}

// DialTLS creates a TLS connection based on the config, must not be nil.
func DialTLS(target string, config *tls.Config, opts ...Option) (*Tunnel, error) {
	if config == nil {
		return nil, errors.Errorf("nil config")
	}

	c, err := tls.Dial("tcp", target, config)
	if err != nil {
		return nil, errors.Errorf("tcp.tls.Dial failed: %s", err)
	}

	return NewClientTunnel(c, opts...)
}
