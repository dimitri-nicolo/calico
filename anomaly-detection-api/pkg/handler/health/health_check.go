// Copyright (c) 2022 Tigera. All rights reserved.
package health

import (
	"fmt"
	"net/http"
)

// HealthCheck is a handler that serves the /health endpoint for liveness and readiness checks
func HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "UP")
	}
}
