// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package handler

import "net/http"

// Handler is a custom handler that defines what HTTP actions are provided when querying a resource
type Handler interface {
	// Serve returns a function which processes requests sent to this handle
	Serve() http.HandlerFunc

	// URL will return the URL path defined to make queries
	URL() string

	// SupportedAPIs returns a mapping between supported methods and the internal handlers
	SupportedAPIs() map[string]http.Handler
}
