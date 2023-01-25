// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package server

import (
	"fmt"
	"net/http"

	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/go-chi/chi/v5"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/projectcalico/calico/linseed/pkg/handler"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// Route defines the server response based on the method and pattern of the request
type Route struct {
	method  string
	pattern string
	handler http.Handler
}

// UnpackRoutes will create routes based on the methods supported for the provided handlers
func UnpackRoutes(handlers ...handler.Handler) []Route {
	var routes []Route

	for _, h := range handlers {
		for k, v := range h.SupportedAPIs() {
			routes = append(routes, []Route{
				{k, h.URL(), v},
			}...)
		}
	}

	return routes
}

// UtilityRoutes defines all available utility routes
func UtilityRoutes() []Route {
	return []Route{
		{"GET", "/version", handler.VersionCheck()},
		{"GET", "/metrics", promhttp.Handler()},
	}
}

// Middlewares defines all available intermediary handlers
func Middlewares(cfg config.Config) []func(http.Handler) http.Handler {
	clusterInfo := middleware.NewClusterInfo(cfg.ExpectedTenantID)
	return []func(http.Handler) http.Handler{
		// LogRequestHeaders needs to be placed before any middlewares that mutate the request
		httputils.LogRequestHeaders,
		// HealthCheck is defined as middleware in order to bypass any route matching
		middleware.HealthCheck,
		// ClusterInfoOld will extract cluster and tenant information from the request to identify the caller
		clusterInfo.Extract(),
	}
}

// Option will configure a Server with different options
type Option func(*Server) error

// WithAPIVersionRoutes will add to the internal router the desired routes to the api version
func WithAPIVersionRoutes(apiVersion string, routes ...Route) Option {
	return func(s *Server) error {
		if s.router == nil {
			return fmt.Errorf("default server is missing a router")
		}

		s.router.Route(apiVersion, func(r chi.Router) {
			for _, route := range routes {
				r.Method(route.method, route.pattern, route.handler)
			}
		})

		return nil
	}
}

// WithRoutes will add to the internal router the desired routes
func WithRoutes(routes ...Route) Option {
	return func(s *Server) error {
		if s.router == nil {
			return fmt.Errorf("default server is missing a router")
		}

		for _, route := range routes {
			s.router.Method(route.method, route.pattern, route.handler)
		}

		return nil
	}
}

// WithMiddlewares will instruct the internal router to make use of the desired middlewares
func WithMiddlewares(middlewares []func(http.Handler) http.Handler) Option {
	return func(s *Server) error {
		if s.router == nil {
			return fmt.Errorf("default server is missing a router")
		}

		s.router.Use(middlewares...)

		return nil
	}
}
