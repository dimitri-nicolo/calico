// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package middlewares

import "net/http"

func EnforceKibanaTenancy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
