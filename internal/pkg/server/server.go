// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/tigera/voltron/internal/pkg/clusters"
	demuxproxy "github.com/tigera/voltron/internal/pkg/proxy"
)

const (
	// ClusterHeaderField represents the request header key used to determine
	// which cluster to proxy for
	ClusterHeaderField = "x-cluster-id"
)

// Server is the voltron server that accepts tunnels from the app clusters. It
// serves HTTP requests and proxies them to the tunnels.
type Server struct {
	http     *http.Server
	proxyMux *http.ServeMux
	clusters *clusters.Clusters

	certFile string
	keyFile  string
}

// New returns a new Server
func New(opts ...Option) (*Server, error) {
	srv := &Server{
		http:     new(http.Server),
		clusters: clusters.New(),
	}

	for _, o := range opts {
		if err := o(srv); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	log.Infof("Targets are: %s", srv.clusters.GetTargets())
	srv.proxyMux = http.NewServeMux()
	srv.http.Handler = srv.proxyMux

	srv.proxyMux.Handle("/", demuxproxy.New(
		demuxproxy.NewHeaderMatcher(
			srv.clusters.GetTargets(),
			ClusterHeaderField,
		),
	))
	proxyHandler := clusterHandler{clusters: srv.clusters}
	srv.proxyMux.HandleFunc("/voltron/api/clusters", proxyHandler.handle)

	return srv, nil
}

// ListenAndServeHTTP starts listening and serving HTTP requests
func (s *Server) ListenAndServeHTTP() error {
	return s.http.ListenAndServe()

}

// ServeHTTP starts serving HTTP requests
func (s *Server) ServeHTTP(lis net.Listener) error {
	return s.http.Serve(lis)
}

// ListenAndServeHTTPS starts listening and serving HTTPS requests
func (s *Server) ListenAndServeHTTPS() error {
	return s.http.ListenAndServeTLS(s.certFile, s.keyFile)
}

// ServeHTTPS starts serving HTTPS requests
func (s *Server) ServeHTTPS(lis net.Listener) error {
	return s.http.ServeTLS(lis, s.certFile, s.keyFile)
}

// Close stop the server
func (s *Server) Close() error {
	return s.http.Close()
}
