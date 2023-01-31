// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l3_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler/l3"
)

func TestBulkIngestion_Serve(t *testing.T) {
	type testResult struct {
		wantErr    bool
		httpStatus int
		errorMsg   string
	}

	tests := []struct {
		name            string
		backendFlowLogs []v1.FlowLog
		backendResponse *v1.BulkResponse
		backendError    error
		reqBody         string
		want            testResult
	}{
		// Failure to parse request and validate
		{name: "empty json",
			backendFlowLogs: noFlowLogs,
			backendError:    nil,
			backendResponse: nil,
			reqBody:         "{}",
			want: testResult{true, 400,
				"{\"Status\":400,\"Msg\":\"Request body contains an empty JSON\",\"Err\":null}"},
		},

		// Ingest all flow logs
		{name: "ingest flows logs",
			backendFlowLogs: flowLogs,
			backendError:    nil,
			backendResponse: bulkResponseSuccess,
			reqBody:         marshal(flowLogs),
			want:            testResult{false, 200, ""},
		},

		// Fails to ingest all flow logs
		{name: "fail to ingest all flows logs",
			backendFlowLogs: flowLogs,
			backendError:    errors.New("any error"),
			backendResponse: nil,
			reqBody:         marshal(flowLogs),
			want:            testResult{true, 500, "{\"Status\":500,\"Msg\":\"any error\",\"Err\":{}}"},
		},

		// Ingest some flow logs
		{name: "ingest some flows logs",
			backendFlowLogs: flowLogs,
			backendError:    nil,
			backendResponse: bulkResponsePartialSuccess,
			reqBody:         marshal(flowLogs),
			want:            testResult{false, 200, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bulkFlowLogs(tt.backendResponse, tt.backendError)

			rec := httptest.NewRecorder()
			req, err := http.NewRequest("POST", b.URL(), bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			require.NoError(t, err)

			b.Serve().ServeHTTP(rec, req)

			bodyBytes, err := io.ReadAll(rec.Body)
			require.NoError(t, err)

			var wantBody string
			if tt.want.wantErr {
				wantBody = tt.want.errorMsg
			} else {
				wantBody = marshalBulkResponse(t, tt.backendResponse)
			}
			fmt.Println(string(bodyBytes))
			assert.Equal(t, tt.want.httpStatus, rec.Result().StatusCode)
			assert.JSONEq(t, wantBody, string(bodyBytes))
		})
	}
}

func bulkFlowLogs(response *v1.BulkResponse, err error) *l3.BulkIngestion {
	mockBackend := &api.MockFlowLogBackend{}
	b := l3.NewBulkIngestion(mockBackend)

	// mock backend to return the required backendFlowLogs
	mockBackend.On("Create", mock.Anything,
		mock.AnythingOfType("api.ClusterInfo"), mock.AnythingOfType("[]v1.FlowLog")).Return(response, err)

	return b
}

func marshal(flowLogs []v1.FlowLog) string {
	var logs []string

	for _, p := range flowLogs {
		newData, _ := json.Marshal(p)
		logs = append(logs, string(newData))
	}

	return strings.Join(logs, "\n")
}

func marshalBulkResponse(t *testing.T, response *v1.BulkResponse) string {
	newData, err := json.Marshal(response)
	require.NoError(t, err)

	return string(newData)
}
