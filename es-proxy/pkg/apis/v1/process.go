// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package v1

import (
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
	Processes []Process `json:"processes" validate:"required"`
}

type Process struct {
	// Name of the process.
	Name string `json:"name" validate:"required"`

	// Endpoint executes the process.
	Endpoint string `json:"endpoint" validate:"required"`

	// InstanceCount counts the unique process id.
	InstanceCount int `json:"instanceCount" validate:"required"`
}
