// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.
package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	lmaerror "github.com/projectcalico/calico/lma/pkg/api"
)

const (
	DefaultRequestTimeout = 60 * time.Second

	MaxNumResults     = 10000
	MaxResultsPerPage = 1000

	clusterParam       = "cluster"
	clusterIdHeader    = "x-cluster-id"
	defaultClusterName = "cluster"
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

	w.WriteHeader(code)
	_, err = w.Write(responseJSON)
	if err != nil {
		log.WithError(err).Infof("Error writing JSON: %v", responseJSON)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func MaybeParseClusterNameFromRequest(r *http.Request) string {
	clusterName := defaultClusterName
	if r != nil && r.Header != nil {
		xClusterID := r.Header.Get(clusterIdHeader)
		if xClusterID != "" {
			clusterName = xClusterID
		}
	}
	return clusterName
}
