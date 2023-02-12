// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import (
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

// DefaultTimeOut is the default timeout that an API will run its query
// until it cancels the execution
const DefaultTimeOut = 60 * time.Second

type Params interface {
	GetMaxResults() int
	SetMaxResults(int)
	SetAfterKey(map[string]interface{})
	SetTimeout(*v1.Duration)
	SetTimeRange(*lmav1.TimeRange)
}

// QueryParams are request parameters that are shared across all APIs
type QueryParams struct {
	// TimeRange will filter data generated within the specified time range
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"required"`

	// Timeout will limit requests to read/write data to the desired duration
	Timeout *v1.Duration `json:"timeout" validate:"omitempty"`

	// Limit the maximum number of returned results.
	MaxResults int `json:"max_results"`

	// AfterKey is used for pagination. If set, the query will start from the given AfterKey.
	// This is generally passed straight through to the datastore, and its type cannot be
	// guaranteed.
	AfterKey map[string]interface{} `json:"after_key"`
}

func (p *QueryParams) SetMaxResults(i int) {
	p.MaxResults = i
}

func (p *QueryParams) GetMaxResults() int {
	if p == nil || p.MaxResults == 0 {
		return 1000
	}
	return p.MaxResults
}

func (p *QueryParams) SetAfterKey(k map[string]interface{}) {
	p.AfterKey = k
}

func (p *QueryParams) SetTimeout(t *v1.Duration) {
	p.Timeout = t
}

func (p *QueryParams) SetTimeRange(t *lmav1.TimeRange) {
	p.TimeRange = t
}

// LogParams are common for all log APIs.
type LogParams struct {
	// Permissions define a set of resource kinds and namespaces that
	// should be used to filter-in results. If present, any results that
	// do not match the given permissions will be omitted.
	Permissions []v3.AuthorizedResourceVerbs `json:"permissions"`

	// Sort configures the sorting of results.
	Sort []SearchRequestSortBy `json:"sort"`
}

// SearchRequestSortBy allows configuration of sorting of results.
type SearchRequestSortBy struct {
	// The field to sort by.
	Field string `json:"field"`

	// True if the returned results should be in descending order. Default is ascending order.
	Descending bool `json:"descending,omitempty"`
}
