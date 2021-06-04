// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import "time"

type ServiceGraphRequest struct {
	// The cluster name. Defaults to "cluster".
	Cluster string `json:"cluster"`

	// Time range.
	TimeRange TimeRange `json:"time_range"`

	// The selected view.
	SelectedView GraphView `json:"selected_view"`

	// Timeout for the request. Defaults to 60s.
	Timeout time.Duration `json:"timeout"`

	// Force a refresh of the data. Generally this should not be required.
	ForceRefresh bool `json:"force_refresh"`
}

type ServiceGraphResponse struct {
	// Time intervals contained in the response. Each node and edge will contain corresponding traffic data sets for
	// each time interval.
	TimeIntervals []TimeRange `json:"time_intervals,omitempty"`

	// Nodes and edges for the graph.
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
