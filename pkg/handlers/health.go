// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handlers

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type state struct {
	Status string `json:"status"`
}

// Health is a health handler that will return 200 OK status code and a json response
// on GET request. A 404 FileNotFound will be returned if other method is used
func Health(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s for %s from %s", r.Method, r.URL, r.RemoteAddr)
	var s = state{Status: "Ok"}

	switch r.Method {
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
		http.NotFound(w, r)
	}
}
