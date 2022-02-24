// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package event

import (
	"context"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/datastore"
	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	elasticvariant "github.com/tigera/es-proxy/pkg/elastic"
	"github.com/tigera/es-proxy/pkg/math"
	"github.com/tigera/es-proxy/pkg/middleware"
	esSearch "github.com/tigera/es-proxy/pkg/search"
	lmaindex "github.com/tigera/lma/pkg/elastic/index"
	"github.com/tigera/lma/pkg/httputils"
)

// EventSearchHandler is a handler for the /events/search endpoint.
//
// Uses a request body (JSON.blob) to extract parameters to build an elasticsearch query,
// executes it and returns the results.
func EventSearchHandler(
	idxHelper lmaindex.Helper,
	k8sClient datastore.ClientSet,
	client *elastic.Client,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body onto search parameters. If an error occurs while decoding define an http
		// error and return.
		params, err := middleware.ParseRequestBodyForParams(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		// Search.
		response, err := search(idxHelper, params, k8sClient, client, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		// Encode reponse to writer. Handles an error.
		httputils.Encode(w, response)
	})
}

// search returns the results of ES search.
func search(
	idxHelper lmaindex.Helper,
	params *v1.SearchRequest,
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

	// Apply alert exceptions.
	eventExceptionList, err := k8sClient.AlertExceptions().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Error("failed to list alert exceptions")
	} else {
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

			alertException.Status.LastExecuted = &metav1.Time{Time: now}

			q, err := idxHelper.NewSelectorQuery(alertException.Spec.Selector)
			if err != nil {
				// skip invalid alert exception selector
				log.WithError(err).Errorf(`failed to create Elastic query from alert exception="%s" selector="%s"`,
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

	cappedTotalHits := math.MinInt(int(result.TotalHits), middleware.MaxSearchResults)
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
