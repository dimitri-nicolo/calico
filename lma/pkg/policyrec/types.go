// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/lma/pkg/api"
)

type PolicyRecommendationParams struct {
	// StartTime represents the beginning of the time window to consider flow logs for.
	StartTime string `json:"start_time"`
	// EndTime represents the end of the time window to consider flow logs for.
	EndTime string `json:"end_time"`
	// EndpointName should correspond to the aggregated name in flow logs.
	EndpointName string `json:"endpoint_name"`
	// Namespace should correspond to the endpoint referenced by EndpointName.
	Namespace string `json:"namespace"`
	// Unprotected specifies whether results should be restricted to Calico Profiles
	Unprotected bool `json:"unprotected"`

	// Helper values
	// DocumentIndex represents the elasticsearch index to search through.
	// In this case, it will be the flow log index.
	DocumentIndex string `json:"doc_index"`
}

// Recommendation is the set of policies recommended by the recommendation engine.
// We don't have a list of added, updated, or deleted policies. We use the StagedAction
// field in the staged policies to make this work.
type Recommendation struct {
	NetworkPolicies       []*v3.StagedNetworkPolicy       `json:"networkPolicies"`
	GlobalNetworkPolicies []*v3.StagedGlobalNetworkPolicy `json:"globalNetworkPolicies"`
}

type RecommendationEngine interface {
	ProcessFlow(api.Flow) error
	Recommend() (*Recommendation, error)
}

type SelectorBuilder interface {
	Expression() string
}
