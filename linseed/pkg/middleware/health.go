// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package middleware

import (
	"net/http"
	"strings"

	"github.com/projectcalico/calico/lma/pkg/httputils"
)

type state struct {
	Status string `json:"status"`
}

// HealthCheck returns 200 OK status code and a json response on GET request with the status
// The request will be passed to be served by the next middleware handler
func HealthCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		s := state{Status: "ok"}
		if req.Method == http.MethodGet && strings.EqualFold(req.URL.Path, "/health") {
			// write state and return 200 ok
			httputils.Encode(w, s)
			return
		}
		next.ServeHTTP(w, req)
	})
}
