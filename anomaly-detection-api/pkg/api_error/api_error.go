// Copyright (c) 2022 Tigera All rights reserved.
package api_error

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type APIError struct {
	StatusCode int
	Err        error
}

func (r *APIError) Error() string {
	return fmt.Sprintf("status %d: err %v", r.StatusCode, r.Err)
}

func WriteAPIErrorToHeader(w http.ResponseWriter, apiErr *APIError) {
	log.WithError(apiErr.Err).Infof("returning status: %d", apiErr.StatusCode)
	WriteStatusErrorToHeader(w, apiErr.StatusCode)
}

func WriteStatusErrorToHeader(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	_, err := w.Write([]byte(http.StatusText(status)))
	if err != nil {
		log.Errorf("Error when writing body to response: %v", err)
	}
}
