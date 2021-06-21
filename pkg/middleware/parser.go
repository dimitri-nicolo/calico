// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	lmak8s "github.com/tigera/lma/pkg/k8s"

	log "github.com/sirupsen/logrus"
)

// QueryString represents the expected query for the download API
const QueryString = "files.zip"

// MalformedRequest is the error message when the API received an invalid request
var MalformedRequest = fmt.Errorf("request URL is malformed")

// Parse is a middleware handler that parses the request and sets the common attributes
// on the its context
func Parse(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var ns, name, err = parse(req.URL)
		if err != nil {
			log.WithError(err).Errorf("Invalid request %s", req.URL)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var clusterID = req.Header.Get(lmak8s.XClusterIDHeader)
		if clusterID == "" {
			clusterID = lmak8s.DefaultCluster
		}

		req = req.WithContext(WithCaptureName(req.Context(), name))
		req = req.WithContext(WithNamespace(req.Context(), ns))
		req = req.WithContext(WithClusterID(req.Context(), clusterID))
		handlerFunc.ServeHTTP(w, req)
	}
}

func parse(url *url.URL) (string, string, error) {
	var tokens = strings.Split(url.Path, "/")
	if len(tokens) != 4 {
		return "", "", MalformedRequest
	}
	if url.RawQuery != QueryString {
		return "", "", MalformedRequest
	}
	return tokens[2], tokens[3], nil
}
