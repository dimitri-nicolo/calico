// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package handler

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

// maxBytes represents the maximum bytes
// an HTTP request body can have
const maxBytes = 2000000

// newlineJsonContent is the supported content type
// for bulk APIs
const newlineJsonContent = "application/x-ndjson"

// jsonContent is the supported content type
// for bulk APIs
const jsonContent = "application/json"

// contentType is the content type header
const contentType = "Content-Type"

// RequestParams is the collection of request parameters types
// that will be decoded and validated from an HTTP request
type RequestParams interface {
	v1.L3FlowParams | v1.L7FlowParams
}

// BulkRequestParams is the collection of request parameters types
// for bulk requests that will be decoded and validated from an HTTP request
type BulkRequestParams interface {
	v1.FlowLog
}

// DecodeAndValidateBulkParams will decode and validate input parameters
// passed on the HTTP body of a bulk request. In case the input parameters
// are invalid or cannot be decoded, an HTTPStatusError will be returned
func DecodeAndValidateBulkParams[T BulkRequestParams](w http.ResponseWriter, req *http.Request) ([]T, error) {
	var bulkParams []T

	// Check content-type
	content := strings.ToLower(strings.TrimSpace(req.Header.Get(contentType)))
	if content != newlineJsonContent {
		return bulkParams, &httputils.HttpStatusError{
			Status: http.StatusUnsupportedMediaType,
			Msg:    "Received a request with content-type that is not supported",
			Err:    errors.New("content-type not supported"),
		}
	}

	// Check body
	if req.Body == nil {
		return bulkParams, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    "Received a request with an empty body",
			Err:    errors.New("empty request body"),
		}
	}

	// Read only max bytes
	req.Body = http.MaxBytesReader(w, req.Body, maxBytes)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return bulkParams, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	// bulk requests will have json delimitated by a newline
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		if string(line) == "{}" {
			return bulkParams, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    "Request body contains an empty JSON",
				Err:    err,
			}
		}
		// decode each newline to its correspondent structure
		var input T
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&input); err != nil {
			return bulkParams, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    "Request body contains badly-formed JSON",
				Err:    err,
			}
		}
		bulkParams = append(bulkParams, input)
	}

	return bulkParams, nil
}

// DecodeAndValidateReqParams will decode and validate input parameters
// passed on the HTTP body of a request. In case the input parameters
// are invalid or cannot be decoded, an HTTPStatusError will be returned
func DecodeAndValidateReqParams[T RequestParams](w http.ResponseWriter, req *http.Request) (*T, error) {
	reqParams := new(T)

	content := strings.ToLower(strings.TrimSpace(req.Header.Get(contentType)))
	if content != jsonContent {
		return reqParams, &httputils.HttpStatusError{
			Status: http.StatusUnsupportedMediaType,
			Msg:    "Received a request with content-type that is not supported",
			Err:    errors.New("content-type not supported"),
		}
	}

	// Decode the http request body into the struct.
	if err := httputils.Decode(w, req, &reqParams); err != nil {
		return reqParams, err
	}

	// Validate parameters.
	if err := validator.Validate(reqParams); err != nil {
		return reqParams, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
			Err:    err,
		}
	}

	return reqParams, nil
}
