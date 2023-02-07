// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l7_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/linseed/pkg/handler/l3"

	"github.com/projectcalico/calico/linseed/pkg/backend/api"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

var (
	//go:embed testdata/input/all_l3flows_within_timerange.json
	withinTimeRange string

	//go:embed testdata/output/missing_timerange_error_msg.json
	missingTimeRangeErrorMsg string
)

func TestNetworkFlows_Post(t *testing.T) {
	type testResult struct {
		wantErr    bool
		httpStatus int
		errorMsg   string
	}

	tests := []struct {
		name           string
		reqBody        string
		want           testResult
		backendL3Flows []v1.L3Flow
	}{
		// Failure to parse request and validate
		{
			name:           "empty json",
			reqBody:        "{}",
			want:           testResult{true, 400, missingTimeRangeErrorMsg},
			backendL3Flows: noFlows,
		},

		// Retrieve all L3 flow logs within a time range
		{
			name:           "retrieve all l3 flows within a certain time range",
			reqBody:        withinTimeRange,
			want:           testResult{false, 200, ""},
			backendL3Flows: flows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := networkFlows(tt.backendL3Flows)

			rec := httptest.NewRecorder()
			req, err := http.NewRequest("POST", n.URL(), bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			require.NoError(t, err)

			n.Serve().ServeHTTP(rec, req)

			bodyBytes, err := io.ReadAll(rec.Body)
			require.NoError(t, err)

			var wantBody string
			if tt.want.wantErr {
				wantBody = tt.want.errorMsg
			} else {
				wantBody = marshalResponse(t, tt.backendL3Flows)
			}
			assert.Equal(t, tt.want.httpStatus, rec.Result().StatusCode)
			assert.JSONEq(t, wantBody, string(bodyBytes))
		})
	}
}

func networkFlows(flows []v1.L3Flow) *l3.NetworkFlows {
	mockBackend := &api.MockFlowBackend{}
	n := l3.NewFlows(mockBackend)

	res := v1.List[v1.L3Flow]{
		Items:    flows,
		AfterKey: nil,
	}

	// mock backend to return the required flows
	mockBackend.On("List", mock.Anything,
		mock.AnythingOfType("api.ClusterInfo"), mock.AnythingOfType("v1.L3FlowParams")).Return(&res, nil)

	return n
}

func marshalResponse(t *testing.T, flows []v1.L3Flow) string {
	response := v1.List[v1.L3Flow]{}
	response.Items = flows

	newData, err := json.Marshal(response)
	require.NoError(t, err)

	return string(newData)
}
