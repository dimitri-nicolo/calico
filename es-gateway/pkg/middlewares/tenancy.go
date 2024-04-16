// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
			} else if asyncSearchRegexp.MatchString(r.URL.Path) && r.Method == http.MethodPost {
				searchRequest, err := readRequest(w, r)
				if err != nil {
					logrus.WithError(err).Error("Failed to process request body")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if searchRequest["query"] != nil {
					logrus.Trace("Request has a query string")
					queryStr, err := json.Marshal(searchRequest["query"])
					if err != nil {
						logrus.WithError(err).Error("Failed to marshal query field")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					query := Query{}
					err = json.Unmarshal(queryStr, &query)
					if err != nil {
						logrus.WithError(err).Error("Failed to unmarshal query string")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					// A valid query. Insert a tenant selector so that we enforce tenancy.
					logrus.Info("Add tenancy enforcement to request")
					// Create a new boolean query, and filter by tenant ID as well as the original query.
					tenancyQuery := elastic.NewBoolQuery()
					tenancyQuery.Must(elastic.NewTermQuery("tenant", k.tenantID))
					tenancyQuery.Filter(query)

					newQuery, err := tenancyQuery.Source()
					if err != nil {
						logrus.Warn("Failed to parse the new query")
						http.Error(w, "Failed to parse the new query", http.StatusBadRequest)
						return
					}
					searchRequest["query"] = FromSource(newQuery)

					// Update the body of the request.
					mod, err := json.Marshal(searchRequest)
					if err != nil {
						logrus.Warn("Failed to parse the marshal the new query")
						http.Error(w, "Failed to parse the marshal the new query", http.StatusBadRequest)
						return
					}

					logrus.Infof("Modified query: %s", string(mod))
					r.Body = io.NopCloser(bytes.NewBuffer(mod))

					// Set a new Content-Length.
					r.ContentLength = int64(len(mod))

				} else {
					// We want to reject all requests without a query field
					k.rejectRequest(w, r)
					return
				}
			}

			// Finally, pass to the next handler.
			next.ServeHTTP(w, r)
		})
	}
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

func readRequest(w http.ResponseWriter, r *http.Request) (map[string]interface{}, error) {
	body, err := ReadBody(w, r)
	if err != nil {
		return nil, err
	}

	request := make(map[string]interface{})
	err = json.Unmarshal(body, &request)
	if err != nil {
		return nil, err
	}
	return request, nil
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

type Query map[string]interface{}

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

type AsyncSearchRequest struct {
}
