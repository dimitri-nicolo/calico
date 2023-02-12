// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.
package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	elasticvariant "github.com/projectcalico/calico/es-proxy/pkg/elastic"
	"github.com/projectcalico/calico/es-proxy/pkg/math"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	esSearch "github.com/projectcalico/calico/es-proxy/pkg/search"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const (
	defaultPageSize = 100
)

// SearchHandler is a handler for the /search endpoint.
//
// Uses a request body (JSON.blob) to extract parameters to build an elasticsearch query,
// executes it and returns the results.
func SearchHandler(idxHelper lmaindex.Helper, authReview middleware.AuthorizationReview, k8sClient datastore.ClientSet, esClient *elastic.Client, lsclient client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body onto search parameters. If an error occurs while decoding define an http
		// error and return.
		request, err := parseRequestBodyForParams(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		if lsclient == nil {
			//////////////////////////////////////////////////////////////////////////////////
			// TODO: Remove the need for this branch entirely. This is the legacy code.
			//       Just need to replace events implementation.
			//////////////////////////////////////////////////////////////////////////////////
			log.Warn("Using legacy search handler")
			response, err := search(idxHelper, request, authReview, k8sClient, esClient, r)
			if err != nil {
				httputils.EncodeError(w, err)
				return
			}

			// Encode reponse to writer. Handles an error.
			httputils.Encode(w, response)
			return
		}

		// Create a context with timeout to ensure we don't block for too long with this query.
		// This releases timer resources if the operation completes before the timeout.
		ctx, cancel := context.WithTimeout(r.Context(), request.Timeout.Duration)
		defer cancel()

		var response *v1.SearchResponse

		// Perform the search based on the type.
		id := idxHelper.GetIndex("")
		if strings.HasPrefix(id, "tigera_secure_ee_flows") {
			response, err = searchFlowLogs(ctx, lsclient, request, authReview, k8sClient)
		} else if strings.HasPrefix(id, "tigera_seucre_ee_dns") {
			response, err = searchDNSLogs(ctx, lsclient, request, authReview, k8sClient)
		} else {
			logrus.Fatal("BUG: Hit unreachable branch")
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

	// Initialize the search parameters to their default values.
	params := v1.SearchRequest{
		PageSize: defaultPageSize,
		Timeout:  &metav1.Duration{Duration: middleware.DefaultRequestTimeout},
	}

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
	if params.PageNum*params.PageSize > middleware.MaxNumResults {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    "page number overflow",
			Err:    errors.New("page number / Page size combination is too large"),
		}
	}

	// At the moment, we only support a single sort by field.
	// TODO(rlb): Need to check the fields are valid for the index type. Maybe something else for the
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

// intoLogParams converts a request into the given Linseed API parameters.
func intoLogParams(ctx context.Context, request *v1.SearchRequest, params lapi.LogParams, authReview middleware.AuthorizationReview) error {
	// Add in the selector.
	if len(request.Selector) > 0 {
		// Validate the selector. Linseed performs the same check, but
		// better to short-circuit the request if we can avoid it.
		_, err := lmaindex.FlowLogs().NewSelectorQuery(request.Selector)
		if err != nil {
			return err
		}
		params.SetSelector(request.Selector)
	}

	// Time range query.
	if request.TimeRange != nil {
		params.SetTimeRange(request.TimeRange)
	}

	// Get the user's permissions. We'll pass these to Linseed to filter out logs that
	// the user doens't have permission to view.
	verbs, err := authReview.PerformReviewForElasticLogs(ctx, request.ClusterName)
	if err != nil {
		return err
	}
	params.SetPermissions(verbs)

	// Validate the RBAC of the user, and return an Unauthorized error if the verbs don't allow any resources.
	// Linseed performs this same check, but better to short-circuit the request if we can avoid it.
	if _, err := lmaindex.FlowLogs().NewRBACQuery(verbs); err != nil {
		return err
	}

	// Configure sorting, if set.
	for _, s := range request.SortBy {
		if s.Field == "" {
			continue
		}
		params.SetSort([]lapi.SearchRequestSortBy{
			{
				Field:      s.Field,
				Descending: s.Descending,
			},
		})
	}

	// Configure pagination, timeout, etc.
	params.SetTimeout(request.Timeout)
	params.SetMaxResults(request.PageSize)
	if request.PageNum != 0 {
		// TODO: Ideally, clients don't know the format of the AfterKey. In order to satisfy
		// the exising UI API, we need to for now.
		params.SetAfterKey(map[string]interface{}{
			"startFrom": request.PageNum * request.PageSize,
		})
	}

	return nil
}

// searchFlowLogs calls searchLogs, configured for flow logs.
func searchFlowLogs(
	ctx context.Context,
	lsclient client.Client,
	request *v1.SearchRequest,
	authReview middleware.AuthorizationReview,
	k8sClient datastore.ClientSet,
) (*v1.SearchResponse, error) {
	params := &lapi.FlowLogParams{}
	err := intoLogParams(ctx, request, params, authReview)
	if err != nil {
		return nil, err
	}
	listFn := lsclient.FlowLogs(request.ClusterName).List
	return searchLogs(ctx, listFn, params, authReview, k8sClient)
}

// searchFlowLogs calls searchLogs, configured for DNS logs.
func searchDNSLogs(
	ctx context.Context,
	lsclient client.Client,
	request *v1.SearchRequest,
	authReview middleware.AuthorizationReview,
	k8sClient datastore.ClientSet,
) (*v1.SearchResponse, error) {
	params := &lapi.DNSLogParams{}
	err := intoLogParams(ctx, request, params, authReview)
	if err != nil {
		return nil, err
	}
	listFn := lsclient.DNSLogs(request.ClusterName).List
	return searchLogs(ctx, listFn, params, authReview, k8sClient)
}

// searchL7Logs calls searchLogs, configured for DNS logs.
func searchL7Logs(
	ctx context.Context,
	lsclient client.Client,
	request *v1.SearchRequest,
	authReview middleware.AuthorizationReview,
	k8sClient datastore.ClientSet,
) (*v1.SearchResponse, error) {
	params := &lapi.L7LogParams{}
	err := intoLogParams(ctx, request, params, authReview)
	if err != nil {
		return nil, err
	}
	listFn := lsclient.L7Logs(request.ClusterName).List
	return searchLogs(ctx, listFn, params, authReview, k8sClient)
}

// searchLogs performs a search against the Linseed API for logs that match the given
// parameters, using the provided client.ListFunc.
func searchLogs[T any](
	ctx context.Context,
	listFunc client.ListFunc[T],
	params lapi.LogParams,
	authReview middleware.AuthorizationReview,
	k8sClient datastore.ClientSet,
) (*v1.SearchResponse, error) {
	pageSize := params.GetMaxResults()

	// Perform the query.
	start := time.Now()
	items, err := listFunc(ctx, params)
	if err != nil {
		return nil, err
	}

	type Hit struct {
		ID     string `json:"id"`
		Index  string `json:"index"`
		Source T      `json:"source"`
	}

	// Build the hits response.
	hits := []json.RawMessage{}
	for i, item := range items.Items {
		hit := Hit{
			ID:     fmt.Sprintf("%d", i), // TODO - what does the UI use this for?
			Index:  "tigera_secure_ee",   // TODO: What does the UI use this for?
			Source: item,
		}
		hitJSON, err := json.Marshal(hit)
		if err != nil {
			return nil, err
		}
		hits = append(hits, hitJSON)
	}

	// Calculate the number of pages, given the request's page size.
	cappedTotalHits := math.MinInt(int(items.TotalHits), middleware.MaxNumResults)
	numPages := 0
	if pageSize > 0 {
		numPages = ((cappedTotalHits - 1) / pageSize) + 1
	}

	return &v1.SearchResponse{
		TimedOut:  false, // TODO: Is this used?
		Took:      metav1.Duration{Duration: time.Since(start)},
		NumPages:  numPages,
		TotalHits: int(items.TotalHits),
		Hits:      hits,
	}, nil
}

// Legacy search implementation. Keeping around while we convert.
func search(
	idxHelper lmaindex.Helper,
	params *v1.SearchRequest,
	authReview middleware.AuthorizationReview,
	k8sClient datastore.ClientSet,
	esClient *elastic.Client,
	r *http.Request,
) (*v1.SearchResponse, error) {
	// Create a context with timeout to ensure we don't block for too long with this query.
	// This releases timer resources if the operation completes before the timeout.
	ctx, cancelWithTimeout := context.WithTimeout(r.Context(), params.Timeout.Duration)
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

		now := &metav1.Time{Time: time.Now()}
		for _, alertException := range eventExceptionList.Items {
			if alertException.Spec.StartTime.Before(now) {
				if alertException.Spec.EndTime != nil && alertException.Spec.EndTime.Before(now) {
					// skip expired alert exceptions
					log.Debugf(`skipping expired alert exception="%s"`, alertException.GetName())
					continue
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
	}

	// Sorting.
	var sortby []esSearch.SortBy
	for _, s := range params.SortBy {
		if s.Field == "" {
			continue
		}
		// TODO(rlb): Maybe include other fields automatically based on selected field.
		sortby = append(sortby, esSearch.SortBy{
			Field:     s.Field,
			Ascending: !s.Descending,
		})
	}

	// Configure paging.

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

	cappedTotalHits := math.MinInt(int(result.TotalHits), middleware.MaxNumResults)
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
