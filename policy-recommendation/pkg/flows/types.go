// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.
package flows

import (
	"time"

	"github.com/projectcalico/calico/lma/pkg/api"
)

type PolicyRecommendationQuery interface {
	QueryFlows(params *PolicyRecommendationParams) ([]*api.Flow, error)
}

type PolicyRecommendationParams struct {
	// StartTime represents the beginning of the time window to consider flow logs for.
	StartTime time.Duration `json:"start_time"`
	// EndTime represents the end of the time window to consider flow logs for.
	EndTime time.Duration `json:"end_time"`
	// Namespace should correspond to the endpoint referenced by EndpointName.
	Namespace string `json:"namespace"`
	// Unprotected specifies whether results should be restricted to Calico Profiles
	Unprotected bool `json:"unprotected"`
}
