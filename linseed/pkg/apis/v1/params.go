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

// QueryParams are request parameters that are shared across
// all APIs
type QueryParams struct {
	// TimeRange will filter data generated within the specified time range
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"required"`

	// Timeout will limit requests to read/write data to the desired duration
	Timeout *v1.Duration `json:"timeout" validate:"omitempty"`
}
