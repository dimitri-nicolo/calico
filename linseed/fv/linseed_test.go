// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestFV_Linseed(t *testing.T) {
	addr := "localhost:8444"
	elasticEndpoint := "http://localhost:9200"
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
			name: "should return 415 unsupported content type for /api/v1/flows/network",
			path: "/api/v1/flows/network", method: "POST",
			headers: contentType("text/plain"), body: "{}", wantStatusCode: 415, wantBody: "",
		},
		{
			name: "should return 415 unsupported content type for /api/v1/bulk/flows/network/logs",
			path: "/api/v1/bulk/flows/network/logs", method: "POST",
			headers: contentType("text/plain"), body: "{}", wantStatusCode: 415, wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := secureHTTPClient(t)
			httpReqSpec := noBodyHTTPReqSpec(tt.method, fmt.Sprintf("https://%s%s", addr, tt.path), tenant, cluster)
			httpReqSpec.AddHeaders(tt.headers)
			httpReqSpec.SetBody(tt.body)
			res, resBody := doRequest(t, client, httpReqSpec)

			assert.Equal(t, tt.wantStatusCode, res.StatusCode)
			assert.Equal(t, tt.wantBody, strings.Trim(string(resBody), "\n"))

			return
		})
	}

	t.Run("should deny any HTTP connection", func(t *testing.T) {
		client := &http.Client{}
		res, resBody := doRequest(t, client, noBodyHTTPReqSpec("GET", fmt.Sprintf("http://%s/", addr), tenant, cluster))

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "Client sent an HTTP request to an HTTPS server.", strings.Trim(string(resBody), "\n"))
	})

	t.Run("should ingest flow logs to Elastic", func(t *testing.T) {
		// setup Elastic client
		esClient, err := elastic.NewClient(
			elastic.SetURL(elasticEndpoint),
			elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
			elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)))
		require.NoError(t, err)

		// setup HTTP client and HTTP request
		client := secureHTTPClient(t)
		spec := xndJSONPostHTTPReqSpec(fmt.Sprintf("https://%s%s", addr, "/api/v1/bulk/flows/network/logs"),
			tenant, cluster, []byte(flowLogsLinux))
		// make the request to ingest flows
		res, resBody := doRequest(t, client, spec)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.JSONEq(t, "{\"failed\":0, \"succeeded\":25, \"total\":25}", strings.Trim(string(resBody), "\n"))

		index := fmt.Sprintf("tigera_secure_ee_flows.cluster.*")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err = esClient.Refresh(index).Do(ctx)
		require.NoError(t, err)
		query := elastic.NewTermQuery("end_time", "1675469001")
		result, err := esClient.Search().Index("tigera_secure_ee_flows.cluster.*").Query(query).Do(ctx)
		require.NoError(t, err)
		assert.Equal(t, result.Hits.TotalHits.Value, int64(25))
	})
}

func contentType(value string) map[string]string {
	return map[string]string{"content-type": value}
}
