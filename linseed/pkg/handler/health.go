// Copyright 2022 Tigera. All rights reserved.

package handler

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type state struct {
	Status string `json:"status"`
}

// HealthCheck returns 200 OK status code and a json response on GET request with the status
// A 404 FileNotFound will be returned if other method is used.
func HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Tracef("%s for %s from %s", req.Method, req.URL, req.RemoteAddr)
		var s = state{Status: "ok"}

		switch req.Method {
		case http.MethodGet:
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
		default:
			http.NotFound(w, req)
		}
	}
}
