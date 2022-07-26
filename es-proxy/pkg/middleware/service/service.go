// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	elasticvariant "github.com/projectcalico/calico/es-proxy/pkg/elastic"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	esSearch "github.com/projectcalico/calico/es-proxy/pkg/search"
	"github.com/projectcalico/calico/lma/pkg/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	httpStatusServerErrorUpperBound = 599
)

// ServiceHandler handles service requests from manager dashboard.
func ServiceHandler(
	authReview middleware.AuthorizationReview,
	client *elastic.Client,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, err := parseServiceRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		resp, err := processServiceRequest(params, authReview, client, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		httputils.Encode(w, resp)
	})
}

// parseServiceRequest extracts parameters from the request body and validates them.
func parseServiceRequest(w http.ResponseWriter, r *http.Request) (*v1.ServiceRequest, error) {
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
	var params v1.ServiceRequest

	if err := httputils.Decode(w, r, &params); err != nil {
		var mr *httputils.HttpStatusError
		if errors.As(err, &mr) {
			log.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			log.WithError(mr.Err).Info("Error validating service requests.")
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

type service struct {
	Count            int
	SourceNameAggr   string
	TotalBytesIn     int           // sum(bytes_in)
	TotalBytesOut    int           // sum(bytes_out)
	TotalDuration    time.Duration // sum(duration_mean * count) in milliseconds
	TotalLatency     time.Duration // sum(latency) in milliseconds
	TotalErrorCount  int           // count(http response_code 400-599)
	TotalLogDuration int64         // sum(end_time - start_time) in seconds
}

// processServiceRequest translates service request parameters to Elastic queries and return responses.
func processServiceRequest(
	params *v1.ServiceRequest,
	authReview middleware.AuthorizationReview,
	client *elastic.Client,
	r *http.Request,
) (*v1.ServiceResponse, error) {
	res, err := search(params, authReview, client, r)
	if err != nil {
		return nil, err
	}

	serviceMap := make(map[string]*service)
	for _, rawHit := range res.RawHits {
		var doc l7Doc
		if err := json.Unmarshal(rawHit, &doc); err != nil {
			log.WithError(err).Warnf("failed to unmarshal L7 raw hit: %s", rawHit)
			continue
		}

		sourceNameAggr := doc.Source.SourceNameAggr
		if sourceNameAggr != api.FlowLogNetworkPrivate && sourceNameAggr != api.FlowLogNetworkPublic {
			errCount := 0
			if responseCode, err := strconv.Atoi(doc.Source.ResponseCode); err == nil {
				// Count HTTP error responses from 400 - 499 (client error) + 500 - 599 (server error)
				if responseCode >= http.StatusBadRequest && responseCode <= httpStatusServerErrorUpperBound {
					errCount = doc.Source.Count
				}
			}

			if s, found := serviceMap[sourceNameAggr]; found {
				s.Count += doc.Source.Count
				s.TotalBytesIn += doc.Source.BytesIn
				s.TotalBytesOut += doc.Source.BytesOut
				s.TotalDuration += doc.Source.DurationMean * time.Duration(doc.Source.Count)
				s.TotalLatency += time.Duration(doc.Source.Latency)
				s.TotalErrorCount += errCount
				s.TotalLogDuration += doc.Source.EndTime - doc.Source.StartTime
			} else {
				serviceMap[sourceNameAggr] = &service{
					Count:            doc.Source.Count,
					SourceNameAggr:   sourceNameAggr,
					TotalBytesIn:     doc.Source.BytesIn,
					TotalBytesOut:    doc.Source.BytesOut,
					TotalDuration:    doc.Source.DurationMean * time.Duration(doc.Source.Count),
					TotalLatency:     time.Duration(doc.Source.Latency),
					TotalErrorCount:  errCount,
					TotalLogDuration: doc.Source.EndTime - doc.Source.StartTime,
				}
			}
		}
	}

	services := make([]v1.Service, 0)
	for k, v := range serviceMap {
		service := v1.Service{
			Name:               k,
			ErrorRate:          float64(v.TotalErrorCount) / float64(v.Count) * 100,        // %
			Latency:            float64(v.TotalDuration.Microseconds()) / float64(v.Count), // microseconds
			InboundThroughput:  float64(v.TotalBytesIn) / v.TotalDuration.Seconds(),        // bytes/second
			OutboundThroughput: float64(v.TotalBytesOut) / v.TotalDuration.Seconds(),       // bytes/second
			RequestThroughput:  float64(v.Count) / float64(v.TotalLogDuration),             // /second
		}
		services = append(services, service)
	}

	return &v1.ServiceResponse{
		Services: services,
	}, nil
}

// search returns the results of ES search.
func search(
	params *v1.ServiceRequest,
	authReview middleware.AuthorizationReview,
	esClient *elastic.Client,
	r *http.Request,
) (*esSearch.ESResults, error) {
	// create a context with timeout to ensure we don't block for too long.
	ctx, cancelWithTimeout := context.WithTimeout(r.Context(), middleware.DefaultRequestTimeout)
	defer cancelWithTimeout()

	// Get service details from L7 ApplicationLayer logs.
	idxHelper := lmaindex.L7Logs()
	index := idxHelper.GetIndex(elasticvariant.AddIndexInfix(params.ClusterName))

	esquery := elastic.NewBoolQuery()
	// Selector query.
	if len(params.Selector) > 0 {
		selector, err := idxHelper.NewSelectorQuery(params.Selector)
		if err != nil {
			// NewSelectorQuery returns an HttpStatusError.
			return nil, err
		}
		esquery = esquery.Must(selector)
	}

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

	query := &esSearch.Query{
		EsQuery:  esquery,
		Index:    index,
		PageSize: middleware.MaxNumResults,
		Timeout:  middleware.DefaultRequestTimeout,
	}

	result, err := esSearch.Hits(ctx, query, esClient)
	if err != nil {
		log.WithError(err).Info("Error getting search results from elastic")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	return result, nil
}
