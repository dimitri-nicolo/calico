// Copyright (c) 2021-2023 Tigera, Inc. All rights reserved.
package v1

import (
	"encoding/json"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

// SearchRequest contains the parameters for defining raw logs queries. This interface returns a maximum of 10,000
// results. For more nuanced queries, use a specific Selector.
type CommonSearchRequest struct {
	// ClusterName defines the name of the cluster a connection will be performed on.
	ClusterName string `json:"cluster" validate:"omitempty"`

	// Time range.
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"omitempty"`

	// Selector defines a query string for raw logs. [Default: empty]
	// Selector is set in the Service Graph page for logs and events.
	Selector string `json:"selector" validate:"omitempty"`

	// Filter defines a list of Elastic filters for raw logs. [Default: empty]
	// Filter is set in the Alert List page for events.
	Filter []json.RawMessage `json:"filter" validate:"omitempty"`

	// PageSize defines the page size of raw flow logs to retrieve per search. [Default: 100]
	PageSize *int `json:"page_size" validate:"gt=0,lte=1000"`

	// PageNum specifies which page of results to return. Indexed from 0. [Default: 0]
	PageNum int `json:"page_num" validate:"gte=0"`

	// Sort by field and direction.
	SortBy []SearchRequestSortBy `json:"sort_by" validate:"omitempty"`

	// Timeout for the request. Defaults to 60s.
	Timeout *v1.Duration `json:"timeout" validate:"omitempty"`
}

// SearchRequestSortBy encapsulates the sort-by parameters a search query will return results by.
type SearchRequestSortBy struct {
	// The field to sort by.
	Field string `json:"field"`

	// True if the returned results should be in descending order. Default is ascending order.
	Descending bool `json:"descending"`
}

type FlowLogSearchRequest struct {
	CommonSearchRequest

	// PolicyMatches is used to fetch flowlogs by policy attributes. If multiple policy match
	// attributes are provided, they are combined by logical OR.
	PolicyMatches []PolicyMatch `json:"policy_matches,omitempty" validate:"omitempty"`
}

type PolicyMatch struct {
	// Tier for the policy.
	Tier string `json:"tier,omitempty" validate:"omitempty"`

	// The action taken by the policy.
	Action *lapi.FlowAction `json:"action,omitempty" validate:"omitempty"`

	// Namespace and name of the policy.
	Namespace *string `json:"namespace,omitempty" validate:"omitempty"`
	Name      *string `json:"name,omitempty" validate:"omitempty"`
}

// SearchResponse contains the response of a raw log search.
type SearchResponse struct {
	// True if ElasticSearch timed out.
	TimedOut bool `json:"timed_out"`

	// Search time.
	Took v1.Duration `json:"took"`

	// The total number of pages. The maximum page number is thus given by (NumPages - 1). The maximum number of
	// pages will be capped to ensure the maximum number of documents that can be enumerated is capped at 10,000.
	NumPages int `json:"num_pages"`

	// The total number of hits.
	TotalHits int `json:"total_hits"`

	// The actual hits returned, as a raw json.
	Hits []json.RawMessage `json:"hits"`
}
