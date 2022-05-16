// Copyright (c) 2022 Tigera All rights reserved.
package clusters

import (
	"io"
	"net/http"
	"strconv"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/config"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/storage"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/validation"
)

const (
	ModelFileDataKey       = "model"
	AcceptHeaderKey        = "Accept"
	ContentLengthHeaderKey = "Content-Length"

	StringMIME = "text/plain"
)

// ClustersEndpointHandler Service for handling the /clusters endpoint
type ClustersEndpointHandler struct {
	storageHandler storage.ModelStorageHandler
}

func NewClustersEndpointHandler(config *config.Config) *ClustersEndpointHandler {
	return &ClustersEndpointHandler{
		storageHandler: &storage.FileModelStorageHandler{
			FileStoragePath: config.StoragePath,
		},
	}
}

// HandleClusters serves the /clusters endpoint with validation. It supports
// GET /models for model file retrieval
// POST /models for model file storage
// throws a 405 - NotSupported error if received any other method
func (c *ClustersEndpointHandler) HandleClusters() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		err := validation.ValidateClustersEndpointRequest(req)
		if err != nil {
			api_error.WriteAPIErrorToHeader(w, err)
			return
		}

		switch req.Method {
		case http.MethodHead:
			size, apiErr := c.storageHandler.Stat(req)
			if apiErr != nil {
				api_error.WriteAPIErrorToHeader(w, apiErr)
				return
			}

			w.Header().Set(AcceptHeaderKey, StringMIME)
			// 10 - for decimal value
			w.Header().Set(ContentLengthHeaderKey, strconv.FormatInt(size, 10))
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			size, fileBytes, apiErr := c.storageHandler.Load(req)
			if apiErr != nil {
				api_error.WriteAPIErrorToHeader(w, apiErr)
				return
			}

			_, err := io.WriteString(w, fileBytes)
			if err != nil {
				apiErr := &api_error.APIError{
					StatusCode: http.StatusInternalServerError,
					Err:        err,
				}
				api_error.WriteAPIErrorToHeader(w, apiErr)
				return
			}

			w.Header().Set(AcceptHeaderKey, StringMIME)
			// 10 - for decimal value
			w.Header().Set(ContentLengthHeaderKey, strconv.FormatInt(size, 10))
			w.WriteHeader(http.StatusOK)
		case http.MethodPost:
			err := c.storageHandler.Save(req)
			if err != nil {
				apiErr := &api_error.APIError{
					StatusCode: http.StatusInternalServerError,
					Err:        err,
				}
				api_error.WriteAPIErrorToHeader(w, apiErr)
			}
		default:
			apiErr := &api_error.APIError{
				StatusCode: http.StatusMethodNotAllowed,
				Err:        err,
			}
			api_error.WriteAPIErrorToHeader(w, apiErr)
		}
	}
}
