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

	// Loads handles retrival of the file specified in the request and
	// return the file content as a bas64 string
	Load(r *http.Request) (int64, string, *api_error.APIError)

	// Stat handles the retrieval of file information of the specified model
	// in the request. Currently returns file size in bytes as int64
	Stat(r *http.Request) (int64, *api_error.APIError)
}
