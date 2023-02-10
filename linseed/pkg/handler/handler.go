// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package handler

import "net/http"

// Handler is a custom handler that defines what HTTP actions are provided when querying a resource
type Handler interface {
	// SupportedAPIs returns a mapping between supported methods and the internal handlers
	APIS() []API
}

// API represents a method, url, and handler combination.
type API struct {
	Method  string
	URL     string
	Handler http.Handler
}
