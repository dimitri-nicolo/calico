// Copyright (c) 2022 Tigera All rights reserved.
package api_error

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// APIError extends Error() to specify a HTTP Status assosciated
// with the error
type APIError struct {
	StatusCode int
	Err        error
}

// Error retursn the error information with attached HTTP Status
func (r *APIError) Error() string {
	return fmt.Sprintf("status %d: err %v", r.StatusCode, r.Err)
}

// WriteAPIErrorToHeader logs the error and writes the HTTP status back to the client
func WriteAPIErrorToHeader(w http.ResponseWriter, apiErr *APIError) {
	log.WithError(apiErr.Err).Infof("returning status: %d", apiErr.StatusCode)
	WriteStatusErrorToHeader(w, apiErr.StatusCode)
}

// WriteAPIErrorToHeader writes the HTTP status back to the client
func WriteStatusErrorToHeader(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	_, err := w.Write([]byte(http.StatusText(status)))
	if err != nil {
		log.Errorf("Error when writing body to response: %v", err)
	}
}
