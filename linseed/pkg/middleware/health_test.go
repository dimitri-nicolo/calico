// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/projectcalico/calico/linseed/pkg/middleware"
)

func TestHealthCheck(t *testing.T) {
	type testResult struct {
		httpStatus int
		httpBody   string
	}

	type testInput struct {
		method string
		path   string
	}

	tests := []struct {
		name  string
		given testInput
		want  testResult
	}{
		{name: "Get HealthCheck", given: testInput{"GET", "/health"}, want: testResult{httpStatus: 200, httpBody: `{"status":"ok"}`}},
		{name: "Not matching path", given: testInput{"GET", "/healthz"}, want: testResult{httpStatus: 404, httpBody: `404 page not found`}},
		{name: "Not matching method", given: testInput{"POST", "/health"}, want: testResult{httpStatus: 404, httpBody: `404 page not found`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			})
			healthCheck := middleware.HealthCheck(testHandler)

			rec := httptest.NewRecorder()
			req, err := http.NewRequest(tt.given.method, tt.given.path, nil)
			assert.NoError(t, err)

			healthCheck.ServeHTTP(rec, req)

			bodyBytes, err := io.ReadAll(rec.Body)
			assert.NoError(t, err)

			assert.Equal(t, tt.want.httpStatus, rec.Result().StatusCode)
			if tt.want.httpStatus != 200 {
				assert.Equal(t, tt.want.httpBody, strings.Trim(string(bodyBytes), "\n"))
			} else {
				assert.JSONEq(t, tt.want.httpBody, string(bodyBytes))
			}
		})
	}
}
