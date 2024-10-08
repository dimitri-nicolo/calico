package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// httpError is the base from which to create errors specifically related to HTTP
// processing, which will have a custom error code.
type httpError struct {
	Err      error  `json:"-"`
	ErrMsg   string `json:"errorMessage"`
	ErrCode  string `json:"errorCode"`
	HTTPCode int    `json:"-"`
}

func (e *httpError) Error() string {
	return fmt.Sprintf("[%s] %s", e.ErrCode, e.ErrMsg)
}

// Returns an HTTP error representing when a selected cluster is not connected.
// This means there is no established tunnel between Voltron and Guardian
// (within the App cluster).
func clusterNotConnectedError(clusterID string) *httpError {
	msg := fmt.Sprintf("Selected cluster %s unreachable, no connection", clusterID)
	return &httpError{
		Err:      errors.New(msg),
		ErrMsg:   msg,
		ErrCode:  "error-cluster-not-connected",
		HTTPCode: 400,
	}
}

// Returns an HTTP error representing when a selected cluster is not found. This
// means Voltron is not aware of the cluster with the given ID. This happens when
// Voltron has not picked up on an added ManagedCluster yet.
func clusterNotFoundError(clusterID string) *httpError {
	msg := fmt.Sprintf("Cluster with ID %s not found", clusterID)
	return &httpError{
		Err:      errors.New(msg),
		ErrMsg:   msg,
		ErrCode:  "error-cluster-not-found",
		HTTPCode: 400,
	}
}

// Returns an error for an unexpected cluster ID.
func unexpectedClusterIDError(clusterID string) *httpError {
	msg := fmt.Sprintf("Unexpected cluster ID: %s", clusterID)
	return &httpError{
		Err:      errors.New(msg),
		ErrMsg:   msg,
		ErrCode:  "error-unexpected-cluster",
		HTTPCode: 403,
	}
}

// Returns an error for an unexpected tenant ID.
func unexpectedTenantIDError(tenantID string) *httpError {
	msg := fmt.Sprintf("Unexpected tenant ID: %s", tenantID)
	return &httpError{
		Err:      errors.New(msg),
		ErrMsg:   msg,
		ErrCode:  "error-unexpected-tenant",
		HTTPCode: 403,
	}
}

// writeHTTPError replies to the request with the specified HTTP error and its corresponding
// HTTP code. It does not otherwise end the request; the caller should ensure no further
// writes are done to w. The HTTP error will be encoded as JSON in the response body.
func writeHTTPError(w http.ResponseWriter, e *httpError) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(e.HTTPCode)
	// Attempt to write the encoded error type. If it fails write internal server
	// error instead.
	if err := json.NewEncoder(w).Encode(e); err != nil {
		log.Errorf("Unable to encode the given error type %#v (%s)", e, err.Error())
		http.Error(w, "Unable to process error type", http.StatusInternalServerError)
		return
	}
}
