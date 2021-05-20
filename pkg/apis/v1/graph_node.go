// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"fmt"
)

type GraphNodeType string

const (
	GraphNodeTypeNamespace    GraphNodeType = "namespace"
	GraphNodeTypeLayer        GraphNodeType = "layer"
	GraphNodeTypeServiceGroup GraphNodeType = "svcgp"
	GraphNodeTypeService      GraphNodeType = "svc"
	GraphNodeTypeServicePort  GraphNodeType = "svcport"
	GraphNodeTypeReplicaSet   GraphNodeType = "rep"
	GraphNodeTypeWorkload     GraphNodeType = "wep"
	GraphNodeTypeHosts        GraphNodeType = "hosts"
	GraphNodeTypeHostEndpoint GraphNodeType = "hep"
	GraphNodeTypeNetwork      GraphNodeType = "net"
	GraphNodeTypeNetworkSet   GraphNodeType = "ns"
	GraphNodeTypeNode         GraphNodeType = "node"
	GraphNodeTypePort         GraphNodeType = "port"
	GraphNodeTypeUnknown      GraphNodeType = ""
)

type GraphNodeID string

type GraphNode struct {
	// The ID of this graph node. See doc file in /pkg/apis/es for details on the node ID construction.
	ID GraphNodeID `json:"id"`

	// The parent (or outer) node.
	ParentID GraphNodeID `json:"parent_id,omitempty"`

	// Node metadata.
	Type        GraphNodeType `json:"type"`
	Namespace   string        `json:"namespace,omitempty"`
	Name        string        `json:"name,omitempty"`
	ServicePort string        `json:"service_port,omitempty"`
	Protocol    string        `json:"protocol,omitempty"`
	Port        int           `json:"port,omitempty"`
	Layer       string        `json:"layer,omitempty"`

	// The services contained within this group.
	Services NamespacedNames `json:"services,omitempty"`

	// Aggregated protocol and port information for this node. Protocols and ports that are explicitly included in the
	// graph because they are part of an expanded service are not included in this aggregated set.
	AggregatedProtoPorts *AggregatedProtoPorts `json:"aggregated_proto_ports,omitempty"`

	// Stats for packets flowing between endpoints within this graph node. Each entry corresponds to the
	// a time slice as specified in the main response object.
	Stats []GraphStats `json:"stats,omitempty"`

	// Whether this node is further expandable. In other words if this node is added as an `Expanded` node to
	// the `GraphView` then the results will return additional nodes and edges.
	Expandable bool `json:"expandable,omitempty"`

	// Whether this node may be further followed in the egress connection direction or ingress connection direction.
	// If true, this node can be added to FollowedEgress or FollowedIngress in the `GraphView` to return additional
	// nodes and edges.
	FollowEgress  bool `json:"follow_egress,omitempty"`
	FollowIngress bool `json:"follow_ingress,omitempty"`

	// The selectors provide the set of selector expressions used to access the raw data that corresponds to this
	// graph node.
	Selectors GraphSelectors `json:"selectors"`

	// The set of events correlated to this node. The json is rendered in the following format (which differs from the
	// struct definitions):
	//   "events": [
	//     {
	//       "id": {
	//         "type": "kubernetes"
	//         "name": "n2",
	//         "namespace": "n",
	//       },
	//       "description": "A k8s thing occurred",
	//       "time": "1973-03-14T00:00:00Z"
	//     },
	//     {
	//       "id": {
	//         "type": "alert"
	//         "id": "aifn93hrbv_Ds",
	//         "name": "policy.pod",
	//       },
	//       "description": "A pod was modified occurred",
	//       "time": "1973-03-14T00:00:00Z"
	//     }
	//   ]
	Events GraphEvents `json:"events,omitempty"`
}

func (n *GraphNode) IncludeStats(ts []GraphStats) {
	if n.Stats == nil {
		n.Stats = ts
	} else if ts != nil {
		for i := range n.Stats {
			n.Stats[i] = n.Stats[i].Combine(ts[i])
		}
	}
}

func (n *GraphNode) IncludeAggregatedProtoPorts(p *AggregatedProtoPorts) {
	n.AggregatedProtoPorts = n.AggregatedProtoPorts.Combine(p)
}

func (n *GraphNode) IncludeService(s NamespacedName) {
	if n.Services == nil {
		n.Services = make(NamespacedNames)
	}
	n.Services[s] = struct{}{}
}

func (n *GraphNode) IncludeEvent(id GraphEventID, event GraphEventDetails) {
	if n.Events == nil {
		n.Events = make(GraphEvents)
	}
	n.Events[id] = event
}

func (n GraphNode) String() string {
	if n.ParentID == "" {
		return fmt.Sprintf("Node(%s; expandable=%v)", n.ID, n.Expandable)
	}
	return fmt.Sprintf("Node(%s; parent=%s; expandable=%v)", n.ID, n.ParentID, n.Expandable)
}
