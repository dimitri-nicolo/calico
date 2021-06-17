// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/tigera/es-proxy/pkg/math"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/lma/pkg/httputils"

	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	esSearch "github.com/tigera/es-proxy/pkg/search"
)

const (
	maxNumResults   = 10000
)

type getIndex func(string) string

// SearchHandler is a handler for the /search endpoint.
//
// Uses a request body (JSON.blob) to extract parameters to build an elasticsearch query,
// executes it and returns the results.
func SearchHandler(getIndex getIndex, client *elastic.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body onto search parameters. If an error occurs while decoding define an http
		// error and return.
		params, err := parseRequestBodyForParams(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}
		// Search.
		response, err := search(getIndex, params, client)
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
		log.WithError(errInvalidMethod).Info("Invalid http method.")

		return nil, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    errInvalidMethod.Error(),
			Err:    errInvalidMethod,
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
	if len(params.ClusterName) == 0 {
		params.ClusterName = "cluster"
	}

	// Check that we are not attempting to enumerate more than the maximum number of results.
	if params.PageNum*params.PageSize > maxNumResults {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    "Page number overflow",
			Err:    errors.New("Page number / Page size combination is too large"),
		}
	}

	// At the moment, we only support a single sort by field.
	//TODO(rlb): Need to check the fields are valid for the index type. Maybe something else for the index helper.
	if len(params.SortBy) > 1 {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    "Too many sort fields specified",
			Err:    errors.New("Too many sort fields specified"),
		}
	}

	return &params, nil
}

// search returns the results of ES search.
func search(
	getIndex getIndex, params *v1.SearchRequest, esClient *elastic.Client,
) (*v1.SearchResponse, error) {
	query := elastic.NewBoolQuery()
	index := getIndex(params.ClusterName)

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

	rquery := &esSearch.Query{
		Query:    query,
		PageSize: params.PageSize,
		From:     params.PageNum * params.PageSize,
		Index:    index,
		Timeout:  params.Timeout.Duration,
		SortBy:   sortby,
	}

	result, err := esSearch.Hits(context.TODO(), esClient, rquery)
	if err != nil {
		log.WithError(err).Info("Error getting search results from elastic")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	cappedTotalHits := math.MinInt(int(result.TotalHits), maxNumResults)
	return &v1.SearchResponse{
		TimedOut:  result.TimedOut,
		Took:      metav1.Duration{Duration: time.Millisecond * time.Duration(result.TookInMillis)},
		NumPages:  ((cappedTotalHits - 1) / params.PageSize) + 1,
		TotalHits: int(result.TotalHits),
		Hits:      result.RawHits,
	}, nil
}
