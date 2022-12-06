// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.
package flows

import (
	"github.com/projectcalico/calico/lma/pkg/api"
)

type QueryFlows interface {
	QueryElasticsearchFlows(ca CompositeAggregator, params *PolicyRecommendationParams) ([]*api.Flow, error)
}

type PolicyRecommendationParams struct {
	// StartTime represents the beginning of the time window to consider flow logs for.
	StartTime int64 `json:"start_time"`
	// EndTime represents the end of the time window to consider flow logs for.
	EndTime int64 `json:"end_time"`
	// Namespace should correspond to the endpoint referenced by EndpointName.
	Namespace string `json:"namespace"`
	// Unprotected specifies whether results should be restricted to Calico Profiles
	Unprotected bool `json:"unprotected"`

	// Helper values.

	// DocumentIndex represents the elasticsearch index to search through. In this case, it will be
	// the flow log index.
	DocumentIndex string `json:"doc_index"`
}
