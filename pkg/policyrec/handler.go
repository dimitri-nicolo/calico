// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	authzv1 "k8s.io/api/authorization/v1"

	"github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/util"
)

const (
	lmaGroup    = "lma.tigera.io"
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

func ValidatePermissions(req *http.Request, k8sAuth auth.K8sAuthInterface) (int, error) {
	// Check permissions against our custom resource for verifying the appropriate permissions
	resAtr := &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    lmaGroup,
		Resource: "index",
		Name:     "flows",
	}

	if stat, err := checkAuthorized(req, *resAtr, k8sAuth); err != nil {
		return stat, fmt.Errorf("Not authorized to get flow logs")
	}

	// Check permissions against our custom resource for verifying the appropriate permissions
	resAtr = &authzv1.ResourceAttributes{
		Verb:     "create",
		Group:    lmaGroup,
		Resource: "policyrecommendation",
	}

	if stat, err := checkAuthorized(req, *resAtr, k8sAuth); err != nil {
		return stat, fmt.Errorf("Not authorized to create policies through policyrecommendation")
	}

	// Authorized for all actions on all resources required
	return 0, nil
}

func checkAuthorized(req *http.Request, atr authzv1.ResourceAttributes, k8sAuth auth.K8sAuthInterface) (int, error) {
	ctx := util.NewContextWithReviewResource(req.Context(), &atr)
	reqWithCtx := req.WithContext(ctx)

	return k8sAuth.Authorize(reqWithCtx)
}

func QueryElasticsearchFlows(ctx context.Context, ca CompositeAggregator, params *PolicyRecommendationParams) ([]*api.Flow, error) {
	query := BuildElasticQuery(params)
	return SearchFlows(ctx, ca, query, params)
}
