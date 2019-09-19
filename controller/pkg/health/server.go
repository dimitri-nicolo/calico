// Copyright 2019 Tigera Inc. All rights reserved.

package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

const DefaultHealthzSockPort = 50000

type Server struct {
	mux  *http.ServeMux
	svr  *http.Server
	port int
}

type Pinger interface {
	Ping(context.Context) error
}

type Readier interface {
	Ready() bool
}

type liveness struct {
	pinger Pinger
}

func (l *liveness) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := l.pinger.Ping(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type readiness struct {
	readier Readier
}

type AlwaysReady struct{}

func (a AlwaysReady) Ready() bool { return true }

func (r *readiness) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if r.readier.Ready() {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func NewServer(p Pinger, r Readier, healthzSockPort int) *Server {
	m := http.NewServeMux()
	m.Handle("/liveness", &liveness{pinger: p})
	m.Handle("/readiness", &readiness{readier: r})
	s := &Server{
		mux:  m,
		port: healthzSockPort,
	}
	return s
}

func (s *Server) Serve() error {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.port))
	if err != nil {
		return err
	}
	s.svr = &http.Server{Handler: s.mux}
	return s.svr.Serve(l)
}

func (s *Server) Close() error {
	if s.svr == nil {
		return errors.New("close on server that isn't serving")
	}
	return s.svr.Close()
}
