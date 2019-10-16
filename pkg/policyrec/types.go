// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package policyrec

type PolicyRecommendationParams struct {
	// StartTime represents the beginning of the time window to consider flow logs for.
	StartTime string `json:"start_time"`
	// EndTime represents the end of the time window to consider flow logs for.
	EndTime string `json:"end_time"`
	// EndpointName should correspond to the aggregated name in flow logs.
	EndpointName string `json:"endpoint_name"`
	// Namespace should correspond to the endpoint referenced by EndpointName.
	Namespace string `json:"namespace"`

	// Helper values
	// DocumentIndex represents the elasticsearch index to search through.
	// In this case, it will be the flow log index.
	DocumentIndex string `json:"doc_index"`
}
