// Copyright 2019 Tigera Inc. All rights reserved.

package health

import (
	"errors"
	"net"
	"net/http"
	"os"
)

const HealthzSockPath = "/var/run/healthz.sock"
const HealthzSockDir = "/var/run/"

type Server struct {
	mux *http.ServeMux
	svr *http.Server
}

type Pinger interface {
	Ping()
}

type Readier interface {
	Ready() bool
}

type liveness struct {
	pinger Pinger
}

func (l *liveness) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.pinger.Ping()
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

func NewServer(p Pinger, r Readier) *Server {
	m := http.NewServeMux()
	m.Handle("/liveness", &liveness{pinger: p})
	m.Handle("/readiness", &readiness{readier: r})
	s := &Server{mux: m}
	return s
}

func (s *Server) Serve() error {
	err := os.MkdirAll(HealthzSockDir, 0777)
	if err != nil {
		return err
	}
	l, err := net.Listen("unix", HealthzSockPath)
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
