// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

type ServiceGraphRequest struct {
	// Time range.
	TimeRange TimeRange `json:"time_range"`

	// The selected view.
	SelectedView GraphView `json:"selected_view"`
}

type ServiceGraphResponse struct {
	// Time intervals contained in the response. Each node and edge will contain corresponding traffic data sets for
	// each time interval.
	TimeIntervals []TimeRange `json:"time_range,omitempty"`

	// Nodes and edges for the graph.
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
