// Copyright (c) 2021 Tigera. All rights reserved.
package handler

import (
	"net/http"
	"net/http/httputil"
)

// Proxy sends the received query to the forwarded host registered in ReverseProxy param
func Proxy(proxy *httputil.ReverseProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		proxy.ServeHTTP(w, req)
	}
}
