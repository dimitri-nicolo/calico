// Copyright 2022 Tigera. All rights reserved.

package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/projectcalico/calico/linseed/pkg/handler"
)

func TestHealthCheck(t *testing.T) {
	type testResult struct {
		httpStatus int
		httpBody   string
	}

	type testInput struct {
		method string
	}

	tests := []struct {
		name  string
		given testInput
		want  testResult
	}{
		{name: "200 OK", given: testInput{"GET"}, want: testResult{httpStatus: 200, httpBody: `{"status":"ok"}`}},
		{name: "404 Not found", given: testInput{"POST"}, want: testResult{httpStatus: 404, httpBody: `404 page not found`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthCheck := handler.HealthCheck()

			rec := httptest.NewRecorder()
			req, err := http.NewRequest(tt.given.method, "/health", nil)
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
