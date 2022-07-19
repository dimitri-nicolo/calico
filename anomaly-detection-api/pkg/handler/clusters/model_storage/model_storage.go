// Copyright (c) 2022 Tigera All rights reserved.
package model_storage

import (
	"io"
	"net/http"
	"strconv"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/config"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/httputils"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/storage"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/validation"
)

type ModelStorageHandler struct {
	fileModelStorageHandler storage.ModelStorage
}

func NewModelStorageHandler(config *config.Config) *ModelStorageHandler {
	return &ModelStorageHandler{
		fileModelStorageHandler: &storage.FileModelStorageHandler{
			FileStoragePath: config.StoragePath,
		},
	}
}

// HandleModelStorage serves the /clusters/.../models endpoint with validation. It supports
// GET /models for model file retrieval
// POST /models for model file storage
// throws a 405 - NotSupported error if received any other method
func (m *ModelStorageHandler) HandleModelStorage(w http.ResponseWriter, req *http.Request) {
	err := validation.ValidateClustersEndpointModelStorageHandlerRequest(req)
	if err != nil {
		api_error.WriteAPIErrorToHeader(w, err)
		return
	}

	switch req.Method {
	case http.MethodHead:
		size, apiErr := m.fileModelStorageHandler.Stat(req)
		if apiErr != nil {
			api_error.WriteAPIErrorToHeader(w, apiErr)
			return
		}

		w.Header().Set(httputils.AcceptHeaderKey, httputils.StringMIME)
		// 10 - for decimal value
		w.Header().Set(httputils.ContentLengthHeaderKey, strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		size, fileBytes, apiErr := m.fileModelStorageHandler.Load(req)
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

		w.Header().Set(httputils.AcceptHeaderKey, httputils.StringMIME)
		// 10 - for decimal value
		w.Header().Set(httputils.ContentLengthHeaderKey, strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusOK)
	case http.MethodPost:
		err := m.fileModelStorageHandler.Save(req)
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
