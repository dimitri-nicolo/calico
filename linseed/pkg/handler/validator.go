// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/davecgh/go-spew/spew"

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
	v1.L3FlowParams | v1.FlowLogParams |
		v1.L7FlowParams | v1.L7LogParams |
		v1.DNSFlowParams | v1.DNSLogParams |
		v1.EventParams | v1.AuditLogParams | v1.BGPLogParams
}

// BulkRequestParams is the collection of request parameters types
// for bulk requests that will be decoded and validated from an HTTP request
type BulkRequestParams interface {
	v1.FlowLog | v1.Event | v1.L7Log | v1.DNSLog | v1.AuditLog | v1.BGPLog
}

// DecodeAndValidateBulkParams will decode and validate input parameters
// passed on the HTTP body of a bulk request. In case the input parameters
// are invalid or cannot be decoded, an HTTPStatusError will be returned
func DecodeAndValidateBulkParams[T BulkRequestParams](w http.ResponseWriter, req *http.Request) ([]T, error) {
	var bulkParams []T

	// Check content-type
	content := strings.ToLower(strings.TrimSpace(req.Header.Get(contentType)))
	if content != newlineJsonContent {
		return bulkParams, &v1.HTTPError{
			Status: http.StatusUnsupportedMediaType,
			Msg:    fmt.Sprintf("Received a request with content-type (%s) that is not supported", content),
		}
	}

	// Check body
	if req.Body == nil {
		return bulkParams, &v1.HTTPError{
			Status: http.StatusBadRequest,
			Msg:    "Received a request with an empty body",
		}
	}

	// Read only max bytes
	req.Body = http.MaxBytesReader(w, req.Body, maxBytes)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return bulkParams, &v1.HTTPError{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		}
	}

	trimBody := bytes.Trim(body, "\n")
	d := json.NewDecoder(bytes.NewReader(trimBody))
	d.DisallowUnknownFields()
	for {
		var input T
		err := d.Decode(&input)
		if err != nil {
			if err != io.EOF {
				logrus.WithError(err).Errorf("Failed to decode message for %s", trimBody)
				return bulkParams, &v1.HTTPError{
					Status: http.StatusBadRequest,
					Msg:    "Request body contains badly-formed JSON",
				}
			}
			break
		}
		bulkParams = append(bulkParams, input)
	}

	if len(bulkParams) == 0 {
		return bulkParams, &v1.HTTPError{
			Status: http.StatusBadRequest,
			Msg:    "Request body contains badly-formed JSON",
		}
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
		return reqParams, &v1.HTTPError{
			Status: http.StatusUnsupportedMediaType,
			Msg:    fmt.Sprintf("Received a request with content-type (%s) that is not supported", content),
		}
	}

	// Decode the http request body into the struct.
	if err := httputils.Decode(w, req, &reqParams); err != nil {
		return reqParams, err
	}

	// Validate parameters.
	if err := validator.Validate(reqParams); err != nil {
		return reqParams, &v1.HTTPError{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		}
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		// If debug logging is enabled, print out pretty params.
		paramsStr := spew.Sdump(reqParams)
		logrus.Debugf("Decoded %T: %s", reqParams, paramsStr)
	}

	return reqParams, nil
}
