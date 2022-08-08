// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middlewares

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func logRequestHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("ES Gateway received request for URI %s", r.RequestURI)
		next.ServeHTTP(w, r)
	})
}
