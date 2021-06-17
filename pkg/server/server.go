package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-gateway/pkg/elastic"
	"github.com/tigera/es-gateway/pkg/handlers/gateway"
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

	http         *http.Server    // Server to accept incoming ES/Kibana requests
	addr         string          // Address for server to listen on
	internalCert tls.Certificate // Certificate chain used for all external requests

	// Elasticsearch client for making API calls required by ES Gateway
	esClient elastic.Client
}

// New returns a new ES Gateway server. Validate and set the server options. Set up the Elasticsearch and Kibana
// related routes and HTTP handlers.
func New(esTarget, kibanaTarget *proxy.Target, opts ...Option) (*Server, error) {
	var err error
	srv := &Server{}
	srv.ctx, srv.cancel = context.WithCancel(context.Background())

	// -----------------------------------------------------------------------------------------------------
	// Parse server options
	// -----------------------------------------------------------------------------------------------------
	for _, o := range opts {
		if err := o(srv); err != nil {
			return nil, errors.WithMessage(err, "applying option failed")
		}
	}

	cfg := &tls.Config{}
	cfg.Certificates = append(cfg.Certificates, srv.internalCert)
	cfg.BuildNameToCertificate()

	// -----------------------------------------------------------------------------------------------------
	// Set up all routing for ES Gateway server (using Gorilla Mux).
	// -----------------------------------------------------------------------------------------------------
	router := mux.NewRouter()
	handlers := middlewares.HandlerMap{
		middlewares.HandlerTypeAuth: middlewares.NewAuthMiddleware(srv.esClient),
	}

	// Route Handling #1: Handle the ES Gateway health check endpoint
	router.HandleFunc("/health", health.Health).Name("health")

	// Route Handling #2: Handle any Kibana request, which we expect will have a common path prefix.
	kibanaHandler, err := gateway.GetProxyHandler(kibanaTarget)
	if err != nil {
		return nil, err
	}
	// The below path prefix syntax provides us a wildcard to specify that kibanaHandler will handle all
	// requests with a path that begins with the given path prefix.
	err = addRoutes(
		router,
		kibanaTarget.Routes,
		kibanaTarget.CatchAllRoute,
		handlers,
		http.HandlerFunc(kibanaHandler),
	)
	if err != nil {
		return nil, err
	}

	// Route Handling #3: Handle any Elasticsearch request. We do the Elasticsearch section last because
	// these routes do not have a universally common path prefix.
	esHandler, err := gateway.GetProxyHandler(esTarget)
	if err != nil {
		return nil, err
	}
	err = addRoutes(
		router,
		esTarget.Routes,
		esTarget.CatchAllRoute,
		handlers,
		http.HandlerFunc(esHandler),
	)
	if err != nil {
		return nil, err
	}

	// Add common middlewares to the router.
	router.Use(middlewares.LogRequests)

	// -----------------------------------------------------------------------------------------------------
	// Return configured ES Gateway server.
	// -----------------------------------------------------------------------------------------------------
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

// addRoutes sets up the given Routes for the provided mux.Router.
func addRoutes(router *mux.Router, routes proxy.Routes, catchAllRoute *proxy.Route, h middlewares.HandlerMap, f http.Handler) error {
	// Set up provided list of Routes
	for _, route := range routes {
		muxRoute := router.NewRoute()
		finalHandler := f

		// Create a wrapping handler that will log the route name when executed.
		wrapper := getHandlerWrapper(route.Name)

		// If this Route has HTTP methods to filter on, then add those.
		if len(route.HTTPMethods) > 0 {
			muxRoute.Methods(route.HTTPMethods...)
		}

		if route.RequireAuth {
			finalHandler = h[middlewares.HandlerTypeAuth](finalHandler)
		}

		if route.IsPathPrefix {
			muxRoute.PathPrefix(route.Path).Handler(wrapper(finalHandler)).Name(route.Name)
		} else {
			muxRoute.Path(route.Path).Handler(wrapper(finalHandler)).Name(route.Name)
		}
	}

	// Set up provided catch-all Route
	if catchAllRoute != nil {
		finalHandler := f

		if !catchAllRoute.IsPathPrefix {
			return errors.Errorf("catch-all route must be marked as a path prefix")
		}

		if catchAllRoute.RequireAuth {
			finalHandler = h[middlewares.HandlerTypeAuth](finalHandler)
		}

		// Create a wrapping handler that will log the route name when executed.
		wrapper := getHandlerWrapper(catchAllRoute.Name)
		router.PathPrefix(catchAllRoute.Path).Handler(wrapper(finalHandler)).Name(catchAllRoute.Name)
	}

	return nil
}

// getHandlerWrapper returns a function that wraps a http.Handler in order to log the Mux route that was
// matched. This is useful for troubleshooting and debugging.
func getHandlerWrapper(routeName string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Debugf("Request %s as been matched with route \"%s\"", r.RequestURI, routeName)
			h.ServeHTTP(w, r)
		})
	}
}
