// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// LogRequestHeaders is a sample Handler/Middleware that will log some headers
// of an incoming request.
// "Middlewares" are just handlers that also return a handler. Inspired by
// various utility handlers in the standard library net/http package.
func LogRequestHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.WithFields(log.Fields{
			"method":     req.Method,
			"path":       req.RequestURI,
			"remoteAddr": req.RemoteAddr,
		}).Infof("Request received")
		h.ServeHTTP(w, req)
	})
}
