// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middlewares

import (
	"bytes"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func logRequestHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("ES Gateway received request for URI %s", r.RequestURI)
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Error reading request body: %v", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Infof("logrequesthandler, Request body length: %v", len(string(buf)))

		reader := ioutil.NopCloser(bytes.NewBuffer(buf))
		r.Body = reader
		next.ServeHTTP(w, r)
	})
}
