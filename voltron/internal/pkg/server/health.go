// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type health struct {
}

// Determine which handler to execute based on HTTP method.
func (h *health) apiHandle(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
	switch r.Method {
	case http.MethodGet:
		returnJSON(w, "OK")
	default:
		http.NotFound(w, r)
	}
}
