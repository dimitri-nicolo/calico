// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	esSearch "github.com/tigera/es-proxy/pkg/search"
	httpUtils "github.com/tigera/es-proxy/pkg/utils"

	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
)

type getIndex func(string) string

type SearchError struct {
	// Status http status code of the request error.
	Status int

	// Parcing error message.
	Msg string

	// Error cause of parcing error.
	Err error
}

type SearchParams struct {
	// ClusterName defines the name of the cluster a connection will be performed on.
	ClusterName string `json:"cluster" validate:"omitempty"`

	// PageSize defines the page size of raw flow logs to retrieve per search. [Default: 100]
	PageSize int `json:"page_size" validate:"gte=0,lte=1000"`

	// SearchAfter defines sort values that indicates which docs this request should "search after".
	// [Default: nil]
	SearchAfter interface{} `json:"search_after" validate:"omitempty"`
}

// decodeRequestBody sets the search parameters to their default values.
func (params *SearchParams) DefaultParams() {
	params.ClusterName = "cluster"
	params.PageSize = 100
}

// Error implementation of error type Error function, which returns the malformed request message
// as a string.
func (ee *SearchError) Error() string {
	return ee.Msg
}

// SearchHandler is a handler for the /search endpoint.
//
// Uses a request body (JSON.blob) to extract parameters to build an elasticsearch query,
// executes it and returns the results.
func SearchHandler(getIndex getIndex, client *elastic.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body onto search parameters. If an error occurs while decoding define an http
		// error and return.
		params, perr := parseRequestBodyForParams(w, r)
		if perr != nil {
			var se *SearchError
			if errors.As(perr, &se) {
				http.Error(w, se.Msg, se.Status)
			} else {
				http.Error(w, perr.Error(), http.StatusInternalServerError)
			}
			return
		}
		// Search.
		response, serr := search(getIndex, params, client)
		if serr != nil {
			var mr *httpUtils.MalformedRequest
			var se *SearchError
			if errors.As(serr, &se) {
				http.Error(w, se.Msg, se.Status)
			} else if errors.As(serr, &mr) {
				http.Error(w, mr.Msg, mr.Status)
			} else {
				http.Error(w, serr.Error(), http.StatusInternalServerError)
			}
			return
		}
		// Encode reponse to writer.
		if eerr := httpUtils.Encode(w, response); eerr != nil {
			var ee *SearchError
			if errors.As(eerr, &ee) {
				http.Error(w, ee.Msg, ee.Status)
			} else {
				http.Error(w, eerr.Error(), http.StatusInternalServerError)
			}
		}
	})
}

// parseRequestBodyForParams extracts query parameters from the request body (JSON.blob) and
// validates them.
//
// Will define an http.Error if an error occurs.
func parseRequestBodyForParams(w http.ResponseWriter, r *http.Request) (*SearchParams, error) {
	// Validate http method.
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		log.WithError(errInvalidMethod).Info("Invalid http method.")

		return nil, &SearchError{
			Status: http.StatusMethodNotAllowed,
			Msg:    errInvalidMethod.Error(),
			Err:    errInvalidMethod,
		}
	}

	var params SearchParams
	params.DefaultParams()

	// Decode the http request body into the struct.
	if err := httpUtils.Decode(w, r, &params); err != nil {
		var mr *httpUtils.MalformedRequest
		if errors.As(err, &mr) {
			log.WithError(mr.Err).Info(mr.Msg)
			return nil, mr
		} else {
			log.WithError(mr.Err).Info("Error validating /search request.")
			return nil, &SearchError{
				Status: http.StatusMethodNotAllowed,
				Msg:    http.StatusText(http.StatusInternalServerError),
				Err:    err,
			}
		}
	}

	// Validate parameters.
	if err := validator.Validate(params); err != nil {
		return nil, &SearchError{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	// Set cluster name to default: "cluster", if empty.
	if len(params.ClusterName) == 0 {
		params.ClusterName = "cluster"
	}

	return &params, nil
}

// search returns the results of ES search.
func search(
	getIndex getIndex, params *SearchParams, esClient *elastic.Client,
) (*esSearch.ESResults, error) {
	query := elastic.NewBoolQuery()
	index := getIndex(params.ClusterName)

	rquery := &esSearch.Query{
		Query:       query,
		PageSize:    params.PageSize,
		SearchAfter: params.SearchAfter,
		Index:       index,
		Timeout:     60 * time.Second,
	}

	result, err := esSearch.Hits(context.TODO(), esClient, rquery)
	if err != nil {
		log.WithError(err).Info("Error getting search results from elastic")
		return result, &SearchError{Status: http.StatusInternalServerError, Msg: err.Error(), Err: err}
	}
	return result, nil
}
