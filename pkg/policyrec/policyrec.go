// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package policyrec

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	k8s "k8s.io/client-go/kubernetes"

	lpolicyrec "github.com/tigera/lma/pkg/policyrec"
)

const (
	defaultTierName = "default"
)

// The response that we return
type PolicyRecommendationResponse struct {
	*lpolicyrec.Recommendation
	ErrorMessage string `json:"errorMessage"`
}

// RecommendationEngine is the entry point into the Policy Recommendation engine.
type RecommendationEngine interface {
	GetRecommendation(context.Context, *lpolicyrec.PolicyRecommendationParams) (*PolicyRecommendationResponse, error)
}

// policyRecommendationEngine implements the RecommendationEngine interface.
type policyRecommendationEngine struct {
	k8sClient k8s.Interface
	esClient  lpolicyrec.CompositeAggregator
}

// NewPolicyRecommendationEngine returns an initialized policyRecommendationEngine
func NewPolicyRecommendationEngine(kc k8s.Interface, ec lpolicyrec.CompositeAggregator) RecommendationEngine {
	return &policyRecommendationEngine{
		k8sClient: kc,
		esClient:  ec,
	}
}

// GetRecommendation stitches together interesting flows from Elasticsearch and feeds them
// through the endpoint recommendation engine to obtain recommended policies.
func (pre *policyRecommendationEngine) GetRecommendation(ctx context.Context, params *lpolicyrec.PolicyRecommendationParams) (*PolicyRecommendationResponse, error) {
	// Query elasticsearch with the parameters provided
	flows, err := lpolicyrec.QueryElasticsearchFlows(ctx, pre.esClient, params)
	if err != nil {
		log.WithError(err).Info("Error querying elasticsearch")
		return nil, err
	}

	if len(flows) == 0 {
		log.WithField("params", params).Info("No matching flows found")
		errorMessage := fmt.Sprintf("No matching flows found for endpoint name '%v' in namespace '%v' within the time range '%v:%v'",
			params.EndpointName, params.Namespace, params.StartTime, params.EndTime)

		policyRecResponse := &PolicyRecommendationResponse{
			ErrorMessage: errorMessage,
		}
		return policyRecResponse, nil
	}

	policyName := lpolicyrec.GeneratePolicyName(pre.k8sClient, params)
	// Setup the recommendation engine. We specify the default tier as the flows that we are fetching
	// is at the end of the default tier. Similarly we set the recommended policy order to nil as well.
	// TODO(doublek): Tier and policy order should be obtained from the observation point policy.
	recEngine := lpolicyrec.NewEndpointRecommendationEngine(params.EndpointName, params.Namespace, policyName, defaultTierName, nil)
	for _, flow := range flows {
		log.WithField("flow", flow).Debug("Calling recommendation engine with flow")
		err := recEngine.ProcessFlow(*flow)
		if err != nil {
			log.WithError(err).WithField("flow", flow).Debug("Error processing flow")
		}
	}

	recommendation, err := recEngine.Recommend()
	if err != nil {
		log.WithError(err).Error("Error when recommending policy")
		return &PolicyRecommendationResponse{
			ErrorMessage: err.Error(),
		}, nil
	}
	log.WithField("recommendation", recommendation).Debug("Policy Recommendation")

	return &PolicyRecommendationResponse{
		Recommendation: recommendation,
	}, nil
}
