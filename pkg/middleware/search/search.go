// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.
package search

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"

	"github.com/tigera/compliance/pkg/datastore"
	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	elasticvariant "github.com/tigera/es-proxy/pkg/elastic"
	"github.com/tigera/es-proxy/pkg/math"
	"github.com/tigera/es-proxy/pkg/middleware"
	esSearch "github.com/tigera/es-proxy/pkg/search"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	maxNumResults = 10000
)

// SearchHandler is a handler for the /search endpoint.
//
// Uses a request body (JSON.blob) to extract parameters to build an elasticsearch query,
// executes it and returns the results.
func SearchHandler(
	idxHelper lmaindex.Helper,
	authReview middleware.AuthorizationReview,
	k8sClient datastore.ClientSet,
	client *elastic.Client,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body onto search parameters. If an error occurs while decoding define an http
		// error and return.
		params, err := parseRequestBodyForParams(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		// Search.
		response, err := search(idxHelper, params, authReview, k8sClient, client, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		// Encode reponse to writer. Handles an error.
		httputils.Encode(w, response)
	})
}

// parseRequestBodyForParams extracts query parameters from the request body (JSON.blob) and
// validates them.
//
// Will define an http.Error if an error occurs.
func parseRequestBodyForParams(w http.ResponseWriter, r *http.Request) (*v1.SearchRequest, error) {
	// Validate http method.
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		log.WithError(middleware.ErrInvalidMethod).Info("Invalid http method.")

		return nil, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    middleware.ErrInvalidMethod.Error(),
			Err:    middleware.ErrInvalidMethod,
		}
	}

	var params v1.SearchRequest
	params.DefaultParams()

	// Decode the http request body into the struct.
	if err := httputils.Decode(w, r, &params); err != nil {
		var mr *httputils.HttpStatusError
		if errors.As(err, &mr) {
			log.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			log.WithError(mr.Err).Info("Error validating /search request.")
			return nil, &httputils.HttpStatusError{
				Status: http.StatusMethodNotAllowed,
				Msg:    http.StatusText(http.StatusInternalServerError),
				Err:    err,
			}
		}
	}

	// Validate parameters.
	if err := validator.Validate(params); err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	// Set cluster name to default: "cluster", if empty.
	if params.ClusterName == "" {
		params.ClusterName = middleware.MaybeParseClusterNameFromRequest(r)
	}

	// Check that we are not attempting to enumerate more than the maximum number of results.
	if params.PageNum*params.PageSize > maxNumResults {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    "page number overflow",
			Err:    errors.New("page number / Page size combination is too large"),
		}
	}

	// At the moment, we only support a single sort by field.
	//TODO(rlb): Need to check the fields are valid for the index type. Maybe something else for the
	// index helper.
	if len(params.SortBy) > 1 {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    "too many sort fields specified",
			Err:    errors.New("too many sort fields specified"),
		}
	}

	return &params, nil
}

// search returns the results of ES search.
func search(
	idxHelper lmaindex.Helper,
	params *v1.SearchRequest,
	authReview middleware.AuthorizationReview,
	k8sClient datastore.ClientSet,
	esClient *elastic.Client,
	r *http.Request,
) (*v1.SearchResponse, error) {
	// Create a context with timeout to ensure we don't block for too long with this query.
	ctx, cancelWithTimeout := context.WithTimeout(r.Context(), params.Timeout.Duration)
	// Releases timer resources if the operation completes before the timeout.
	defer cancelWithTimeout()

	index := idxHelper.GetIndex(elasticvariant.AddIndexInfix(params.ClusterName))

	esquery := elastic.NewBoolQuery()
	// Selector query.
	var selector elastic.Query
	var err error
	if len(params.Selector) > 0 {
		selector, err = idxHelper.NewSelectorQuery(params.Selector)
		if err != nil {
			// NewSelectorQuery returns an HttpStatusError.
			return nil, err
		}
		esquery = esquery.Must(selector)
	}
	if len(params.Filter) > 0 {
		for _, filter := range params.Filter {
			q := elastic.NewRawStringQuery(string(filter))
			esquery = esquery.Filter(q)
		}
	}

	// Time range query.
	if params.TimeRange != nil {
		timeRange := idxHelper.NewTimeRangeQuery(params.TimeRange.From, params.TimeRange.To)
		esquery = esquery.Filter(timeRange)
	}

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

	// For security event search requests, we need to modify the Elastic query
	// to exclude events which match exceptions created by users.
	if strings.Contains(index, lmaelastic.EventsIndex) {
		eventExceptionList, err := k8sClient.AlertExceptions().List(ctx, metav1.ListOptions{})
		if err != nil {
			log.WithError(err).Error("failed to list alert exceptions")
			return nil, &httputils.HttpStatusError{
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
				Err:    err,
			}
		}

		now := time.Now()
		for _, alertException := range eventExceptionList.Items {
			if alertException.Spec.Period != nil {
				createTimestamp := alertException.GetCreationTimestamp()
				if createTimestamp.Add(alertException.Spec.Period.Duration).Before(now) {
					// skip expired alert exceptions
					log.Debugf(`skipping expired alert exception="%s"`, alertException.GetName())
					continue
				}
			}

			q, err := idxHelper.NewSelectorQuery(alertException.Spec.Selector)
			if err != nil {
				// skip invalid alert exception selector
				log.WithError(err).Warnf(`ignoring alert exception="%s", failed to parse selector="%s"`,
					alertException.GetName(), alertException.Spec.Selector)
				continue
			}

			esquery = esquery.MustNot(q)
		}
	}

	// Sorting.
	var sortby []esSearch.SortBy
	for _, s := range params.SortBy {
		if s.Field == "" {
			continue
		}
		//TODO(rlb): Maybe include other fields automatically based on selected field.
		sortby = append(sortby, esSearch.SortBy{
			Field:     s.Field,
			Ascending: !s.Descending,
		})
	}

	query := &esSearch.Query{
		EsQuery:  esquery,
		PageSize: params.PageSize,
		From:     params.PageNum * params.PageSize,
		Index:    index,
		Timeout:  params.Timeout.Duration,
		SortBy:   sortby,
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

	cappedTotalHits := math.MinInt(int(result.TotalHits), maxNumResults)
	numPages := 0
	if params.PageSize > 0 {
		numPages = ((cappedTotalHits - 1) / params.PageSize) + 1
	}

	return &v1.SearchResponse{
		TimedOut:  result.TimedOut,
		Took:      metav1.Duration{Duration: time.Millisecond * time.Duration(result.TookInMillis)},
		NumPages:  numPages,
		TotalHits: int(result.TotalHits),
		Hits:      result.RawHits,
	}, nil
}
