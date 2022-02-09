// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	lmaerror "github.com/tigera/lma/pkg/api"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
)

var (
	ErrInvalidMethod = errors.New("Invalid http method")
	ErrParseRequest  = errors.New("Error parsing request parameters")
)

func createAndReturnError(err error, errorStr string, code int, featureID lmaerror.FeatureID, w http.ResponseWriter) {
	log.WithError(err).Info(errorStr)

	lmaError := lmaerror.Error{
		Code:    code,
		Message: errorStr,
		Feature: featureID,
	}

	responseJSON, err := json.Marshal(lmaError)
	if err != nil {
		log.WithError(err).Error("Error marshalling response to JSON")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_, err = w.Write(responseJSON)
	if err != nil {
		log.WithError(err).Infof("Error writing JSON: %v", responseJSON)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// NewMockSearchClient creates a mock client used for testing search results that will return
// an empty result before returning an error. This tests the case where we try to get every result
// from elasticsearch without needing to explicitly handle how many results we are expecting.
// This is a copy of the LMA version but with extra handling to deal with getting no results from elasticsearch.
func NewMockSearchClient(results []interface{}) lmaelastic.Client {
	idx := 0

	doFunc := func(_ context.Context, _ *elastic.SearchService) (*elastic.SearchResult, error) {
		// If there is no more results to give, return an empty
		// response once ONLY. If whatever is calling this will
		// loop forever and keeps calling the client, then it will
		// return an error on the next iteration.
		if idx == len(results) {
			idx++
			return new(elastic.SearchResult), nil
		}

		if idx > len(results) {
			return nil, errors.New("Enumerated past end of results")
		}

		result := results[idx]
		idx++

		switch rt := result.(type) {
		case *elastic.SearchResult:
			return rt, nil
		case elastic.SearchResult:
			return &rt, nil
		case error:
			return nil, rt
		case string:
			result := new(elastic.SearchResult)
			decoder := &elastic.DefaultDecoder{}
			err := decoder.Decode([]byte(rt), result)
			return result, err
		}

		return nil, errors.New("Unexpected result type")
	}

	return lmaelastic.NewMockClient(doFunc)
}
