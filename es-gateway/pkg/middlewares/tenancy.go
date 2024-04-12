// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package middlewares

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
			if k.traceRequest(w, r) {
				return
			}

			allow, inspectBody := IsWhiteListed(r)
			if !allow {
				k.rejectRequest(w, r)
				return
			} else if inspectBody {
				if r.URL.Path == "/_search" {
					// We expect this type of request to be issued only against Kibana indices
					body, err := ReadBody(w, r)
					if err != nil {
						logrus.WithError(err).Error("Failed to read body")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					searchRequest := SearchRequestWithPIT{}
					err = json.Unmarshal(body, &searchRequest)
					if err != nil {
						logrus.WithError(err).Error("Failed to read _search request with pit")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					if searchRequest.PIT.ID == "" {
						// We will reject any search without an index and a point in time
						k.rejectRequest(w, r)
						return
					}

					id, err := base64Decode(searchRequest.PIT.ID)
					if err != nil {
						logrus.WithError(err).Errorf("Failed to process point in time")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					if !strings.Contains(id, ".kibana") {
						// We will reject any search request with a point in time that does
						// reference a kibana index
						k.rejectRequest(w, r)
						return
					}
				} else if strings.HasPrefix(r.URL.Path, "/_bulk") {
					// This is a bulk request. Bulk request have the following format
					// POST _bulk
					//{ "index" : { "_index" : "test", "_id" : "1" } }
					//{ "field1" : "value1" }
					//{ "delete" : { "_index" : "test", "_id" : "2" } }
					//{ "create" : { "_index" : "test", "_id" : "3" } }
					//{ "field1" : "value3" }
					//{ "update" : {"_id" : "1", "_index" : "test"} }
					//{ "doc" : {"field2" : "value2"} }
					body, err := ReadBody(w, r)
					if err != nil {
						logrus.WithError(err).Error("Failed to read body")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					// We need to process each line and determine if we have index, delete, create or update
					lines := strings.Split(string(bytes.Trim(body, "\r\n")), "\n")
					for index := 0; index < len(lines); index++ {
						logrus.Trace("Process line for bulk operation")
						bulkRequest := BulkRequest{}
						err = json.Unmarshal([]byte(lines[index]), &bulkRequest)
						if err != nil {
							logrus.WithError(err).Error("Failed to read bulk operation")
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						if bulkRequest.Index != nil || bulkRequest.Update != nil || bulkRequest.Create != nil {
							// This is an index/update/create elastic action
							// These actions expect the full document on the next line
							// We will need to skip processing next element
							index++
						}

						indexMetadata := bulkRequest.GetIndexMetadata()
						if indexMetadata == nil {
							logrus.WithError(err).Error("Failed to determine action type for bulk request")
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						if !strings.HasPrefix(indexMetadata.Index, ".kibana") {
							logrus.Debugf("Index %s does not start with .kibana", indexMetadata.Index)
							// Reject all
							k.rejectRequest(w, r)
							return
						}
					}
				} else if r.URL.Path == "/_mget" {
					// This is an mget request and these requests have the following format
					// POST /_mget
					//{
					//  "docs": [
					//    {
					//      "_index": "my-index-000001",
					//      "_id": "1"
					//    },
					//    {
					//      "_index": "my-index-000001",
					//      "_id": "2"
					//    }
					//  ]
					//}
					body, err := ReadBody(w, r)
					if err != nil {
						logrus.WithError(err).Error("Failed to read body")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					mGetRequest := MultipleGetRequest{}
					err = json.Unmarshal(body, &mGetRequest)
					if err != nil {
						logrus.WithError(err).Error("Failed to read mget request")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					for _, doc := range mGetRequest.Docs {
						if !strings.HasPrefix(doc.Index, ".kibana") {
							logrus.Debugf("Index %s does not start with .kibana", doc.Index)
							// Reject all
							k.rejectRequest(w, r)
							return
						}
					}
				} else if strings.HasPrefix(r.URL.Path, "/_async_search") && r.Method == http.MethodGet {
					queryParams := r.URL.Query()
					if queryParams.Has("q") {
						// Reject all
						k.rejectRequest(w, r)
						return
					}
				} else if asyncSearchRegexp.MatchString(r.URL.Path) && r.Method == http.MethodPost {
					queryParams := r.URL.Query()
					if queryParams.Has("q") {
						// Reject all
						k.rejectRequest(w, r)
						return
					}
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
	logrus.Warnf("Request %s %s is not whitelisted - reject it", r.Method, r.URL.Path)
	http.Error(w, fmt.Sprintf("Request is not whitelisted %s", r.URL.Path), http.StatusForbidden)
}

func base64Decode(pitID string) (string, error) {
	// Search request with a point in time do not specify
	// the index on the request. We can bse64 decode and extract the name
	decodedID, err := base64.StdEncoding.DecodeString(pitID)
	if err != nil {
		return "", err
	}

	return string(decodedID), nil
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
	logrus.Warn("Failed to parse query of type %t", i)
	return nil
}

type AsyncSearchRequest struct {
}

type SearchRequestWithPIT struct {
	PIT PointInTime `json:"pit"`
}

type PointInTime struct {
	ID string `json:"id"`
}

type IndexMetadata struct {
	Index string `json:"_index"`
}

type BulkRequest struct {
	Update *IndexMetadata `json:"update,omitempty"`
	Index  *IndexMetadata `json:"index,omitempty"`
	Delete *IndexMetadata `json:"delete,omitempty"`
	Create *IndexMetadata `json:"create,omitempty"`
}

func (r BulkRequest) GetIndexMetadata() *IndexMetadata {
	if r.Create != nil {
		return r.Create
	} else if r.Delete != nil {
		return r.Delete
	} else if r.Index != nil {
		return r.Index
	} else if r.Update != nil {
		return r.Update
	}

	return nil
}

type MultipleGetRequest struct {
	Docs []IndexMetadata `json:"docs"`
}
