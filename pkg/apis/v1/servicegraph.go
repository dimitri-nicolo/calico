// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

type ServiceGraphRequest struct {
	// Time range.
	TimeRange TimeRange `json:"time_range,omitempty"`

	// Include time series. If set to true, the time range will be divided up into equally spaced chunks. The response
	// will contain a set of time intervals. Each graph node and edge contains a slice of traffic stats, each traffic
	// stat contains an index that indicates the time interval the stats corresponds to. If no stats were recorded then
	// the index will be missing from the slice.
	IncludeTimeSeries bool

	// The selected view.
	SelectedView GraphView `json:"selected_view,omitempty"`
}

type ServiceGraphResponse struct {
	// Time intervals contained in the response. Each node and edge will contain corresponding traffic data sets for
	// each time interval.
	TimeIntervals []TimeRange `json:"time_range,omitempty"`

	// Nodes and edges for the graph.
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
