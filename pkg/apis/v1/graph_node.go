// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import "fmt"

type GraphNodeType string

const (
	GraphNodeTypeNamespace    GraphNodeType = "namespace"
	GraphNodeTypeLayer        GraphNodeType = "layer"
	GraphNodeTypeServiceGroup GraphNodeType = "svcgp"
	GraphNodeTypeService      GraphNodeType = "svc"
	GraphNodeTypeServicePort  GraphNodeType = "svcport"
	GraphNodeTypeReplicaSet   GraphNodeType = "rep"
	GraphNodeTypeWorkload     GraphNodeType = "wep"
	GraphNodeTypeHostEndpoint GraphNodeType = "hep"
	GraphNodeTypeNetwork      GraphNodeType = "net"
	GraphNodeTypeNetworkSet   GraphNodeType = "ns"
	GraphNodeTypeNode         GraphNodeType = "node"
	GraphNodeTypeNodeGroup    GraphNodeType = "nodegp"
	GraphNodeTypePort         GraphNodeType = "port"
	GraphNodeTypeProcess      GraphNodeType = "process"
	GraphNodeTypeUnknown      GraphNodeType = ""
)

type GraphNode struct {
	// The ID of this graph node. See doc file in /pkg/apis/es for details on the node ID construction.
	ID string `json:"id"`

	// The parent (or outer) node.
	ParentID string `json:"parent_id"`

	// Node metadata.
	Type        GraphNodeType `json:"type"`
	Namespace   string        `json:"namespace,omitempty"`
	Name        string        `json:"name,omitempty"`
	ServicePort string        `json:"service_port,omitempty"`
	Protocol    string        `json:"protocol,omitempty"`
	Port        int           `json:"port,omitempty"`
	Layer       string        `json:"layer,omitempty"`

	// Aggregated port and protocol stats
	Aggregated *AggregatedProtoPorts `json:"aggregated,omitempty"`

	// Traffic stats for packets flowing between endpoints within this graph node. Each entry corresponds to the
	TrafficStats []GraphTrafficStats `json:"traffic_stats"`

	// Whether this node is further expandable. In other words if this node is added as an `Expanded` node to
	// the `GraphView` then the results will return additional nodes and edges.
	Expandable bool `json:"expandable"`

	// Whether this node may be further followed in the egress connection direction or ingress connection direction.
	// If true, this node can be added to FollowedEgress or FollowedIngress in the `GraphView` to return additional
	// nodes and edges.
	FollowEgress  bool `json:"follow_egress"`
	FollowIngress bool `json:"follow_ingress"`
}

func (n *GraphNode) Include(ts []GraphTrafficStats) {
	if n.TrafficStats == nil {
		n.TrafficStats = ts
	} else if ts != nil {
		for i := range n.TrafficStats {
			n.TrafficStats[i] = n.TrafficStats[i].Add(ts[i])
		}
	}
}

func (n GraphNode) String() string {
	if n.ParentID == "" {
		return fmt.Sprintf("Node(%s; expandable=%v)", n.ID, n.Expandable)
	}
	return fmt.Sprintf("Node(%s; parent=%s; expandable=%v)", n.ID, n.ParentID, n.Expandable)
}

type AggregatedProtoPorts struct {
	ProtoPorts        []AggregatedPorts `json:"protoPorts,omitempty"`
	NumOtherProtocols int               `json:"numOtherProtocols,omitempty"`
}

func (a AggregatedProtoPorts) String() string {
	return fmt.Sprintf("Aggregated protocol and ports: %#v", a)
}

type AggregatedPorts struct {
	Protocol      string      `json:"protocol,omitempty"`
	PortRanges    []PortRange `json:"portRanges,omitempty"`
	NumOtherPorts int         `json:"numOtherPorts,omitempty"`
}

type PortRange struct {
	MinPort int `json:"minPort,omitempty"`
	MaxPort int `json:"maxPort,omitempty"`
}

func (p PortRange) Num() int {
	return p.MaxPort - p.MinPort + 1
}
