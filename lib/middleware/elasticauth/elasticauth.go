// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elasticauth

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// BasicAuthHeaderInjector middleware replaces the Authorization HTTP header
// with the value of base64(user:password).
func BasicAuthHeaderInjector(user, password string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if user == "" || password == "||" {
			log.Error("Basic auth handler misconfigured")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		req.SetBasicAuth(user, password)
		h.ServeHTTP(w, req)
	})
}
