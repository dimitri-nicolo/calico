package log_type

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/data"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/httputils"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/storage"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/validation"
)

const (
	MetadataEndpointPath = "/metadata"
	requestFloatSize     = 64
)

// LogTypeEndpointHandler Service for handling the /clusters/../{log_type} endpoint
type LogTypeEndpointHandler struct {
	metadataCache storage.ObjectCache
}

func NewLogTypeHandler() *LogTypeEndpointHandler {
	return &LogTypeEndpointHandler{
		metadataCache: storage.NewSynchronizedObjectCache(),
	}
}

// HandleLogTypeMetaData serves the /clusters/.../{log_type}/metadata endpoint with validation. It supports
// GET /log_type/metadata for retrieval of information relating to Anomaly Detection processing for the given log_type
// PUT /log_type/metadata for update  information relating to Anomaly Detection processing for the given log_type
// throws a 405 - NotSupported error if received any other method
func (l *LogTypeEndpointHandler) HandleLogTypeMetaData(w http.ResponseWriter, req *http.Request) {
	err := validation.ValidateClustersLogTypeMetadataRequest(req)
	if err != nil {
		api_error.WriteAPIErrorToHeader(w, err)
		return
	}

	switch req.Method {
	case http.MethodGet:
		clustersLogTypePath := strings.TrimSuffix(req.URL.Path, MetadataEndpointPath)
		logTypeMetaData := l.metadataCache.Get(clustersLogTypePath)
		if logTypeMetaData == nil {
			apiErr := &api_error.APIError{
				StatusCode: http.StatusNotFound,
				Err:        err,
			}
			api_error.WriteAPIErrorToHeader(w, apiErr)
			return
		}

		writeLogTypeMetaDataToResponse(w, logTypeMetaData.(data.LogTypeMetadata))

	case http.MethodPut:
		clustersLogTypePath := strings.TrimSuffix(req.URL.Path, MetadataEndpointPath)

		b, err := ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		if err != nil {
			apiErr := &api_error.APIError{
				StatusCode: http.StatusBadRequest,
				Err:        fmt.Errorf("unexpected request body received: %s", err.Error()),
			}
			api_error.WriteAPIErrorToHeader(w, apiErr)
			return
		}

		// use unmarshal to decode to respect json annotations
		var logTypeMetadataReqBody data.LogTypeMetadata
		err = json.Unmarshal(b, &logTypeMetadataReqBody)
		if err != nil {
			apiErr := &api_error.APIError{
				StatusCode: http.StatusBadRequest,
				Err:        fmt.Errorf("invalid request body received: %s", err.Error()),
			}
			api_error.WriteAPIErrorToHeader(w, apiErr)
			return
		}

		storedData := l.metadataCache.Get(clustersLogTypePath)
		// if data previously exists for the cluster's logtype path check if the request
		// containes a last_updated timestamp occuring before the stored timestamp
		if storedData != nil {
			storedLogTypeMetadata, ok := storedData.(data.LogTypeMetadata)
			if !ok {
				apiErr := &api_error.APIError{
					StatusCode: http.StatusInternalServerError,
					Err:        fmt.Errorf("unexpected failure converting stored data to LogTypeMetadata"),
				}
				api_error.WriteAPIErrorToHeader(w, apiErr)
				return
			}

			requestLastUpdated, err := strconv.ParseFloat(logTypeMetadataReqBody.LastUpdated, requestFloatSize)
			if err != nil {
				apiErr := &api_error.APIError{
					StatusCode: http.StatusBadRequest,
					Err:        fmt.Errorf("invalid LogTypeMetadata.LastUpdated from request: %s", err.Error()),
				}
				api_error.WriteAPIErrorToHeader(w, apiErr)
				return
			}

			storedLastUpdated, err := strconv.ParseFloat(storedLogTypeMetadata.LastUpdated, requestFloatSize)
			if err != nil {
				apiErr := &api_error.APIError{
					StatusCode: http.StatusInternalServerError,
					Err:        fmt.Errorf("unexpected LogTypeMetadata.LastUpdated stored: %s", err.Error()),
				}
				api_error.WriteAPIErrorToHeader(w, apiErr)
				return
			}

			if storedLastUpdated > requestLastUpdated {
				apiErr := &api_error.APIError{
					StatusCode: http.StatusBadRequest,
					Err:        fmt.Errorf("received request to update a timestamp to a time previous than the stored timestamp"),
				}
				api_error.WriteAPIErrorToHeader(w, apiErr)
				return
			}
		}

		updatedlogTypeMetaData := l.metadataCache.Set(clustersLogTypePath, logTypeMetadataReqBody)

		if updatedlogTypeMetaData == nil {
			apiErr := &api_error.APIError{
				StatusCode: http.StatusInternalServerError,
				Err:        err,
			}
			api_error.WriteAPIErrorToHeader(w, apiErr)
			return
		}
		writeLogTypeMetaDataToResponse(w, updatedlogTypeMetaData.(data.LogTypeMetadata))

	default:
		apiErr := &api_error.APIError{
			StatusCode: http.StatusMethodNotAllowed,
			Err:        err,
		}
		api_error.WriteAPIErrorToHeader(w, apiErr)
	}
}

func writeLogTypeMetaDataToResponse(w http.ResponseWriter, logTypeMetaData data.LogTypeMetadata) {
	err := json.NewEncoder(w).Encode(logTypeMetaData)
	if err != nil {
		apiErr := &api_error.APIError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
		api_error.WriteAPIErrorToHeader(w, apiErr)
		return
	}

	w.Header().Set(httputils.AcceptHeaderKey, httputils.JSSONMIME)
	w.WriteHeader(http.StatusOK)
}
