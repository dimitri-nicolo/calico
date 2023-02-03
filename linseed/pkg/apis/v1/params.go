// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import (
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

// DefaultTimeOut is the default timeout that an API will run its query
// until it cancels the execution
const DefaultTimeOut = 60 * time.Second

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

func (p *QueryParams) GetMaxResults() int {
	if p == nil || p.MaxResults == 0 {
		return 1000
	}
	return p.MaxResults
}
