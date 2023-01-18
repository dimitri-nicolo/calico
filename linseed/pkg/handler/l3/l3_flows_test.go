// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l3

import (
	"bytes"
	_ "embed"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	//go:embed testdata/input/all_l3flows_within_timerange.json
	withinTimeRange string
	//go:embed testdata/input/missing_timerange.json
	missingTimeRange string
	//go:embed testdata/input/all_stats.json
	allStats string
	//go:embed testdata/input/only_tcp_stats.json
	onlyTCPStats string
	//go:embed testdata/input/unknown_stats.json
	unknownStats string

	//go:embed testdata/output/missing_input_error_msg.json
	missingInputErrorMsg string
	//go:embed testdata/output/malformed_input_error_msg.json
	malformedInputErrorMsg string
	//go:embed testdata/output/missing_timerange_error_msg.json
	missingTimeRangeErrorMsg string
	//go:embed testdata/output/unknown_stats.json
	unknownStatsRangeErrorMsg string
	//go:embed testdata/output/only_tcp_stats.json
	onlyTCPStatsMsg string
	//go:embed testdata/output/all_stats.json
	allStatsMsg string
	//go:embed testdata/output/ok.json
	okMsg string
)

func TestNetworkFlows_Post(t *testing.T) {
	type testResult struct {
		wantErr    bool
		httpStatus int
		httpBody   string
	}

	tests := []struct {
		name    string
		reqBody string
		want    testResult
	}{
		// Failure to parse request and validate
		{name: "empty json",
			reqBody: "{}",
			want:    testResult{true, 400, missingTimeRangeErrorMsg}},
		{name: "missing input",
			reqBody: "",
			want:    testResult{true, 400, missingInputErrorMsg}},
		{name: "malformed json",
			reqBody: "{#$.}",
			want:    testResult{true, 400, malformedInputErrorMsg}},
		{name: "missing time range",
			reqBody: missingTimeRange,
			want:    testResult{true, 400, missingTimeRangeErrorMsg}},
		{name: "unknown statistics",
			reqBody: unknownStats,
			want:    testResult{true, 400, unknownStatsRangeErrorMsg}},

		// Retrieve all L3 flow logs within a time range
		{name: "retrieve all l3 flows within a certain time range",
			reqBody: withinTimeRange,
			want:    testResult{false, 200, okMsg}},
		// Retrieve all L3 flow logs with all statistics
		{name: "retrieve all l3 flows with all statistics",
			reqBody: allStats,
			want:    testResult{false, 200, allStatsMsg}},
		// Retrieve only TCP statistics for the L3 flow logs
		{name: "retrieve all l3 flows and generate only TCP statistics",
			reqBody: onlyTCPStats,
			want:    testResult{false, 200, onlyTCPStatsMsg}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NetworkFlows{}

			rec := httptest.NewRecorder()
			req, err := http.NewRequest("POST", n.URL(), bytes.NewBufferString(tt.reqBody))
			assert.NoError(t, err)

			n.Post().ServeHTTP(rec, req)

			bodyBytes, err := io.ReadAll(rec.Body)
			assert.NoError(t, err)

			assert.Equal(t, tt.want.httpStatus, rec.Result().StatusCode)
			assert.JSONEq(t, tt.want.httpBody, string(bodyBytes))
		})
	}
}
