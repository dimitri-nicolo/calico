// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFV_Ingestion(t *testing.T) {
	addr := "localhost:8444"
	tenant := ""

	ingestionTests := []struct {
		name             string
		bulkAPI          string
		logs             string
		expectedResponse string
	}{
		{
			name:             "ingest flow logs via bulk API",
			bulkAPI:          "/api/v1/flows/logs/bulk",
			logs:             flowLogsLinux,
			expectedResponse: `{"failed":0, "succeeded":25, "total":25}`,
		},
		{
			name:             "ingest dns logs via bulk API",
			bulkAPI:          "/api/v1/dns/logs/bulk",
			logs:             dnsLogsLinux,
			expectedResponse: `{"failed":0, "succeeded":30, "total":30}`,
		},
		{
			name:             "ingest l7 logs via bulk API",
			bulkAPI:          "/api/v1/l7/logs/bulk",
			logs:             l7LogsLinux,
			expectedResponse: `{"failed":0, "succeeded":15, "total":15}`,
		},
	}

	for _, tt := range ingestionTests {
		t.Run(tt.name, func(t *testing.T) {
			defer setupLinseedFV(t)()

			// setup HTTP client and HTTP request
			client := secureHTTPClient(t)
			spec := xndJSONPostHTTPReqSpec(fmt.Sprintf("https://%s%s", addr, tt.bulkAPI), tenant, cluster, []byte(tt.logs))

			// make the request to ingest flows
			res, resBody := doRequest(t, client, spec)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.JSONEq(t, tt.expectedResponse, strings.Trim(string(resBody), "\n"))
		})
	}
}
