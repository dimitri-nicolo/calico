// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Package tunnel defines an authenticated tunnel API, that allows creating byte
// pipes in both directions, initiated from either side of the tunnel.
package tunnel

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/projectcalico/calico/voltron/internal/pkg/utils"

	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ErrTunnelClosed is used to notify a caller that an action can't proceed because the tunnel is closed
var ErrTunnelClosed = fmt.Errorf("tunnel closed")

// DialInRoutineWithTimeout calls dialer.Dial() in a routine and sends the result back on the given resultsChan. The
// timeout given is not the timeout for dialing (the implementation of the Dialer needs to take care of that), but used
// to timeout writing to the resultsChan in the event that the channel is blocked.
//
// The channel return is needed to signal the routine that we no longer need the result. This channel should be closed
// to send that signal, and is the responsibility of the caller to close that channel regardless.
func DialInRoutineWithTimeout(dialer Dialer, resultsChan chan interface{}, timeout time.Duration) chan struct{} {
	closeChan := make(chan struct{})

	go func() {
		defer close(resultsChan)

		log.Debug("Dialing tunnel")
		tun, err := dialer.Dial()

		var result interface{}
		if err != nil {
			result = err
		} else {
			result = tun
		}

		timer := time.NewTimer(timeout)
		defer timer.Stop()

		select {
		case <-closeChan:
			log.Debug("Received signal to close, dropping dialing result.")
			return
		case resultsChan <- result:
		case <-timer.C:
			log.Error("Timed out trying to send the result over the results channel")
			return
		}

		log.Debug("Finished dialing tunnel.")
	}()

	return closeChan
}

// Dialer is an interface that supports dialing to create a *Tunnel
type Dialer interface {
	Dial() (*Tunnel, error)
	Timeout() time.Duration
}

type dialer struct {
	dialerFun       DialerFunc
	retryAttempts   int
	retryInterval   time.Duration
	timeout         time.Duration
	fipsModeEnabled bool
}

// NewDialer creates a new Dialer.
func NewDialer(dialerFunc DialerFunc, retryAttempts int, retryInterval time.Duration, timeout time.Duration, fipsModeEnabled bool) Dialer {
	return &dialer{
		dialerFun:       dialerFunc,
		retryAttempts:   retryAttempts,
		retryInterval:   retryInterval,
		timeout:         timeout,
		fipsModeEnabled: fipsModeEnabled,
	}
}

func (d *dialer) Timeout() time.Duration {
	return d.timeout
}

func (d *dialer) Dial() (*Tunnel, error) {
	var err error
	for i := 0; i < d.retryAttempts; i++ {
		t, err := d.dialerFun()
		if err != nil {
			var xerr x509.UnknownAuthorityError
			if errors.As(err, &xerr) {
				log.WithError(err).Infof("tcp.tls.Dial failed: %s. fingerprint='%s' issuerCommonName='%s' subjectCommonName='%s'", xerr.Error(), utils.GenerateFingerprint(d.fipsModeEnabled, xerr.Cert), xerr.Cert.Issuer.CommonName, xerr.Cert.Subject.CommonName)
			} else {
				log.WithError(err).Infof("dial attempt %d failed, will retry in %s", i, d.retryInterval.String())
			}
			time.Sleep(d.retryInterval)
			continue
		}
		return t, nil
	}

	return nil, err
}

// DialerFunc is a function type used to create a tunnel
type DialerFunc func() (*Tunnel, error)

// Tunnel represents either side of the tunnel that allows waiting for,
// accepting and initiating creation of new BytePipes.
type Tunnel struct {
	stream io.ReadWriteCloser
	mux    *yamux.Session

	errOnce sync.Once
	errCh   chan struct{}
	LastErr error

	keepAliveEnable   bool
	keepAliveInterval time.Duration
	DialTimeout       time.Duration
}

