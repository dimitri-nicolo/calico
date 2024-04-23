// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

const maxSize = 100 * 1000000

// KibanaTenancy is a middleware that enforces tenant isolations
// for all queries made to Elastic.
type KibanaTenancy struct {
	tenantID string
}

func NewKibanaTenancy(tenantID string) *KibanaTenancy {
	return &KibanaTenancy{tenantID: tenantID}
}

func (k KibanaTenancy) Enforce() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if k.traceRequest(w, r) {
				return
			}

			allow, err := IsAllowed(w, r)
			if err != nil {
				logrus.WithError(err).Error("Failed to process request")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else if !allow {
				k.rejectRequest(w, r)
				return
			}

			if asyncSearchRegexp.MatchString(r.URL.Path) {
				err := k.enhanceWithTenancyQuery(w, r)
				if err != nil {
					logrus.WithError(err).Error("Failed to enforce tenancy")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// Finally, pass to the next handler.
			next.ServeHTTP(w, r)
		})
	}
}

func (k KibanaTenancy) enhanceWithTenancyQuery(w http.ResponseWriter, r *http.Request) error {
	body, err := ReadBody(w, r)
	if err != nil {
		return err
	}

	asyncSearchRequest := AsyncSearchRequest{}
	err = json.Unmarshal(body, &asyncSearchRequest)
	if err != nil {
		return err
	}

	if asyncSearchRequest.Query == nil || len(asyncSearchRequest.Query) == 0 {
		return fmt.Errorf("RequestQuery is empty")
	}

	// A valid query. Insert a tenant selector so that we enforce tenancy.
	logrus.Debug("Add tenancy enforcement to request")
	// Create a new boolean query, and filter by tenant ID as well as the original query.
	tenancyQuery := elastic.NewBoolQuery()
	tenancyQuery.Must(elastic.NewTermQuery("tenant", k.tenantID))
	tenancyQuery.Filter(asyncSearchRequest.Query)

	newQuery, err := tenancyQuery.Source()
	if err != nil {
		return err
	}
	asyncSearchRequest.Query = FromSource(newQuery)

	// Update the body of the request.
	mod, err := json.Marshal(asyncSearchRequest)
	if err != nil {
		return err
	}

	logrus.Tracef("Modified query: %s", string(mod))
	r.Body = io.NopCloser(bytes.NewBuffer(mod))

	// Set a new Content-Length.
	r.ContentLength = int64(len(mod))
	return nil
}

func (k KibanaTenancy) traceRequest(w http.ResponseWriter, r *http.Request) bool {
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		body, err := ReadBody(w, r)
		if err != nil {
			logrus.WithError(err).Error("Failed to read body")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		logrus.Tracef("URL: %s", r.URL.Path)
		logrus.Tracef("Query: %v", r.URL.Query())
		logrus.Tracef("Body: %s", string(body))
		// TODO: Alina: Hide Authorization from print
		logrus.Tracef("Headers: %v", r.Header)
	}
	return false
}

func (k KibanaTenancy) rejectRequest(w http.ResponseWriter, r *http.Request) {
	logrus.Warnf("Request %s %s is not allowed - reject it", r.Method, r.URL.Path)
	http.Error(w, fmt.Sprintf("Request is not allowed %s", r.URL.Path), http.StatusForbidden)
}

func ReadBody(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	req.Body = http.MaxBytesReader(w, req.Body, maxSize)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

// AsyncSearchRequest is the definition used to define which fields are allowed by an_ async_search
// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/async-search.html
// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/search-search.html
type AsyncSearchRequest struct {
	// docvalue_fields (Optional, array of strings and objects)
	DocValueFields []interface{} `json:"doc_value_fields,omitempty"`
	// fields (Optional, array of strings and objects)
	Fields []interface{} `json:"fields,omitempty"`
	// explain (Optional, Boolean)
	Explain *bool `json:"explain,omitempty"`
	// from (Optional, integer)
	From *int `json:"from,omitempty"`
	// indices_boost (Optional, array of objects)
	IndicesBoost []interface{} `json:"indices_boost,omitempty"`
	// min_score (Optional, float)
	MinScore *float64 `json:"min_score,omitempty"`
	// query (Optional, query object)
	Query Query `json:"query,omitempty"`
	// runtime_mappings (Optional, object of objects)
	RuntimeMappings map[string]interface{} `json:"runtime_mappings,omitempty"`
	// seq_no_primary_term (Optional, Boolean)
	SequenceNoPrimaryTerm *bool `json:"seq_no_primary_term,omitempty"`
	// size (Optional, integer)
	Size *int `json:"size,omitempty"`
	// _source (Optional)
	Source map[string]interface{} `json:"_source,omitempty"`
	// stats (Optional, array of strings)
	Stats []string `json:"stats,omitempty"`
	// terminate_after (Optional, integer)
	TerminateAfter *int `json:"terminate_after,omitempty"`
	// timeout (Optional, time units)
	Timeout *time.Duration `json:"timeout,omitempty"`
	// version (Optional, Boolean)
	Version *bool `json:"version,omitempty"`
	// aggregations (Optional)
	// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/search-aggregations.html
	Aggregations map[string]interface{} `json:"aggs,omitempty"`
}

// Query field pass on the async request
type Query map[string]interface{}

// Source will convert a query to an interface
// We need to overwrite this function needed by oliver library
// in order to perform converstion from interface{} to Query struct
func (r Query) Source() (interface{}, error) {
	return r, nil
}

func FromSource(i interface{}) Query {
	if r, ok := i.(map[string]interface{}); ok {
		return r
	}
	logrus.Warnf("Failed to parse query of type %t", i)
	return nil
}
