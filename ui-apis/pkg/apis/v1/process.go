// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package v1

import (
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

type ProcessRequest struct {
	// The cluster name. Defaults to "cluster".
	ClusterName string `json:"cluster" validate:"omitempty"`

	// Selector defines a query string for raw logs. [Default: empty]
	Selector string `json:"selector" validate:"omitempty"`

	// Time range.
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"required"`
}

type ProcessResponse struct {
	// Processes contains a list of process info.
	Processes []lapi.ProcessInfo `json:"processes" validate:"required"`
}
