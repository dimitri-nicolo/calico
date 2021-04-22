// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import "fmt"

type GraphEdgeID struct {
	// The source and destination of this edge.
	SourceNodeID string `json:"source_node_id"`
	DestNodeID   string `json:"dest_node_id"`
}

type GraphEdge struct {
	// The graph edge ID.
	ID GraphEdgeID `json:"id"`

	// Statistics associated with this edge. Each entry corresponds to the time range in ServiceGraphResponse.
	TrafficStats []GraphTrafficStats `json:"traffic_stats,omitempty"`

	// The selectors provide the set of selector expressions used to access the raw data that corresponds to this
	// graph edge.
	Selectors GraphSelectors `json:"selectors"`
}

func (e *GraphEdge) Include(ts []GraphTrafficStats) {
	if e.TrafficStats == nil {
		e.TrafficStats = ts
	} else if ts != nil {
		for i := range e.TrafficStats {
			e.TrafficStats[i] = e.TrafficStats[i].Combine(ts[i])
		}
	}
}

func (e GraphEdge) String() string {
	return fmt.Sprintf("Edge(%s -> %s)", e.ID.SourceNodeID, e.ID.DestNodeID)
}