func newTunnel(stream io.ReadWriteCloser, isServer bool, opts ...Option) (*Tunnel, error) {
	t := &Tunnel{
		stream: stream,
		errCh:  make(chan struct{}),
		// Defaults
		keepAliveEnable: true,

		keepAliveInterval: 100 * time.Millisecond,
		DialTimeout:       60 * time.Second,
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
		return nil, errors.Errorf("New failed creating muxer: %s", err)
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
	return convertYAMUXErr(t.mux.Close())
}

// IsClosed checks if the tunnel is closed. If it is true is returned, otherwise false is returned
func (t *Tunnel) IsClosed() bool {
	return t.mux.IsClosed()
}

// Accept waits for a new connection, returns net.Conn or an error
func (t *Tunnel) Accept() (net.Conn, error) {
	log.Debugf("Tunnel: Accepting connections")
	defer log.Debugf("Tunnel: Accepted connection")
	conn, err := t.mux.Accept()
	return conn, convertYAMUXErr(err)
}

// AcceptWithChannel takes a channel of ConnWithError, kicks of a go routine that starts accepting connection, and sends
// any connections received to the given channel. The channel returned from calling this function is used to signal that
// we're done accepting connections.
//
// If the tunnel hasn't been setup prior to calling this function it will panic.
func (t *Tunnel) AcceptWithChannel(acceptChan chan interface{}) chan bool {
	a := acceptChan
	done := make(chan bool)
	go func() {
		log.Debug("tunnel writing connections to new channel")
		defer func() {
			log.Debug("tunnel finished writing connections to new channel")
			close(a)
		}()

		for {
			conn, err := t.mux.Accept()
			select {
			case <-done:
				return
			default:
			}
			if err == nil {
				a <- conn
			} else {
				err = convertYAMUXErr(err)
				// if the tunnel is closed we're done
				if err == ErrTunnelClosed {
					return
				}

				a <- err
			}
		}
	}()

	return done
}

// AcceptStream waits for a new connection, returns io.ReadWriteCloser or an error
func (t *Tunnel) AcceptStream() (io.ReadWriteCloser, error) {
	log.Debugf("Tunnel: Accepting stream")
	defer log.Debugf("Tunnel: Accepted stream")
	rc, err := t.mux.AcceptStream()
	return rc, convertYAMUXErr(err)
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
	err = convertYAMUXErr(err)
	t.checkErr(err)
	return c, err
}

// OpenStream returns, unlike NewConn, an io.ReadWriteCloser
func (t *Tunnel) OpenStream() (io.ReadWriteCloser, error) {
	s, err := t.mux.OpenStream()
	err = convertYAMUXErr(err)
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
	return t.LastErr
}

// ErrChan returns the channel that's notified when an error occurs
func (t *Tunnel) ErrChan() chan struct{} {
	return t.errCh
}

func (t *Tunnel) checkErr(err error) {
	if err != nil {
		t.errOnce.Do(func() {
			t.LastErr = err
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
		return nil, errors.Errorf("tcp.Dial failed: %v", err)
	}

	return NewClientTunnel(c, opts...)
}

// DialTLS creates a TLS connection based on the config, must not be nil.
func DialTLS(target string, config *tls.Config, timeout time.Duration, opts ...Option) (*Tunnel, error) {
	if config == nil {
		return nil, errors.Errorf("nil config")
	}

	log.Debugf("Starting TLS dial to %s with a timeout of %v", target, timeout)

	// We need to explicitly set the timeout as it seems it's possible for this to hang indefinitely if we
	// don't.
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	c, err := tls.DialWithDialer(dialer, "tcp", target, config)
	log.Debugf("Finished TLS dial to %s", target)
	if err != nil {
		return nil, fmt.Errorf("tcp.tls.Dial failed: %w", err)
	}

	return NewClientTunnel(c, append(opts, WithDialTimeout(timeout))...)
}

// We don't want to / need to expose that we're using the yamux library
func convertYAMUXErr(err error) error {
	switch err {
	case yamux.ErrSessionShutdown:
		return ErrTunnelClosed
	case io.EOF:
		return ErrTunnelClosed
	}

	return err
}
