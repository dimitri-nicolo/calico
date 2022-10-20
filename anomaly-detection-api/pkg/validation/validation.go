// Copyright (c) 2022 Tigera All rights reserved.
package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/data"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/httputils"
)

const (
	MaxModelStringSizeAllowed = 100000000 // 100MB
)

var (
	stateChangingMethods = map[string]bool{
		http.MethodPost: true,
		http.MethodPut:  true,
	}
)

// ValidateClustersEndpointModelStorageHandlerRequest validates the request to the /clusters/.../{detector_category}/{detector_class}.model endpoint handled
// by handler.ModelStorageHandler for model storage purposes
// sub validation includes:
//   - valid content type text/plain
//   - models request body less than MaxModelStringSizeAllowed
func ValidateClustersEndpointModelStorageHandlerRequest(req *http.Request) *api_error.APIError {
	err := ValidateContentType(*req, httputils.StringMIME)
	if err != nil {
		log.WithError(err).Infof("invalid content-type on request")
		return err
	}

	err = ValidateModelSizeRequestBody(req)
	if err != nil {
		log.WithError(err).Infof("invalid path error on request")
		return err
	}

	return nil
}

// ValidateClustersLogTypeMetadataRequest validates the request for the /clusters/.../{log_type} endpoint handled
// by handler.ClustersEndpointHandler
// sub validation includes:
//	- valid content type is application/JSON if PUT method
//	- valid request LogTypeMetadata body
//  	- last_updated is a float representing unix timestamp
func ValidateClustersLogTypeMetadataRequest(req *http.Request) *api_error.APIError {
	if !stateChangingMethods[req.Method] {
		return nil
	}

	err := ValidateContentType(*req, httputils.JSSONMIME)
	if err != nil {
		log.WithError(err).Infof("invalid content-type on request")
		return err
	}

	err = ValidateClustersLogTypeMetadataRequestBody(req)
	if err != nil {
		log.WithError(err).Infof("invalid LogTypeMetadata request body")
		return err
	}

	return nil
}

// ValidateClustersLogTypeMetadataRequestBody validates the fields in the  request's body as a LogTypeMetadata struct.
// sub validations includes:
//	- last_updated is a float representing unix timestamp
func ValidateClustersLogTypeMetadataRequestBody(req *http.Request) *api_error.APIError {
	var logTypeMetadataReqBody data.LogTypeMetadata

	b, err := ioutil.ReadAll(req.Body)

	if err != nil {
		return &api_error.APIError{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("invalid request body received: %s", err.Error()),
		}
	}

	// use unmarshal to decode to respect json annotations
	err = json.Unmarshal(b, &logTypeMetadataReqBody)
	if err != nil {
		return &api_error.APIError{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("invalid request body received: %s", err.Error()),
		}
	}

	if _, err := strconv.ParseFloat(logTypeMetadataReqBody.LastUpdated, 32); err != nil {
		return &api_error.APIError{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("invalid LogTypeMetadata.last_update field received: %s", err.Error()),
		}
	}

	// to fully convert to a unix time stamp:
	// sec, dec := math.Modf(timeFloat)
	// time.Unix(int64(sec), int64(dec*(1e9)))
	// for validation purposes it's enough to ensure float format as conversion to unix time stamp is just
	// a calculation between the decimal and integer portion of a valid float

	// restore body as it will be read again in further handlers
	req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return nil
}

// ValidateContentType validates the request has a Content-Type header
// of text/plain if method is POST
func ValidateContentType(req http.Request, contentTypeValue string) *api_error.APIError {
	contentType := req.Header.Get(httputils.ContentTypeHeaderKey)
	if len(contentType) == 0 && !stateChangingMethods[req.Method] {
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

		if t == contentTypeValue {
			return nil
		}
	}

	return &api_error.APIError{
		StatusCode: http.StatusUnsupportedMediaType,
		Err:        fmt.Errorf(http.StatusText(http.StatusUnsupportedMediaType)),
	}
}

// ValidateModelSizeRequestBody validates model in request body is <= MaxModelStringSizes on
// the /clusters{cluster_name}/models/ endpoint
func ValidateModelSizeRequestBody(req *http.Request) *api_error.APIError {
	if stateChangingMethods[req.Method] {

		log.Debugf("attempt at uploading %s model of size %d", req.URL.Path, req.ContentLength)

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
