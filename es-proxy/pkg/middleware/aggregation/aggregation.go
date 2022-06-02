// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package aggregation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	elasticvariant "github.com/projectcalico/calico/es-proxy/pkg/elastic"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	"github.com/projectcalico/calico/lma/pkg/k8s"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

// This file implements an aggregated data query handler. The primary use of this is for the UX when querying aggregated
// data for specific service graph nodes and edges. This is essentially a wrapper around the elasticsearch _search
// interface, but provides the following useful additional function:
// - Selector based queries (can select data using a selector format)
// - RBAC limited data
// - Handling of time series with auto selection of time buckets to avoid querying excessive amounts of data.
// - Hard coded limits on the aggregation hits.

const (
	defaultRequestTimeout  = 60 * time.Second
	minAggregationInterval = 10 * time.Minute
	minTimeBuckets         = 4
	maxTimeBuckets         = 24
	timeBucket             = "tb"
)

func NewAggregationHandler(
	elasticClient lmaelastic.Client,
	clientSetFactory k8s.ClientSetFactory,
	indexHelper lmaindex.Helper,
) http.Handler {
	return NewAggregationHandlerWithBackend(indexHelper, &realAggregationBackend{
		elastic:          elasticClient,
		clientSetFactory: clientSetFactory,
	})
}

func NewAggregationHandlerWithBackend(indexHelper lmaindex.Helper, backend AggregationBackend) http.Handler {
	return &aggregation{
		backend:     backend,
		indexHelper: indexHelper,
	}
}

// RequestData encapsulates data parsed from the request that is shared between the various components that construct
// the service graph.
type RequestData struct {
	HTTPRequest        *http.Request
	AggregationRequest v1.AggregationRequest
	Index              string
	IsTimeSeries       bool
	Aggregations       map[string]elastic.Aggregation
}

// aggregation implements the Aggregation interface.
type aggregation struct {
	backend     AggregationBackend
	indexHelper lmaindex.Helper
}

func (s *aggregation) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	// Extract the request specific data used to collate and filter the data.
	rd, err := s.getAggregationRequest(w, req)
	if err != nil {
		httputils.EncodeError(w, err)
		return
	}

	// Construct a context with timeout based on the service graph request.
	ctx, cancel := context.WithTimeout(req.Context(), rd.AggregationRequest.Timeout)
	defer cancel()

	// Create the elastic query.
	res := v1.AggregationResponse{}
	if sel, err := s.indexHelper.NewSelectorQuery(rd.AggregationRequest.Selector); err != nil {
		httputils.EncodeError(w, err)
		return
	} else if auth, err := s.backend.PerformUserAuthorizationReview(ctx, rd); err != nil {
		httputils.EncodeError(w, err)
		return
	} else if query, err := s.getQuery(rd, sel, auth); err != nil {
		httputils.EncodeError(w, err)
		return
	} else if aggs, err := s.backend.RunQuery(ctx, rd, query); err != nil {
		httputils.EncodeError(w, err)
		return
	} else if !rd.IsTimeSeries {
		// There is no time series, therefore the data is all in the main bucket.
		res.Buckets = append(res.Buckets, v1.AggregationTimeBucket{
			StartTime:    metav1.Time{Time: rd.AggregationRequest.TimeRange.From},
			Aggregations: aggs,
		})
	} else {
		// There is a time series. The time aggregation is in the main bucket and then the data for each time
		// bucket is in the sub aggregation.
		timebuckets, ok := aggs.AutoDateHistogram(timeBucket)
		if !ok {
			httputils.EncodeError(w, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    "no valid time buckets in aggregation response",
				Err:    errors.New("no valid time buckets in aggregation response"),
			})
			return
		}
		for _, b := range timebuckets.Buckets {
			// Pull out the aggregation results.
			results := make(map[string]json.RawMessage)
			for an := range rd.AggregationRequest.Aggregations {
				results[an] = b.Aggregations[an]
			}

			// Elasticsearch stores dates in milliseconds since the epoch.
			res.Buckets = append(res.Buckets, v1.AggregationTimeBucket{
				StartTime:    metav1.Time{Time: time.Unix(int64(b.Key)/1000, 0)},
				Aggregations: results,
			})
		}
	}

	httputils.Encode(w, res)
	log.Debugf("Aggregation request took %s", time.Since(start))
}

