// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/config"

	"github.com/stretchr/testify/assert"
)

// Token to use for HTTP requests against Linseed.
var token []byte

func setupLinseedFV(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an ES client.
	var err error
	esClient, err = elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)

	// Random cluster name to prevent overlap with other tests.
	cluster = testutils.RandomClusterName()

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	// Get the token to use in HTTP authorization header.
	token, err = os.ReadFile(TokenPath)
	require.NoError(t, err)

	return func() {
		// Cleanup any data that might left over from a previous failed run.
		err := testutils.CleanupIndices(context.Background(), esClient, cluster)
		require.NoError(t, err)
		logCancel()
		cancel()
	}
}

func TestFV_Linseed(t *testing.T) {
	addr := "localhost:8444"
	healthAddr := "localhost:8080"
	cluster := "cluster"
	tenant := "tenant-a"

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
			path: "/", method: "GET", wantStatusCode: 404, wantBody: `{"Status":404,"Msg":"No matching authz options for GET /"}`,
		},
		{
			name: "should return 404 for /foo",
			path: "/foo", method: "GET", wantStatusCode: 404, wantBody: `{"Status":404,"Msg":"No matching authz options for GET /foo"}`,
		},
		{
			name: "should return 404 for /api/v1/flows/foo",
			path: "/api/v1/flows/foo", method: "GET", wantStatusCode: 404, wantBody: `{"Status":404,"Msg":"No matching authz options for GET /api/v1/flows/foo"}`,
		},
		{
			name: "should return 404 for DELETE /version",
			path: "/version", method: "DELETE", wantStatusCode: 404, wantBody: `{"Status":404,"Msg":"No matching authz options for DELETE /version"}`,
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

			client := mTLSClient(t)
			httpReqSpec := noBodyHTTPReqSpec(tt.method, fmt.Sprintf("https://%s%s", addr, tt.path), tenant, cluster, token)
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
		res, resBody := doRequest(t, client, noBodyHTTPReqSpec("GET", fmt.Sprintf("http://%s/", addr), tenant, cluster, nil))

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "Client sent an HTTP request to an HTTPS server.", strings.Trim(string(resBody), "\n"))
	})

	t.Run("should deny any TLS connection", func(t *testing.T) {
		defer setupLinseedFV(t)()

		client := tlsClient(t)
		req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/", addr), nil)
		require.NoError(t, err)
		_, err = client.Do(req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "remote error: tls: bad certificate")
	})

	t.Run("should deny mTLS connections that use a certificate generated by a different CA", func(t *testing.T) {
		defer setupLinseedFV(t)()

		// Create the certificates
		ca, caKey := mustCreateCAKeyPair(t)
		caBytes := signAndEncodeCert(t, ca, caKey, ca, caKey)
		cert, key := mustCreateClientKeyPair(t)
		certBytes := signAndEncodeCert(t, ca, caKey, cert, key)
		keyBytes := encodeKey(t, key)

		client := mTLSClientWithCerts(certPool(caBytes), mustGetTLSKeyPair(t, certBytes, keyBytes))

		req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/", addr), nil)
		require.NoError(t, err)
		_, err = client.Do(req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "x509: certificate signed by unknown authority")
	})

	t.Run("should be ready", func(t *testing.T) {
		defer setupLinseedFV(t)()

		client := mTLSClient(t)
		httpReqSpec := noBodyHTTPReqSpec("GET", fmt.Sprintf("http://%s/readiness", healthAddr), tenant, cluster, token)
		res, _ := doRequest(t, client, httpReqSpec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("should be live", func(t *testing.T) {
		defer setupLinseedFV(t)()

		client := mTLSClient(t)
		httpReqSpec := noBodyHTTPReqSpec("GET", fmt.Sprintf("http://%s/liveness", healthAddr), tenant, cluster, token)
		res, _ := doRequest(t, client, httpReqSpec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}

func contentType(value string) map[string]string {
	return map[string]string{"content-type": value}
}
