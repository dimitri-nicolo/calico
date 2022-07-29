// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package process

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	elasticvariant "github.com/projectcalico/calico/es-proxy/pkg/elastic"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	defaultAggregationSize = 1000

	sourceNameAggrKey = "agg-source_name_aggr"
	processNameKey    = "agg-process_name"
	processIDKey      = "agg-process_id"
)

// ProcessHandler handles process instance requests from manager dashboard.
func ProcessHandler(
	idxHelper lmaindex.Helper,
	authReview middleware.AuthorizationReview,
	client *elastic.Client,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, err := parseProcessRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		resp, err := processProcessRequest(idxHelper, params, authReview, client, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		httputils.Encode(w, resp)
	})
}

// parseProcessRequest extracts parameters from the request body and validates them.
func parseProcessRequest(w http.ResponseWriter, r *http.Request) (*v1.ProcessRequest, error) {
	// Validate http method.
	if r.Method != http.MethodPost {
		log.WithError(middleware.ErrInvalidMethod).Info("Invalid http method.")

		return nil, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    middleware.ErrInvalidMethod.Error(),
			Err:    middleware.ErrInvalidMethod,
		}
	}

	// Decode the http request body into the struct.
	var params v1.ProcessRequest

	if err := httputils.Decode(w, r, &params); err != nil {
		var mr *httputils.HttpStatusError
		if errors.As(err, &mr) {
			log.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			log.WithError(mr.Err).Info("Error validating process requests.")
			return nil, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    http.StatusText(http.StatusInternalServerError),
				Err:    err,
			}
		}
	}

	// Set cluster name to default: "cluster", if empty.
	if params.ClusterName == "" {
		params.ClusterName = middleware.MaybeParseClusterNameFromRequest(r)
	}

	return &params, nil
}

