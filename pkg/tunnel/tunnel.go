// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Package tunnel defines an authenticated tunnel API, that allows creating byte
// pipes in both directions, initiated from either side of the tunnel.
package tunnel

import (
	"io"
	"net"
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
}

func newTunnel(stream io.ReadWriteCloser, isServer bool) (*Tunnel, error) {
	t := &Tunnel{
		stream: stream,
	}

	var mux *yamux.Session
	var err error

	// XXX all the config options should probably become options taken by New()
	// XXX that can override the defaults set here
	config := yamux.DefaultConfig()
	config.AcceptBacklog = 1
	config.EnableKeepAlive = true
	config.KeepAliveInterval = 100 * time.Millisecond

	if isServer {
		mux, err = yamux.Server(stream, config)
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
func NewServerTunnel(stream io.ReadWriteCloser) (*Tunnel, error) {
	return newTunnel(stream, true)
}

// NewClientTunnel returns a new tunnel that uses the provided stream as the
// carrier. The stream must be the client side of the stream
func NewClientTunnel(stream io.ReadWriteCloser) (*Tunnel, error) {
	return newTunnel(stream, false)
}

// Identity represents remote peer identity
// XXX the exact type TBD
type Identity = interface{}

type hasIndentity interface {
	Identity() Identity
}

// Close closes this end of the tunnel and so all existing connections
func (t *Tunnel) Close() error {
	defer log.Debugf("Tunnel: Closed")
	return t.mux.Close()
}

// Accept waits for a new connection, returns net.Conn or an error
func (t *Tunnel) Accept() (net.Conn, error) {
	log.Debugf("Tunnel: Accepting")
	defer log.Debugf("Tunnel: Accepted")
	return t.mux.Accept()
}

// AcceptStream waits for a new connection, returns io.ReadWriteCloser or an error
func (t *Tunnel) AcceptStream() (io.ReadWriteCloser, error) {
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
	return t.mux.Open()
}

// OpenStream returns, unlike NewConn, an io.ReadWriteCloser
func (t *Tunnel) OpenStream() (io.ReadWriteCloser, error) {
	return t.mux.OpenStream()
}

// Identity provides the identity of the remote side that initiated the tunnel
func (t *Tunnel) Identity() Identity {
	if id, ok := t.stream.(hasIndentity); ok {
		return id.Identity()
	}

	return nil
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
func Dial(target string) (*Tunnel, error) {
	c, err := net.Dial("tcp", target)
	if err != nil {
		return nil, errors.Errorf("tcp.Dial failed: %s", err)
	}

	return NewClientTunnel(c)
}
