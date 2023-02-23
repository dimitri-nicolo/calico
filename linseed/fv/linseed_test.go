// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/stretchr/testify/assert"
)

func setupLinseedFV(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Random cluster name to prevent overlap with other tests.
	cluster = testutils.RandomClusterName()

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		// Cleanup any data that might left over from a previous failed run.
		err := testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_flows.%s", cluster))
		require.NoError(t, err)
		logCancel()
		cancel()
	}
}

func TestFV_Linseed(t *testing.T) {
	addr := "localhost:8444"
	healthAddr := "localhost:8080"
	cluster := "cluster"
	tenant := ""

	tests := []struct {
		name           string
		path           string
		method         string
		headers        map[string]string
		body           string
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "should return 404 for /",
			path: "/", method: "GET", wantStatusCode: 404, wantBody: "404 page not found",
		},
		{
			name: "should return 404 for /foo",
			path: "/foo", method: "GET", wantStatusCode: 404, wantBody: "404 page not found",
		},
		{
			name: "should return 404 for /api/v1/flows/foo",
			path: "/api/v1/flows/foo", method: "GET", wantStatusCode: 404, wantBody: "404 page not found",
		},
		{
			name: "should return 405 for DELETE /version",
			path: "/version", method: "DELETE", wantStatusCode: 405, wantBody: "",
		},
		{
			name: "should return 415 unsupported content type for /api/v1/flows",
			path: "/api/v1/flows/", method: "POST",
			headers: contentType("text/plain"), body: "{}", wantStatusCode: 415, wantBody: "",
		},
		{
			name: "should return 415 unsupported content type for /api/v1/flows/logs/bulk",
			path: "/api/v1/flows/logs/bulk", method: "POST",
			headers: contentType("text/plain"), body: "{}", wantStatusCode: 415, wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer setupLinseedFV(t)()

			client := secureHTTPClient(t)
			httpReqSpec := noBodyHTTPReqSpec(tt.method, fmt.Sprintf("https://%s%s", addr, tt.path), tenant, cluster)
			httpReqSpec.AddHeaders(tt.headers)
			httpReqSpec.SetBody(tt.body)
			res, resBody := doRequest(t, client, httpReqSpec)

			assert.Equal(t, tt.wantStatusCode, res.StatusCode)
			assert.Equal(t, tt.wantBody, strings.Trim(string(resBody), "\n"))
		})
	}

	t.Run("should deny any HTTP connection", func(t *testing.T) {
		defer setupLinseedFV(t)()

		client := &http.Client{}
		res, resBody := doRequest(t, client, noBodyHTTPReqSpec("GET", fmt.Sprintf("http://%s/", addr), tenant, cluster))

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "Client sent an HTTP request to an HTTPS server.", strings.Trim(string(resBody), "\n"))
	})

	t.Run("should be ready", func(t *testing.T) {
		defer setupLinseedFV(t)()

		client := secureHTTPClient(t)
		httpReqSpec := noBodyHTTPReqSpec("GET", fmt.Sprintf("http://%s/readiness", healthAddr), tenant, cluster)
		res, _ := doRequest(t, client, httpReqSpec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("should be live", func(t *testing.T) {
		defer setupLinseedFV(t)()

		client := secureHTTPClient(t)
		httpReqSpec := noBodyHTTPReqSpec("GET", fmt.Sprintf("http://%s/liveness", healthAddr), tenant, cluster)
		res, _ := doRequest(t, client, httpReqSpec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}

func contentType(value string) map[string]string {
	return map[string]string{"content-type": value}
}