type rawAgg struct {
	json.RawMessage
}

func (a rawAgg) Source() (interface{}, error) {
	return a.RawMessage, nil
}

// getAggregationRequest parses the request from the HTTP request body.
func (s *aggregation) getAggregationRequest(w http.ResponseWriter, req *http.Request) (*RequestData, error) {
	// Extract the request from the body.
	var ar v1.AggregationRequest

	if err := httputils.Decode(w, req, &ar); err != nil {
		return nil, err
	}

	// Validate parameters.
	if err := validator.Validate(ar); err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    fmt.Sprintf("Request body contains invalid data: %v", err),
			Err:    err,
		}
	}

	if ar.Timeout == 0 {
		ar.Timeout = defaultRequestTimeout
	}
	if ar.Cluster == "" {
		ar.Cluster = "cluster"
	}
	if len(ar.Aggregations) == 0 {
		return nil, httputils.NewHttpStatusErrorBadRequest("Request body contains no aggregations", nil)
	}

	// If the request is for a time series then determine the number of buckets based on the interval size and the
	// minimum sample period.
	var buckets int
	var aggs map[string]elastic.Aggregation
	if buckets = getNumBuckets(ar); buckets != 0 {
		// Modify the user supplied aggregation to be nested within a histogram aggregation for the time field.
		adagg := elastic.NewAutoDateHistogramAggregation().
			Field(s.indexHelper.GetTimeField()).
			Buckets(buckets)
		for an, av := range ar.Aggregations {
			adagg = adagg.SubAggregation(an, rawAgg{av})
		}
		aggs = map[string]elastic.Aggregation{
			timeBucket: adagg,
		}
	} else {
		aggs = make(map[string]elastic.Aggregation)
		for an, av := range ar.Aggregations {
			aggs[an] = rawAgg{av}
		}
	}

	return &RequestData{
		HTTPRequest:        req,
		AggregationRequest: ar,
		Index:              s.indexHelper.GetIndex(elasticvariant.AddIndexInfix(ar.Cluster)),
		IsTimeSeries:       buckets != 0,
		Aggregations:       aggs,
	}, nil
}

// getNumBuckets returns the max number of buckets to request for a time series.
func getNumBuckets(ar v1.AggregationRequest) int {
	if !ar.IncludeTimeSeries {
		return 0
	}

	// Each bucket should be a least _minAggregationInterval_, and we always want at least _minTimeBuckets_ data points.
	// Determine the ideal number of buckets, maxing out at _maxTimeBuckets_.
	duration := ar.TimeRange.Duration()

	numMinIntervals := duration / minAggregationInterval
	if numMinIntervals < minTimeBuckets {
		return 0
	} else if numMinIntervals <= maxTimeBuckets {
		return int(numMinIntervals)
	} else {
		return maxTimeBuckets
	}
}

// getQuery returns the query for the aggregation. This is a combination of the users selector, a users RBAC limiting
// query, and the time range.
func (s *aggregation) getQuery(rd *RequestData, sel elastic.Query, auth []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	q := elastic.NewBoolQuery()
	if sel != nil {
		q = q.Must(sel)
	}

	if rbac, err := s.indexHelper.NewRBACQuery(auth); err != nil {
		return nil, err
	} else if rbac != nil {
		q = q.Must(rbac)
	}

	tr := s.indexHelper.NewTimeRangeQuery(rd.AggregationRequest.TimeRange.From, rd.AggregationRequest.TimeRange.To)
	q = q.Must(tr)

	return q, nil
}
