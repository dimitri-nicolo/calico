// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/lma/pkg/api"
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

func ValidatePolicyRecommendationParams(params *PolicyRecommendationParams) error {
	// StartTime, EndTime, and EndpointName are not allowed to be nil
	// If Namespace is empty, then the query is global
	if params.StartTime == "" {
		return fmt.Errorf("Invalid start_time specified")
	}
	if params.EndTime == "" {
		return fmt.Errorf("Invalid end_time specified")
	}
	if params.EndpointName == "" {
		return fmt.Errorf("endpoint_name cannot be empty")
	}
	if params.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	return nil
}

func QueryElasticsearchFlows(ctx context.Context, ca CompositeAggregator, params *PolicyRecommendationParams) ([]*api.Flow, error) {
	query := BuildElasticQuery(params)
	return SearchFlows(ctx, ca, query, params)
}
