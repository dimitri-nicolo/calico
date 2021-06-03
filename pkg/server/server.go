package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/tigera/es-gateway/pkg/handlers/health"
	"github.com/tigera/es-gateway/pkg/middlewares"
	"github.com/tigera/es-gateway/pkg/proxy"
)

const (
	DefaultReadTimeout = 45 * time.Second
)

// Server is the ES Gateway server that accepts requests from various components that require
// access to Elasticsearch (& Kibana). It serves HTTP requests and proxies them Elasticsearch
// and Kibana.
type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	http   *http.Server

	internalCert tls.Certificate

	addr string
}

// New returns a new Server. k8s may be nil and options must check if it is nil
// or not if they set its user and return an error if it is nil
func New(esTarget, kibanaTarget *proxy.Target, opts ...Option) (*Server, error) {
	srv := &Server{}

	srv.ctx, srv.cancel = context.WithCancel(context.Background())
	for _, o := range opts {
		if err := o(srv); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	cfg := &tls.Config{}

	cfg.Certificates = append(cfg.Certificates, srv.internalCert)

	cfg.BuildNameToCertificate()

	// Set up HTTP routes (using Gorilla mux framework). Routes consist of a path and a handler function.
	router := mux.NewRouter()

	// Route Pattern 1: Handle the ES Gateway health check endpoint
	router.HandleFunc("/health", health.Health).Name("health")

	// Route Pattern 2: Handle any Kibana request, which we expect will have the provided path prefix.
	kibanaHandler, err := kibanaTarget.GetProxyHandler()
	if err != nil {
		return nil, err
	}
	// The below path prefix syntax provides us a wildcard to specify that kibanaHandler will handle all
	// requests with a path that begins with the given path prefix.
	router.PathPrefix(kibanaTarget.PathPrefix).HandlerFunc(kibanaHandler).Name("kibana")

	// Route Pattern 3: Handle any Elasticsearch request, which we treat as a catch all cases.
	esHandler, err := esTarget.GetProxyHandler()
	if err != nil {
		return nil, err
	}
	// We add a path prefix to enable wildcard. This base path is more generic and acts as a catch all.
	// Since it has been added last (after the previous routes) will will only match if the previous
	// routes did not match.
	router.PathPrefix(esTarget.PathPrefix).HandlerFunc(esHandler).Name("elasticsearch")

	// Add middlewares to the router
	router.Use(middlewares.LogRequests)

	srv.http = &http.Server{
		Addr:        srv.addr,
		Handler:     router,
		TLSConfig:   cfg,
		ReadTimeout: DefaultReadTimeout,
	}

	return srv, nil
}

// ListenAndServeHTTPS starts listening and serving HTTPS requests
func (s *Server) ListenAndServeHTTPS() error {
	return s.http.ListenAndServeTLS("", "")
}
