// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"encoding/json"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lmav1 "github.com/tigera/lma/pkg/apis/v1"
)

const (
	defaultRequestTimeout = 60 * time.Second
	defaultPageSize       = 100
)

// SearchRequest contains the parameters for defining raw logs queries. This interface returns a maximum of 10,000
// results. For more nuanced queries, use a specific Selector.
type SearchRequest struct {
	// ClusterName defines the name of the cluster a connection will be performed on.
	ClusterName string `json:"cluster" validate:"omitempty"`

	// Time range. Required.
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"required"`

	// Selector defines a query string for raw logs. [Default: empty]
	Selector string `json:"selector" validate:"omitempty"`

	// PageSize defines the page size of raw flow logs to retrieve per search. [Default: 100]
	PageSize int `json:"page_size" validate:"gte=0,lte=1000"`

	// PageNum specifies which page of results to return. Indexed from 0. [Default: 0]
	PageNum int `json:"page_num" validate:"gte=0"`

	// Sort by field and direction.
	SortBy []SearchRequestSortBy `json:"sort_by"`

	// Timeout for the request. Defaults to 60s.
	Timeout v1.Duration `json:"timeout"`
}

// decodeRequestBody sets the search parameters to their default values.
func (params *SearchRequest) DefaultParams() {
	params.ClusterName = "cluster"
	params.PageSize = defaultPageSize
	params.Timeout.Duration = defaultRequestTimeout
}

// SearchRequestSortBy encapsulates the sort-by parameters a search query will return results by.
type SearchRequestSortBy struct {
	// The field to sort by.
	Field string `json:"field"`

	// True if the returned results should be in descending order. Default is ascending order.
	Descending bool `json:"descending"`
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
