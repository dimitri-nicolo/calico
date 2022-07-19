package auth

import (
	"fmt"
	"net/http"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api_error"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/clusters"

	authzv1 "k8s.io/api/authorization/v1"
)

var (
	// RBAC for /clusters/{cluster_name}/models/(dynamic|static)/{detector_category}/{detector_class}.models
	modelsMethodRBACAttributes = map[string][]*authzv1.ResourceAttributes{
		http.MethodHead: {
			{

				Group:    adDetectorsResourceGroup,
				Resource: adModelsResourceName,
				Verb:     "get",
			},
		},
		http.MethodGet: {
			{

				Group:    adDetectorsResourceGroup,
				Resource: adModelsResourceName,
				Verb:     "get",
			},
		},
		http.MethodPost: {
			{

				Group:    adDetectorsResourceGroup,
				Resource: adModelsResourceName,
				Verb:     "create",
			},
			{

				Group:    adDetectorsResourceGroup,
				Resource: adModelsResourceName,
				Verb:     "update",
			},
		},
	}

	// RBAC for /clusters/{cluster_name}/{log_type}/metadata endpoint
	logTypeMetadataRBACAttributes = map[string][]*authzv1.ResourceAttributes{
		http.MethodHead: {
			{
				Group:    adDetectorsResourceGroup,
				Resource: adMetadataResourceName,
				Verb:     "get",
			},
		},
		http.MethodGet: {
			{
				Group:    adDetectorsResourceGroup,
				Resource: adMetadataResourceName,
				Verb:     "get",
			},
		},
		http.MethodPut: {
			{
				Group:    adDetectorsResourceGroup,
				Resource: adMetadataResourceName,
				Verb:     "create",
			},
			{
				Group:    adDetectorsResourceGroup,
				Resource: adMetadataResourceName,
				Verb:     "update",
			},
		},
	}
)

// GetRBACResoureAttribute returns the RBAC Attributes for the Anomaly Detection API based on
// the exposed endpoints of the AD API
func GetRBACResoureAttribute(namespace string, req *http.Request) ([]*authzv1.ResourceAttributes, *api_error.APIError) {
	path := req.URL.Path

	var rbacAttributes []*authzv1.ResourceAttributes

	switch {
	// /clusters/{cluster_name}/models/(dynamic|static)/{detector_category}/{detector_class}.models
	case clusters.ModelStorageEndpointRegex.MatchString(path):
		rbacAttributes = modelsMethodRBACAttributes[req.Method]

	// /clusters/{cluster_name}/{log_type}/metadata
	case clusters.LogTypeMetadataEndpointRegex.MatchString(path):
		rbacAttributes = logTypeMetadataRBACAttributes[req.Method]

	default:
		return nil, &api_error.APIError{
			StatusCode: http.StatusNotFound,
			Err:        fmt.Errorf("authentication refused for request to unknown path: %s", path),
		}
	}

	for _, rbacAtrribute := range rbacAttributes {
		rbacAtrribute.Namespace = namespace
	}

	return rbacAttributes, nil
}