// processProcessRequest translates process request parameters to Elastic queries and return responses.
func processProcessRequest(
	idxHelper lmaindex.Helper,
	params *v1.ProcessRequest,
	authReview middleware.AuthorizationReview,
	client *elastic.Client,
	r *http.Request,
) (*v1.ProcessResponse, error) {
	// create a context with timeout to ensure we don't block for too long.
	ctx, cancelWithTimeout := context.WithTimeout(r.Context(), middleware.DefaultRequestTimeout)
	defer cancelWithTimeout()

	query, err := getQuery(ctx, idxHelper, params, authReview)
	if err != nil {
		log.WithError(err).Info("Error getting elastic query for process requests.")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	aggregation, err := getAggregation(client)
	if err != nil {
		log.WithError(err).Info("Error getting elastic aggregations for process requests.")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	// perform elastic search with aggregation
	index := idxHelper.GetIndex(elasticvariant.AddIndexInfix(params.ClusterName))
	search := client.Search(index).
		Query(query).
		Aggregation(sourceNameAggrKey, aggregation).
		Size(0) // we don't need hits

	res, err := search.Do(ctx)
	if err != nil {
		log.WithError(err).Info("Error getting search results from elastic.")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	sourceNameAggItems, found := res.Aggregations.Terms(sourceNameAggrKey)
	if !found {
		err := fmt.Errorf("failed to get bucket key %s in aggregation from search results", sourceNameAggrKey)
		log.WithError(err).Info("Error parsing search results.")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	var processes []v1.Process
	for _, b := range sourceNameAggItems.Buckets {
		if endpoint, ok := b.Key.(string); !ok {
			log.Warnf("failed to convert bucket key %v to string", b.Key)
			continue
		} else {
			if processNameItems, found := b.Aggregations.Terms(processNameKey); !found {
				log.Warnf("failed to get bucket key %s in sub-aggregation", processNameKey)
				continue
			} else {
				for _, bb := range processNameItems.Buckets {
					if processName, ok := bb.Key.(string); !ok {
						log.Warnf("failed to convert bucket key %v to string", bb.Key)
						continue
					} else {
						if processIDItems, found := bb.Aggregations.Terms(processIDKey); !found {
							log.Warnf("failed to get bucket key %s in sub-aggregation", processIDKey)
							continue
						} else {
							process := v1.Process{
								Name:          processName,
								Endpoint:      endpoint,
								InstanceCount: len(processIDItems.Buckets),
							}

							processes = append(processes, process)
						}
					}
				}
			}
		}
	}

	return &v1.ProcessResponse{
		Processes: processes,
	}, nil
}

// getQuery returns the query for flow log elastic search.
func getQuery(
	ctx context.Context,
	idxHelper lmaindex.Helper,
	params *v1.ProcessRequest,
	authReview middleware.AuthorizationReview,
) (*elastic.BoolQuery, error) {
	esquery := elastic.NewBoolQuery()

	// Selector query.
	var selector elastic.Query
	var err error
	// we want the flow came from the pod that initiated the connection.
	// see: https://docs.tigera.io/v3.14/visibility/elastic/flow/datatypes
	if params.Selector != "" {
		params.Selector += " AND reporter = src"
	} else {
		params.Selector = "reporter = src"
	}
	selector, err = idxHelper.NewSelectorQuery(params.Selector)
	if err != nil {
		// NewSelectorQuery returns an HttpStatusError.
		return nil, err
	}
	esquery = esquery.Must(selector)

	// Time range query.
	if params.TimeRange == nil {
		now := time.Now()
		params.TimeRange = &lmav1.TimeRange{
			From: now.Add(-middleware.DefaultRequestTimeRange),
			To:   now,
		}
	}
	timeRange := idxHelper.NewTimeRangeQuery(params.TimeRange.From, params.TimeRange.To)
	esquery = esquery.Filter(timeRange)

	// Rbac query.
	verbs, err := authReview.PerformReviewForElasticLogs(ctx, params.ClusterName)
	if err != nil {
		return nil, err
	}
	if rbac, err := idxHelper.NewRBACQuery(verbs); err != nil {
		// NewRBACQuery returns an HttpStatusError.
		return nil, err
	} else if rbac != nil {
		esquery = esquery.Filter(rbac)
	}

	// exclude process_name in ["-", "*"] and process_id in "*"
	excludes := []elastic.Query{
		elastic.NewTermsQuery("process_name", "-", "*"),
		elastic.NewTermQuery("process_id", "*"),
	}
	esquery = esquery.MustNot(excludes...)

	return esquery, nil
}

// getAggregation returns the aggregations for flow log elastic search.
func getAggregation(esClient *elastic.Client) (*elastic.TermsAggregation, error) {
	// aggregation
	// "aggs": {
	//     "agg-source_name_aggr": {
	//       "terms": {
	//         "field": "source_name_aggr",
	//         "size": 1000
	//       },
	//       "aggs": {
	//         "agg-process_name": {
	//           "terms": {
	//             "field": "process_name",
	//             "size": 1000
	//           },
	//           "aggs": {
	//             "agg-process_id": {
	//               "terms": {
	//                 "field": "process_id",
	//                 "size": 1000
	//               }
	//             }
	//           }
	//         }
	//       }
	//     }
	//   }
	aggSourceNameAggr := elastic.NewTermsAggregation()
	aggSourceNameAggr.Field("source_name_aggr")
	aggSourceNameAggr.Size(defaultAggregationSize)

	aggProcessName := elastic.NewTermsAggregation()
	aggProcessName.Field("process_name")
	aggProcessName.Size(defaultAggregationSize)
	aggSourceNameAggr.SubAggregation(processNameKey, aggProcessName)

	aggProcessID := elastic.NewTermsAggregation()
	aggProcessID.Field("process_id")
	aggProcessID.Size(defaultAggregationSize)
	aggProcessName.SubAggregation(processIDKey, aggProcessID)

	return aggSourceNameAggr, nil
}
