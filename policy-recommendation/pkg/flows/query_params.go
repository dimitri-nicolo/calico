// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.
package flows

import (
	"time"
)

type RecommendationFlowLogQueryParams struct {
	// StartTime represents the beginning of the time window to consider flow logs for.
	StartTime time.Duration `json:"start_time"`
	// EndTime represents the end of the time window to consider flow logs for.
	EndTime time.Duration `json:"end_time"`
	// Namespace should correspond to the endpoint referenced by EndpointName.
	Namespace string `json:"namespace"`
	// Unprotected specifies whether results should be restricted to Calico Profiles
	Unprotected bool `json:"unprotected"`
}

// NewRecommendationFlowLogQueryParams returns the policy parameters of a namespaces based policy
// recommendation query to flow logs
func NewRecommendationFlowLogQueryParams(st time.Duration, ns, cl string) *RecommendationFlowLogQueryParams {
	return &RecommendationFlowLogQueryParams{
		StartTime: st,
		// This is used in a function returning a duration as a string in seconds relative to now. Zero
		// means the end time will be now, at the time the query is executed.
		EndTime:     time.Duration(0),
		Namespace:   ns,
		Unprotected: true,
	}
}
