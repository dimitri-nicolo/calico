package middlewares_test

import (
	"bytes"
	"encoding/json"
	"github.com/projectcalico/calico/es-gateway/pkg/middlewares"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKibanaTenancy_Enforce(t *testing.T) {
	const (
		tenantID           string = "anyTenant"
		sampleBooleanQuery        = `
{
  "query": { 
    "bool": { 
      "must": [
        { "match": { "title":   "Search"        }},
        { "match": { "content": "Elasticsearch" }}
      ],
      "filter": [ 
        { "term":  { "status": "published" }},
        { "range": { "publish_date": { "gte": "2015-01-01" }}}
      ]
    }
  }
}`
		sampleQueryStringQuery = `
{
  "query": {
    "query_string": {
      "query": "(new york city) OR (big apple)",
      "default_field": "content"
    }
  }
}`
		sampleFuzzyQuery = `
{
  "query": {
    "fuzzy": {
      "user.id": {
        "value": "ki"
      }
    }
  }
}`
		sampleRegexpQuery = `
{
  "query": {
    "regexp": {
      "user.id": {
        "value": "k.*y",
        "flags": "ALL",
        "case_insensitive": true,
        "max_determinized_states": 10000,
        "rewrite": "constant_score"
      }
    }
  }
}`
		samplePrefixQuery = `
{
  "query": {
    "prefix": {
      "user.id": {
        "value": "ki"
      }
    }
  }
}`
		sampleWildcardQuery = `
{
  "query": {
    "wildcard": {
      "user.id": {
        "value": "ki*y",
        "boost": 1.0,
        "rewrite": "constant_score"
      }
    }
  }
}`
		sampleRangeQuery = `
{
  "query": {
    "range": {
      "age": {
        "gte": 10,
        "lte": 20,
        "boost": 2.0
      }
    }
  }
}`
	)

	expectedTenantQuery := map[string]interface{}{
		"term": map[string]interface{}{
			"tenant": tenantID,
		},
	}

	tests := []struct {
		name       string
		url        string
		body       string
		wantStatus int
	}{
		{
			name:       "Should enforce tenancy for a generic query boolean query using POST",
			url:        "/tigera_secure_ee_anyData*/_async_search",
			body:       sampleBooleanQuery,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Should deny any search request without an empty query field",
			url:        "/tigera_secure_ee_anyData*/_async_search",
			body:       `{"query":{}}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "Should deny an empty requests",
			url:        "/tigera_secure_ee_anyData*/_async_search",
			body:       `{}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "Should not process malformed requests",
			url:        "/tigera_secure_ee_anyData*/_async_search",
			body:       `{#!#$!1}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "Should deny aggregations requests without a query field",
			url:        "/tigera_secure_ee_anyData*/_async_search",
			body:       `{"agg":{}}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Should enforce tenancy for a generic query string",
			url:  "/tigera_secure_ee_anyData*/_async_search",
			// https://www.elastic.co/guide/en/elasticsearch/reference/7.17/query-dsl-query-string-query.html
			body:       sampleQueryStringQuery,
			wantStatus: http.StatusOK,
		},
		{
			name: "Should enforce tenancy for a generic fuzzy string",
			url:  "/tigera_secure_ee_anyData*/_async_search",
			// https://www.elastic.co/guide/en/elasticsearch/reference/7.17/query-dsl-fuzzy-query.html
			body:       sampleFuzzyQuery,
			wantStatus: http.StatusOK,
		},
		{
			name: "Should enforce tenancy for a generic regex query",
			url:  "/tigera_secure_ee_anyData*/_async_search",
			// https://www.elastic.co/guide/en/elasticsearch/reference/7.17/query-dsl-regexp-query.html
			body:       sampleRegexpQuery,
			wantStatus: http.StatusOK,
		},
		{
			name: "Should enforce tenancy for a generic prefix query",
			url:  "/tigera_secure_ee_anyData*/_async_search",
			// https://www.elastic.co/guide/en/elasticsearch/reference/7.17/query-dsl-regexp-query.html
			body:       samplePrefixQuery,
			wantStatus: http.StatusOK,
		},
		{
			name: "Should enforce tenancy for a generic wildcard query",
			url:  "/tigera_secure_ee_anyData*/_async_search",
			// https://www.elastic.co/guide/en/elasticsearch/reference/7.17/query-dsl-wildcard-query.html
			body:       sampleWildcardQuery,
			wantStatus: http.StatusOK,
		},
		{
			name: "Should enforce tenancy for a generic ranger query",
			url:  "/tigera_secure_ee_anyData*/_async_search",
			// https://www.elastic.co/guide/en/elasticsearch/reference/7.17/query-dsl-range-query.html#range-query-ex-request
			body:       sampleRegexpQuery,
			wantStatus: http.StatusOK,
		},
		// Script, Script score, Percolate, Nested requests, has child, has parent
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middlewares.NewKibanaTenancy(tenantID).Enforce()
			rec := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, tt.url, bytes.NewBufferString(tt.body))
			require.NoError(t, err)

			// Process the requests
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// The test handler simply returns an OK body. This is used by the
				// tests to prove that the middleware passed the request on
				// to the next handler.
				w.WriteHeader(http.StatusOK)

				body, err := middlewares.ReadBody(w, r)
				require.NoError(t, err)
				queryBody := make(map[string]interface{})
				err = json.Unmarshal(body, &queryBody)
				require.NoError(t, err)
				require.NotNil(t, queryBody["query"])
				query := queryBody["query"].(map[string]interface{})
				require.NotNil(t, query["bool"])
				filter := query["bool"].(map[string]interface{})
				require.NotNil(t, filter["must"])
				tenantQuery := filter["must"].(map[string]interface{})
				require.Equal(t, expectedTenantQuery, tenantQuery)
			})

			handler(testHandler).ServeHTTP(rec, req)
			// Check the returned status code
			require.Equal(t, tt.wantStatus, rec.Result().StatusCode)
		})
	}
}
