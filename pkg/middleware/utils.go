// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	lmaerror "github.com/tigera/lma/pkg/api"
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

	return
}
