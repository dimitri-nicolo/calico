// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

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

			if logrus.IsLevelEnabled(logrus.TraceLevel) {
				body, err := ReadBody(w, r)
				if err != nil {
					logrus.WithError(err).Error("Failed to read body")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				logrus.Tracef("URL: %s", r.URL.Path)
				logrus.Tracef("Query: %s", r.URL.Query())
				logrus.Tracef("Body: %s", string(body))
				logrus.Tracef("Headers: %v", r.Header)
				logrus.Tracef("isJSON: %v", isJSON(r))
				logrus.Tracef("isGZIP: %v", isGZIP(r))
				logrus.Tracef("isAsyncSearch: %v", isAsyncSearch(r))
				logrus.Tracef("isSearch: %v", isSearch(r))
				logrus.Tracef("EnforceTenancy: %v", (isAsyncSearch(r) || isSearch(r)) && isJSON(r) && !isGZIP(r))
			}

			if (isAsyncSearch(r) || isSearch(r)) && isJSON(r) && !isGZIP(r) {
				logrus.Trace("Processing request to enforce tenancy")
				body, err := ReadBody(w, r)
				if err != nil {
					logrus.WithError(err).Error("Failed to read body")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				asyncRequest := make(map[string]interface{})
				err = json.Unmarshal(body, &asyncRequest)
				if err != nil {
					logrus.WithError(err).Error("Failed to unmarshal async query")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if asyncRequest["query"] != nil {
					logrus.Trace("Request has a query string")
					queryStr, err := json.Marshal(asyncRequest["query"])
					if err != nil {
						logrus.WithError(err).Error("Failed to marshal query field")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					query := RawQuery{}
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
					asyncRequest["query"] = FromSource(newQuery)

					// Update the body of the request.
					mod, err := json.Marshal(asyncRequest)
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
					// Not a query we understand - reject the request altogether.
					logrus.Warn("Request does not contain a query field - reject it")
					http.Error(w, "Request does not contain a query field", http.StatusBadRequest)
					return
				}
			}

			// Finally, pass to the next handler.
			next.ServeHTTP(w, r)
		})
	}
}

func isSearch(r *http.Request) bool {
	return strings.Contains(r.URL.Path, "tigera_secure_ee")
}

func isAsyncSearch(r *http.Request) bool {
	return strings.Contains(r.URL.Path, "_async_search")
}

func isGZIP(r *http.Request) bool {
	return strings.Contains(r.Header.Get(http.CanonicalHeaderKey("Content-Encoding")), "gzip")
}

func isJSON(r *http.Request) bool {
	contentType := r.Header.Get(http.CanonicalHeaderKey("Content-Type"))
	return r.Body != nil && strings.Contains(contentType, "application/json")
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

type RawQuery map[string]interface{}

func (r RawQuery) Source() (interface{}, error) {
	return r, nil
}

func FromSource(i interface{}) RawQuery {
	if r, ok := i.(map[string]interface{}); ok {
		return r
	}
	logrus.Warn("Failed to parse query of type %t", i)
	return nil
}
