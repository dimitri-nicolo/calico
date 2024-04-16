package middlewares_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/es-gateway/pkg/middlewares"
)

func TestIsWhiteListed(t *testing.T) {
	const (
		bulkBodySampleWithKibanaIndices = `{"index" : { "_index" : ".kibana", "_id" : "1" }}
{ "field1" : "value1" }
{ "delete" : { "_index" : ".kibana", "_id" : "2" } }
{ "create" : { "_index" : ".kibana", "_id" : "3" } }
{ "field1" : "value1" }
{ "update" : {"_id" : "1", "_index" : ".kibana"} }
{ "doc" : {"field2" : "value2"} }`
		bulkBodyIndexWithNonKibanaIndices = `{"index" : { "_index" : "anyIndex", "_id" : "1" }}
{ "field1" : "value1" }`
		bulkBodyDeleteWithNonKibanaIndices = `{ "delete" : { "_index" : "anyIndex", "_id" : "2" } }`
		bulkBodyCreateWithNonKibanaIndices = `{ "create" : { "_index" : "anyIndex", "_id" : "3" } }
{ "field1" : "value1" }`
		bulkBodyUpdateWithNonKibanaIndices = `{ "update" : {"_id" : "1", "_index" : "anyIndex"} }
{ "doc" : {"field2" : "value2"} }`
		bulkBodyWithKibanaAndNonKibanaIndices = `{"index" : { "_index" : ".kibana", "_id" : "1" }}
{ "field1" : "value1" }
{"index" : { "_index" : "anyIndex", "_id" : "1" }}
{ "field1" : "value1" }`
	)

	tests := []struct {
		name      string
		method    string
		url       string
		body      string
		wantAllow bool
		wantError error
	}{
		{
			name:      "Should reject any bulk requests that targets a calico index ",
			method:    http.MethodPost,
			url:       "/tigera_secure_ee_flows.123/_bulk",
			wantAllow: false,
		},
		{
			name:      "Should allow any bulk requests that targets a kibana index",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      bulkBodySampleWithKibanaIndices,
			wantAllow: true,
		},
		{
			name:      "Should reject any bulk requests that does not a targets a kibana index for an index action",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      bulkBodyIndexWithNonKibanaIndices,
			wantAllow: false,
		},
		{
			name:      "Should reject any bulk requests that does not targets a kibana index for a delete action",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      bulkBodyDeleteWithNonKibanaIndices,
			wantAllow: false,
		},
		{
			name:      "Should reject any bulk requests that does not targets a kibana index for a create action",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      bulkBodyCreateWithNonKibanaIndices,
			wantAllow: false,
		},
		{
			name:      "Should reject any bulk requests that does not targets a kibana index for a update action",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      bulkBodyUpdateWithNonKibanaIndices,
			wantAllow: false,
		},
		{
			name:      "Should reject any bulk requests that contains both kibana indices and non-kibana indices",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      bulkBodyWithKibanaAndNonKibanaIndices,
			wantAllow: false,
		},
		{
			name:      "Should process bulk requests ending in newline",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      fmt.Sprintf("%s\n", bulkBodySampleWithKibanaIndices),
			wantAllow: true,
		},
		{
			name:      "Should process bulk requests ending in newline for Windows",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      fmt.Sprintf("%s\r\n", bulkBodySampleWithKibanaIndices),
			wantAllow: true,
		},
		{
			name:      "Should not process a empty bulk request",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      ``,
			wantAllow: false,
			wantError: fmt.Errorf("unexpected end of JSON input"),
		},
		{
			name:      "Should not process bulk request with no actions",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      fmt.Sprintf(`{}%s{}%s`, "\n", "\n"),
			wantAllow: false,
			wantError: middlewares.NoIndexBulkError,
		},
		{
			name:      "Should not process a malformed bulk request",
			method:    http.MethodPost,
			url:       "/_bulk",
			body:      `{@#$!@#!32}`,
			wantAllow: false,
			wantError: fmt.Errorf("invalid character '@' looking for beginning of object key string"),
		},
		{
			name:      "Should allow all GET requests for an index that starts with Kibana",
			method:    http.MethodGet,
			url:       "/.kibana",
			wantAllow: true,
		},
		{
			name:      "Allow all PUT requests for an index that starts with Kibana",
			method:    http.MethodPut,
			url:       "/.kibana",
			wantAllow: true,
		},
		{
			name:      "Allow all DELETE requests for an index that starts with Kibana",
			method:    http.MethodDelete,
			url:       "/.kibana",
			wantAllow: true,
		},
		{
			name:      "Allow all POST requests for an index that starts with Kibana",
			method:    http.MethodPost,
			url:       "/.kibana",
			wantAllow: true,
		},
		{
			name:      "Allow all HEAD requests for an index that starts with Kibana",
			method:    http.MethodHead,
			url:       "/.kibana",
			wantAllow: true,
		},
		{
			name:      "Should allow any node information requests",
			method:    http.MethodGet,
			url:       "/_nodes?filter_path=nodes.*.version%2Cnodes.*.http.publish_address%2Cnodes.*.ip",
			wantAllow: true,
		},
		{
			name:      "Should allow any point in time deletion request",
			method:    http.MethodDelete,
			url:       "/_pit",
			wantAllow: true,
		},
		{
			name:      "Should allow any requests to read tasks",
			method:    http.MethodGet,
			url:       "/_tasks/",
			wantAllow: true,
		},
		{
			name:      "Should allow to check existence for templates for kibana",
			method:    http.MethodHead,
			url:       "/_template/.kibana",
			wantAllow: true,
		},
		{
			name:      "Should allow to read templates for kibana index templates",
			method:    http.MethodGet,
			url:       "/_template/kibana_index_template*",
			wantAllow: true,
		},
		// _search
		{
			name:      "Should allow for kibana to read its privileges",
			method:    http.MethodGet,
			url:       "/_security/privilege/kibana-.kibana",
			wantAllow: true,
		},
		{
			name:      "Should allow for kibana is check its privileges",
			method:    http.MethodPost,
			url:       "/_security/user/_has_privileges",
			wantAllow: true,
		},
		{
			name:      "Should allow for kibana to check elastic license",
			method:    http.MethodGet,
			url:       "/_xpack",
			wantAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
			require.NoError(t, err)
			gotAllow, gotError := middlewares.IsAllowed(nil, req)
			require.Equal(t, tt.wantAllow, gotAllow)
			if tt.wantError != nil {
				require.Error(t, gotError)
				require.Equal(t, tt.wantError.Error(), gotError.Error())
			} else {
				require.NoError(t, gotError)
			}
		})
	}
}
