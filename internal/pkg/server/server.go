// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"net"
	"net/http"

	"github.com/pkg/errors"
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

	clusters *clusters

	certFile string
	keyFile  string
}

// New returns a new Server
func New(opts ...Option) (*Server, error) {
	srv := &Server{
		http: new(http.Server),
		clusters: &clusters{
			clusters: make(map[string]*cluster),
		},
	}

	for _, o := range opts {
		if err := o(srv); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	srv.proxyMux = http.NewServeMux()
	srv.http.Handler = srv.proxyMux

	/* XXX this will be replaced by https://github.com/tigera/voltron/pull/23
	srv.proxyMux.Handle("/", demuxproxy.New(
		demuxproxy.NewHeaderMatcher(
			srv.clusters.GetTargets(),
			ClusterHeaderField,
		),
	))
	*/
	srv.proxyMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "PROXYING DOES NOT WORK YET", 400)
	})

	srv.proxyMux.HandleFunc("/voltron/api/clusters", srv.clusters.handle)

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
