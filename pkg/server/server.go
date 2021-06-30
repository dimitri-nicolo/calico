package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-gateway/pkg/clients/elastic"
	"github.com/tigera/es-gateway/pkg/clients/kibana"
	"github.com/tigera/es-gateway/pkg/clients/kubernetes"
	"github.com/tigera/es-gateway/pkg/handlers/gateway"
	"github.com/tigera/es-gateway/pkg/handlers/health"
	mid "github.com/tigera/es-gateway/pkg/middlewares"
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

	esTarget     *proxy.Target // Proxy target for Elasticsearch API
	kibanaTarget *proxy.Target // Proxy target for Kibana API

	esClient  elastic.Client    // Elasticsearch client for making API calls required by ES Gateway
	kbClient  kibana.Client     // Kibana client for making API calls required by ES Gateway
	k8sClient kubernetes.Client // K8s client for retrieving K8s resources related to ES users

	adminESUsername string // Used to store the username for a real ES admin user
	adminESPassword string // Used to store the password for a real ES admin user
}

// New returns a new ES Gateway server. Validate and set the server options. Set up the Elasticsearch and Kibana
// related routes and HTTP handlers.
func New(opts ...Option) (*Server, error) {
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
	middlewares := mid.GetHandlerMap(
		srv.esClient,
		srv.k8sClient,
		srv.adminESUsername,
		srv.adminESPassword,
	)

	// Route Handling #1: Handle the ES Gateway health check endpoint
	healthHandler := health.GetHealthHandler(srv.esClient, srv.kbClient, srv.k8sClient)
	router.HandleFunc("/health", healthHandler).Name("health")

	// Route Handling #2: Handle any Kibana request, which we expect will have a common path prefix.
	kibanaHandler, err := gateway.GetProxyHandler(srv.kibanaTarget)
	if err != nil {
		return nil, err
	}
	// The below path prefix syntax provides us a wildcard to specify that kibanaHandler will handle all
	// requests with a path that begins with the given path prefix.
	err = addRoutes(
		router,
		srv.kibanaTarget.Routes,
		srv.kibanaTarget.CatchAllRoute,
		middlewares,
		http.HandlerFunc(kibanaHandler),
	)
	if err != nil {
		return nil, err
	}

	// Route Handling #3: Handle any Elasticsearch request. We do the Elasticsearch section last because
	// these routes do not have a universally common path prefix.
	esHandler, err := gateway.GetProxyHandler(srv.esTarget)
	if err != nil {
		return nil, err
	}
	err = addRoutes(
		router,
		srv.esTarget.Routes,
		srv.esTarget.CatchAllRoute,
		middlewares,
		http.HandlerFunc(esHandler),
	)
	if err != nil {
		return nil, err
	}

	// Add common middlewares to the router.
	router.Use(middlewares[mid.TypeLog])

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
func addRoutes(router *mux.Router, routes proxy.Routes, catchAllRoute *proxy.Route, h mid.HandlerMap, f http.Handler) error {
	// Set up provided list of Routes
	for _, route := range routes {
		muxRoute := router.NewRoute()

		// If this Route has HTTP methods to filter on, then add those.
		if len(route.HTTPMethods) > 0 {
			muxRoute.Methods(route.HTTPMethods...)
		}

		handlerChain := buildMiddlewareChain(&route, h, f)
		if route.IsPathPrefix {
			muxRoute.PathPrefix(route.Path).Handler(handlerChain).Name(route.Name)
		} else {
			muxRoute.Path(route.Path).Handler(handlerChain).Name(route.Name)
		}
	}

	// Set up provided catch-all Route
	if catchAllRoute != nil {
		if !catchAllRoute.IsPathPrefix {
			return errors.Errorf("catch-all route must be marked as a path prefix")
		}

		handlerChain := buildMiddlewareChain(catchAllRoute, h, f)
		router.PathPrefix(catchAllRoute.Path).Handler(handlerChain).Name(catchAllRoute.Name)
	}

	return nil
}

// getLogRouteMatchHandler returns a function that wraps a http.Handler in order to log the Mux route that was
// matched. This is useful for troubleshooting and debugging.
func getLogRouteMatchHandler(routeName string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Debugf("Request %s as been matched with route \"%s\"", r.RequestURI, routeName)
			h.ServeHTTP(w, r)
		})
	}
}

// buildMiddlewareChain takes a proxy.Route and builds a chain of middleware handlers including the final
// HTTP handler f to be executed for the given route r. Which middlewares are included depends on r's
// configuration flags.
// When applying the chain, the last handler is applied to f first (since the chain is built outwards).
// This will ensure that the handlers are executed in the correct order for a request (ending with f).
// So if chain = {m1, m2, m3}, then we apply them on f, like this m1(m2(m3(f))). And the order of execution
// will be m1, m2, m3, f.
func buildMiddlewareChain(r *proxy.Route, h mid.HandlerMap, f http.Handler) http.Handler {
	chain := []mux.MiddlewareFunc{}

	// Add a wrapping handler that will log the route name when executed.
	wrapper := getLogRouteMatchHandler(r.Name)
	chain = append(chain, wrapper)

	// Add auth middleware to the chain for this Route, if the flag is enabled.
	if r.RequireAuth {
		chain = append(chain, h[mid.TypeAuth])

		// Alongside auth, add credential swapping middlware to the Handler chain for this
		// Route, depending on whether thie Route allows skipping the swap.
		if r.AllowSwapSkip {
			chain = append(chain, h[mid.TypeSwapAllowSkip])
		} else {
			chain = append(chain, h[mid.TypeSwap])
		}
	}

	// Now apply the chain of middleware handlers on the given route handler f, starting with the last one.
	finalHandler := f
	for i := range chain {
		finalHandler = chain[len(chain)-1-i](finalHandler)
	}

	return finalHandler
}
