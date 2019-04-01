// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// PathValidator will check the request path of an request and reject
// ones that are not in the paths map.
// TODO(doublek):
//  - PathValidator should probably be less generic and store
//    parsed Elasticsearch index names in the request context for
//    use later on.
//  - Load paths from a file.
func PathValidator(paths map[string]struct{}, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		reqPath := req.URL.EscapedPath()
		if _, ok := paths[reqPath]; !ok {
			log.WithField("path", reqPath).Debug("Request path invalid")
			// TODO(doublek): Check the return code.
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.WithField("path", reqPath).Debug("Request path valid")
		h.ServeHTTP(w, req)
	})
}
