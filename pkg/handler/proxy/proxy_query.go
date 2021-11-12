// Copyright (c) 2021 Tigera. All rights reserved.
package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/tigera/lma/pkg/auth"

	authzv1 "k8s.io/api/authorization/v1"
)

var (
	// The RBAC permissions that allow a user to perform an HTTP GET to Prometheus.
	getResources = []*authzv1.ResourceAttributes{
		{
			Verb:     "get",
			Resource: "services/proxy",
			Name:     "https:tigera-api:8080",
		},
		{
			Verb:     "get",
			Resource: "services/proxy",
			Name:     "calico-node-prometheus:9090",
		},
	}
	// The RBAC permissions that allow a user to perform an HTTP POST to Prometheus.
	createResources = []*authzv1.ResourceAttributes{
		{
			Verb:     "create",
			Resource: "services/proxy",
			Name:     "https:tigera-api:8080",
		},
		{
			Verb:     "create",
			Resource: "services/proxy",
			Name:     "calico-node-prometheus:9090",
		},
	}
)

// Proxy sends the received query to the forwarded host registered in ReverseProxy param
func Proxy(proxy *httputil.ReverseProxy, authnAuthz auth.AuthNAuthZ) (http.HandlerFunc, error) {
	return func(w http.ResponseWriter, req *http.Request) {
		usr, stat, err := authnAuthz.Authenticate(req)
		if err != nil {
			w.WriteHeader(stat)
			w.Write([]byte(err.Error()))
			return
		}

		// Perform AuthZ checks
		var resources []*authzv1.ResourceAttributes
		if req.Method == http.MethodGet {
			resources = getResources
		} else if req.Method == http.MethodPost {
			resources = createResources
		} else {
			// At this time only HTTP GET/POST are allowed
			w.WriteHeader(405)
			w.Write([]byte("Method Not Allowed"))
			return
		}

		authorized := false
		// Check if either of the permissions are allowed, then the user is authorized.
		for _, res := range resources {
			ok, err := authnAuthz.Authorize(usr, res)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}
			if ok {
				authorized = true
				break
			}
		}

		if !authorized {
			w.WriteHeader(403)
			w.Write([]byte(fmt.Sprintf("user %v is not authorized to perform %v https:tigera-api:8080", usr, req.Method)))
			return
		}

		proxy.ServeHTTP(w, req)
	}, nil
}
