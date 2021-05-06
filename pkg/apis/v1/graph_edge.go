// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import "fmt"

type GraphEdgeID struct {
	// The source and destination of this edge.
	SourceNodeID GraphNodeID `json:"source_node_id"`
	DestNodeID   GraphNodeID `json:"dest_node_id"`
}

type GraphEdge struct {
	// The graph edge ID.
	ID GraphEdgeID `json:"id"`

	// Statistics associated with this edge. Each entry corresponds to the time range in ServiceGraphResponse.
	Stats []GraphStats `json:"stats,omitempty"`

	// The selectors provide the set of selector expressions used to access the raw data that corresponds to this
	// graph edge.
	Selectors GraphSelectors `json:"selectors"`
}

func (e *GraphEdge) IncludeStats(ts []GraphStats) {
	if e.Stats == nil {
		e.Stats = ts
	} else if ts != nil {
		for i := range e.Stats {
			e.Stats[i] = e.Stats[i].Combine(ts[i])
		}
	}
}

func (e GraphEdge) String() string {
	return fmt.Sprintf("Edge(%s -> %s)", e.ID.SourceNodeID, e.ID.DestNodeID)
}
