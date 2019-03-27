// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"net/http"
	"net/http/httputil"

	log "github.com/sirupsen/logrus"
)

// LogRequestHeaders is a sample Handler/Middleware that will log the headers
// of an incoming request.
// "Middlewares" are just handlers that also return a handler. Inspired by
// various utility handlers in the standard library net/http package.
func LogRequestHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		reqB, err := httputil.DumpRequest(req, true)
		if err == nil {
			log.Infof("Request dump %v", string(reqB))
		} else {
			log.Infof("Couldn't dump request.")
		}
		h.ServeHTTP(w, req)
	})
}
