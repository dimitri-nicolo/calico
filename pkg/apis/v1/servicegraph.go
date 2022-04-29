// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

type ServiceGraphRequest struct {
	// The cluster name. Defaults to "cluster".
	Cluster string `json:"cluster" validate:"omitempty"`

	// Time range.
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"required"`

	// The selected view.
	SelectedView GraphView `json:"selected_view" validate:"omitempty"`

	// Timeout for the request. Defaults to 60s.
	Timeout v1.Duration `json:"timeout" validate:"omitempty"`

	// Force a refresh of the data. Generally this should not be required.
	ForceRefresh bool `json:"force_refresh" validate:"omitempty"`
}

type ServiceGraphResponse struct {
	// Time intervals contained in the response. Each node and edge will contain corresponding traffic data sets for
	// each time interval.
	TimeIntervals []lmav1.TimeRange `json:"time_intervals,omitempty"`

	// Nodes and edges for the graph.
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`

	// Whether the data is truncated (query window needs to be reduced).
	Truncated bool `json:"truncated"`

	// Selectors for the view.
	Selectors GraphSelectors `json:"selectors"`
}
