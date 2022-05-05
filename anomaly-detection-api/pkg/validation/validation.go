// Copyright (c) 2022 Tigera All rights reserved.
package validation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
)

const (
	ContentTypeHeaderKey   = "Content-Type"
	ContentLengthHeaderKey = "Content-Length"
	StringMIME             = "text/plain"

	MaxModelStringSizeAllowed = 15730000 // 15MB
)

var (
	validFilePathRegex = regexp.MustCompile(`^\/(clusters){1}\/(.+)\/(models){1}\/([^\/]+)$`)
)

// ValidateClustersEndpointRequest validates the request for the /clusters endpoint handled
// by handler.ClustersEndpointHandler
// sub validation includes:
//   - valid path
//   - valid content type
//   - models request body less than MaxModelStringSizeAllowed
func ValidateClustersEndpointRequest(req *http.Request) *api_error.APIError {
	err := ValidateTextContentType(*req)
	if err != nil {
		log.WithError(err).Infof("invalid content-type on request")
		return err
	}

	err = ValidateClustersModelsEndpointPath(req.URL.Path)
	if err != nil {
		log.WithError(err).Infof("invalid path error on request")
		return err
	}

	err = ValidateModelSizeRequestBody(req)
	if err != nil {
		log.WithError(err).Infof("invalid path error on request")
		return err
	}

	return nil
}

// ValidateTextContentType validates the request has a Content-Type header
// of text/plain if method is POST
func ValidateTextContentType(req http.Request) *api_error.APIError {
	contentType := req.Header.Get(ContentTypeHeaderKey)
	if len(contentType) == 0 && req.Method == http.MethodGet {
		return nil
	}

	for _, v := range strings.Split(contentType, ";") {
		t, _, err := mime.ParseMediaType(v)
		if err != nil {
			return &api_error.APIError{
				StatusCode: http.StatusBadRequest,
				Err:        err,
			}
		}

		if t == StringMIME {
			return nil
		}
	}

	return &api_error.APIError{
		StatusCode: http.StatusUnsupportedMediaType,
		Err:        fmt.Errorf(http.StatusText(http.StatusUnsupportedMediaType)),
	}
}

// ValidateClustersModelsEndpointPath validates the path to the Clusters endpoint is correct
func ValidateClustersModelsEndpointPath(path string) *api_error.APIError {
	match := validFilePathRegex.MatchString(path)
	if !match {
		err := fmt.Errorf("invalid path")
		return &api_error.APIError{
			StatusCode: http.StatusBadRequest,
			Err:        err,
		}
	}

	return nil
}

// ValidateModelSizeRequestBody validates model in request body is <= MaxModelStringSizes on
// the /clusters{cluster_name}/models/ endpoint
func ValidateModelSizeRequestBody(req *http.Request) *api_error.APIError {
	// only check on the models endpoint
	if match := validFilePathRegex.MatchString(req.URL.Path); match && req.Method == http.MethodPost {

		if req.ContentLength > MaxModelStringSizeAllowed {
			return &api_error.APIError{
				StatusCode: http.StatusRequestEntityTooLarge,
				Err:        fmt.Errorf(http.StatusText(http.StatusRequestEntityTooLarge)),
			}
		}

		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = ioutil.ReadAll(req.Body)
		}

		// Restore the io.ReadCloser to its original state
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		if int64(len(bodyBytes)) > req.ContentLength {
			return &api_error.APIError{
				StatusCode: http.StatusBadRequest,
				Err:        fmt.Errorf(http.StatusText(http.StatusBadRequest)),
			}
		}
	}

	return nil
}
