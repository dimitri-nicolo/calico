package storage

import (
	"net/http"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
)

// ModelStorageHandler serves storage and retrieval for models content
// in the associated storage method
type ModelStorageHandler interface {
	// Save handles the content in the request to save it
	// with the associated storage method
	Save(r *http.Request) error

	// Loads handles retrival of the file specified in the request
	Load(r *http.Request) (string, *api_error.APIError)
}
