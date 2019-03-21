// Copyright 2019 Tigera Inc. All rights reserved.

package health

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

const DefaultHealthzSockPath = "/var/run/healthz.sock"

type Server struct {
	mux             *http.ServeMux
	svr             *http.Server
	healthzSockPath string
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

func (r *readiness) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if r.readier.Ready() {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func NewServer(p Pinger, r Readier, healthzSockPath string) *Server {
	m := http.NewServeMux()
	m.Handle("/liveness", &liveness{pinger: p})
	m.Handle("/readiness", &readiness{readier: r})
	s := &Server{
		mux:             m,
		healthzSockPath: healthzSockPath,
	}
	return s
}

func (s *Server) Serve() error {
	healthzSockDir := filepath.Dir(s.healthzSockPath)
	err := os.MkdirAll(healthzSockDir, 0777)
	if err != nil {
		return err
	}
	l, err := net.Listen("unix", s.healthzSockPath)
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
