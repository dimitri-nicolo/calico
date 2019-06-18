// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/textproto"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/pkg/tunnel"
)

const (
	// ClusterHeaderField represents the request header key used to determine
	// which cluster to proxy for
	ClusterHeaderField = "x-cluster-id"
)

var clusterHeaderFieldCanon = textproto.CanonicalMIMEHeaderKey(ClusterHeaderField)

// Server is the voltron server that accepts tunnels from the app clusters. It
// serves HTTP requests and proxies them to the tunnels.
type Server struct {
	ctx      context.Context
	cancel   context.CancelFunc
	http     *http.Server
	proxyMux *http.ServeMux

	clusters *clusters
	health   *health

	tunSrv *tunnel.Server

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

	srv.ctx, srv.cancel = context.WithCancel(context.Background())

	for _, o := range opts {
		if err := o(srv); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	srv.proxyMux = http.NewServeMux()
	srv.http.Handler = srv.proxyMux

	srv.proxyMux.HandleFunc("/", srv.clusterMuxer)
	srv.proxyMux.HandleFunc("/voltron/api/health", srv.health.apiHandle)
	srv.proxyMux.HandleFunc("/voltron/api/clusters", srv.clusters.apiHandle)

	srv.tunSrv = tunnel.NewServer()
	go srv.acceptTunnels()

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
	s.cancel()
	s.tunSrv.Stop()
	return s.http.Close()
}

// ServeTunnels starts serving tunnels using the provided listener
func (s *Server) ServeTunnels(lis net.Listener) error {
	err := s.tunSrv.Serve(lis)
	if err != nil {
		return errors.WithMessage(err, "ServeTunnels")
	}

	return nil
}

func (s *Server) acceptTunnels() {
	defer log.Debugf("acceptTunnels exited")

	for {
		t, err := s.tunSrv.AcceptTunnel()
		if err != nil {
			select {
			case <-s.ctx.Done():
				// N.B. When s.ctx.Done() AcceptTunnel will return with an
				// error, will not block
				return
			default:
				log.Errorf("accepting tunnel failed: %s", err)
				continue
			}
		}

		var clusterID string

		switch id := t.Identity().(type) {
		case net.Addr:
			clusterID = id.String()
		default:
			log.Errorf("unknown tunnel identity type %T", id)
		}

		c := s.clusters.get(clusterID)
		if c == nil {
			log.Errorf("cluster %q does not exist", clusterID)

			// XXX for now, we add a cluster eve if it does not exist
			c = new(cluster)
			c.ID = clusterID
			c.DisplayName = clusterID
			s.clusters.Lock()
			s.clusters.add(clusterID, c)
			s.clusters.Unlock()
			// XXX
		}

		c.assignTunnel(t)

		log.Debugf("Accepted a new tunnel from %s", clusterID)
	}
}

func (s *Server) clusterMuxer(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Header[clusterHeaderFieldCanon]; !ok {
		msg := fmt.Sprintf("missing %q header", ClusterHeaderField)
		log.Errorf("clusterMuxer: %s", msg)
		http.Error(w, msg, 400)
		return
	}

	if len(r.Header[clusterHeaderFieldCanon]) > 1 {
		msg := fmt.Sprintf("multiple %q headers", ClusterHeaderField)
		log.Errorf("clusterMuxer: %s", msg)
		http.Error(w, msg, 400)
		return
	}

	clusterID := r.Header.Get(ClusterHeaderField)

	c := s.clusters.get(clusterID)

	if c == nil {
		msg := fmt.Sprintf("Unknown target cluster %q", clusterID)
		log.Errorf("clusterMuxer: %s", msg)
		http.Error(w, msg, 400)
		return
	}

	r.URL.Scheme = "http"
	// N.B. Host is only set to make the ReverseProxy happy, DialContext ignores
	// this as the destinatination has been decided by choosing the tunnel.
	r.URL.Host = "voltron-tunnel"

	// TODO this is the place for the impersonation hook

	log.Debugf("tunneling %q from %q through %q", r.URL, r.RemoteAddr, clusterID)
	c.ServeHTTP(w, r)
	c.RUnlock()
}
