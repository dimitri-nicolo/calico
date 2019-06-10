// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package tunnel

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Server is a connection server that accepts connections from the provided
// Listener and provides the data streams for the tunnel.
type Server struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	streamC chan *ServerStream
}

// NewServer returns a new server
func NewServer() *Server {
	s := &Server{
		streamC: make(chan *ServerStream),
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	return s
}

// Serve start serving connections on the given Listener
func (s *Server) Serve(lis net.Listener) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		for {
			c, err := lis.Accept()
			if err != nil {
				s.cancel()
				return
			}

			ss := &ServerStream{
				Conn: c,
			}
			ss.ctx, ss.cancel = context.WithCancel(s.ctx)

			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				ss.watchServerStop()
			}()

			select {
			case s.streamC <- ss:
			case <-s.ctx.Done():
				return
			}
		}

	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-s.ctx.Done()
		lis.Close()
	}()

	return nil
}

// Accept returns the next available stream or returns an error
func (s *Server) Accept() (io.ReadWriteCloser, error) {
	select {
	case ss := <-s.streamC:
		return ss, nil
	case <-s.ctx.Done():
		return nil, errors.Errorf("server is exitting")
	}
}

// AcceptTunnel accepts a new connection as a tunnel
func (s *Server) AcceptTunnel() (*Tunnel, error) {
	c, err := s.Accept()
	if err != nil {
		return nil, err
	}

	return NewServerTunnel(c)
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
	ss.closeConn()
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
