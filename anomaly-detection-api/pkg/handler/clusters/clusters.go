// Copyright (c) 2022 Tigera All rights reserved.
package clusters

import (
	"fmt"
	"net/http"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/config"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/clusters/log_type"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/clusters/model_storage"
)

// ClustersEndpointHandler Service for handling the /clusters endpoint
type ClustersEndpointHandler struct {
	modelStorageHandler    *model_storage.ModelStorageHandler
	logTypeMetadataHandler *log_type.LogTypeEndpointHandler
}

func NewClustersEndpointHandler(config *config.Config) *ClustersEndpointHandler {
	return &ClustersEndpointHandler{
		modelStorageHandler:    model_storage.NewModelStorageHandler(config),
		logTypeMetadataHandler: log_type.NewLogTypeHandler(),
	}
}

// RouteClustersEndpoint routes the request to the other sub path handlers.  It defaults to a 404 error
// if the request's URL does not match any registered path
func (c *ClustersEndpointHandler) RouteClustersEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		path := req.URL.Path

		switch {
		// /clusters/{cluster_name}/{log_type}/metadata
		case LogTypeMetadataEndpointRegex.MatchString(path):
			c.logTypeMetadataHandler.HandleLogTypeMetaData(w, req)

		// /clusters/{cluster_name}/models/(dynamic|static)/{detector_category}/{detector_class}.models
		case ModelStorageEndpointRegex.MatchString(path):
			c.modelStorageHandler.HandleModelStorage(w, req)
		default: // default to 404
			err := fmt.Errorf("request has not matched any registered path")
			apiErr := &api_error.APIError{
				StatusCode: http.StatusNotFound,
				Err:        err,
			}
			api_error.WriteAPIErrorToHeader(w, apiErr)
		}
	}
}
