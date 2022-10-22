// Copyright (c) 2019, 2022 Tigera, Inc. All rights reserved.
package policyrec

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/lma/pkg/api"
)

const (
	esFlowIndex = "tigera_secure_ee_flows*"
)

func ExtractPolicyRecommendationParamsFromRequest(req *http.Request) (*PolicyRecommendationParams, error) {
	// Read the body data
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Info("Error reading request body")
		return nil, err
	}

	// Unmarshal the body of data to PolicyRecommendationParams object
	var reqParams PolicyRecommendationParams
	err = json.Unmarshal(b, &reqParams)
	if err != nil {
		log.WithError(err).Info("Error unmarshaling request parameters")
		return nil, err
	}

	// Set the searched index to the flow log index
	reqParams.DocumentIndex = esFlowIndex

	// Validate the params are valid
	err = ValidatePolicyRecommendationParams(&reqParams)
	if err != nil {
		log.WithError(err).Info("Error validating the request parameters")
		return nil, err
	}

	return &reqParams, nil
}

// ValidatePolicyRecommendationParams returns an error when the policy recommendation query
// parameters are not set correctly.
// If Endpoint is not empty and Namespace is empty, then the query is global.
func ValidatePolicyRecommendationParams(params *PolicyRecommendationParams) error {
	// StartTime, EndTime, are not allowed to be empty. Endpoint and Namespace cannot both be empty.
	if params.StartTime == "" {
		return fmt.Errorf("invalid start_time specified")
	}
	if params.EndTime == "" {
		return fmt.Errorf("invalid end_time specified")
	}
	if params.Namespace == "" && params.EndpointName == "" {
		return fmt.Errorf("namespace and endpoint_name cannot both be empty")
	}

	return nil
}

func QueryElasticsearchFlows(ctx context.Context, ca CompositeAggregator, params *PolicyRecommendationParams) ([]*api.Flow, error) {
	query := BuildElasticQuery(params)
	return SearchFlows(ctx, ca, query, params)
}
