// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

type state struct {
	Status string `json:"status"`
}

// HealthCheck returns 200 OK status code and a json response on GET request with the status
// The request will be passed to be served by the next middleware handler
func HealthCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var s = state{Status: "ok"}
		if req.Method == http.MethodGet && strings.EqualFold(req.URL.Path, "/health") {
			js, err := json.MarshalIndent(s, "", "  ")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(js)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, err = w.Write([]byte{'\n'})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// write state and return 200 ok
			return
		}
		next.ServeHTTP(w, req)

	})
}
