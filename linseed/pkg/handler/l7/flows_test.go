// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l7

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
)

func setupTest(t *testing.T) func() {
	cancel := logutils.RedirectLogrusToTestingT(t)
	return func() {
		cancel()
	}
}

// The expected error returned from the server when there are no parameters provided.
const errNoInput = `
{
  "Status": 400,
  "Msg": "error with field TimeRange = '\u003cnil\u003e' (Reason: failed to validate Field: TimeRange because of Tag: required )"
}
`

// A valid query input that provides the necessary time range parameters.
const withinTimeRange = `
{
  "time_range": {
    "from": "2021-04-19T14:25:30.169821857-07:00",
    "to": "2021-04-19T14:25:30.169827009-07:00"
  },
  "timeout": "60s"
}
`

func TestL7FlowsHandler(t *testing.T) {
	type testResult struct {
		httpStatus int
		errorMsg   string
	}

	tests := []struct {
		name         string
		reqBody      string
		want         testResult
		backendFlows []v1.L7Flow
	}{
		// Failure to parse request and validate
		{
			name:         "empty json",
			reqBody:      "{}",
			want:         testResult{400, errNoInput},
			backendFlows: []v1.L7Flow{},
		},

		// Retrieve all  flow logs within a time range
		{
			name:         "retrieve all flows within time range",
			reqBody:      withinTimeRange,
			want:         testResult{200, ""},
			backendFlows: []v1.L7Flow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer setupTest(t)()

			n := flowHandler(tt.backendFlows)

			rec := httptest.NewRecorder()
			req, err := http.NewRequest("POST", n.APIS()[0].URL, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			require.NoError(t, err)

			// Serve the request and read the response.
			n.Serve().ServeHTTP(rec, req)
			bodyBytes, err := io.ReadAll(rec.Body)
			require.NoError(t, err)

			var wantBody string
			if tt.want.errorMsg != "" {
				wantBody = tt.want.errorMsg
			} else {
				wantBody = marshalResponse(t, tt.backendFlows)
			}
			assert.Equal(t, tt.want.httpStatus, rec.Result().StatusCode)
			assert.JSONEq(t, wantBody, string(bodyBytes))
		})
	}
}

func flowHandler(flows []v1.L7Flow) *flows {
	mockBackend := &api.MockL7FlowBackend{}
	n := NewFlows(mockBackend)

	res := v1.List[v1.L7Flow]{
		Items:    flows,
		AfterKey: nil,
	}

	// mock backend to return the required flows
	mockBackend.On("List", mock.Anything,
		mock.AnythingOfType("api.ClusterInfo"), mock.AnythingOfType("v1.L7FlowParams")).Return(&res, nil)

	return n
}

func marshalResponse(t *testing.T, flows []v1.L7Flow) string {
	response := v1.List[v1.L7Flow]{}
	response.Items = flows
	newData, err := json.Marshal(response)
	require.NoError(t, err)
	return string(newData)
}
