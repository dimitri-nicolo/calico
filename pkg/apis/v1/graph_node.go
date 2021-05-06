// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/types"
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
	ParentID GraphNodeID `json:"parent_id"`

	// Node metadata.
	Type        GraphNodeType `json:"type"`
	Namespace   string        `json:"namespace,omitempty"`
	Name        string        `json:"name,omitempty"`
	ServicePort string        `json:"service_port,omitempty"`
	Protocol    string        `json:"protocol,omitempty"`
	Port        int           `json:"port,omitempty"`
	Layer       string        `json:"layer,omitempty"`

	// The services contained within this group.
	Services Services `json:"services,omitempty"`

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

	// The set of events correlated to this node
	EventIDs GraphEventIDs `json:"event_ids,omitempty"`
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
	if p == nil {
		return
	} else if n.AggregatedProtoPorts == nil {
		n.AggregatedProtoPorts = p
		return
	}

	// We can't really combine the aggregated ports and protos very easily. If one has more protocols than the other
	// use that.
	if p.NumOtherProtocols > n.AggregatedProtoPorts.NumOtherProtocols {
		n.AggregatedProtoPorts = p
		return
	}

	// Number of other protocols is the same.  Combine the data.
	otherProtos := p.NumOtherProtocols
	protoPorts := map[string]AggregatedPorts{}
	var protos []string
	for _, ap := range n.AggregatedProtoPorts.ProtoPorts {
		protoPorts[ap.Protocol] = ap
		protos = append(protos, ap.Protocol)
	}
	for _, ap := range p.ProtoPorts {
		if existing, ok := protoPorts[ap.Protocol]; ok {
			if ap.NumOtherPorts > existing.NumOtherPorts {
				protoPorts[ap.Protocol] = ap
			}
		} else {
			protoPorts[ap.Protocol] = ap
			protos = append(protos, ap.Protocol)
			otherProtos--
		}
	}
	if otherProtos < 0 {
		otherProtos = 0
	}

	app := AggregatedProtoPorts{
		NumOtherProtocols: otherProtos,
	}
	sort.Strings(protos)
	for _, proto := range protos {
		ap := protoPorts[proto]
		app.ProtoPorts = append(app.ProtoPorts, ap)
	}

	n.AggregatedProtoPorts = &app
}

func (n *GraphNode) IncludeService(s types.NamespacedName) {
	if n.Services == nil {
		n.Services = make(Services)
	}
	n.Services[s] = struct{}{}
}

func (n *GraphNode) IncludeEvent(id string) {
	if n.EventIDs == nil {
		n.EventIDs = make(GraphEventIDs)
	}
	n.EventIDs[id] = struct{}{}
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
	return fmt.Sprintf("AggregatedProtoPorts protocol and ports: %#v", a)
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

type Services map[types.NamespacedName]struct{}

func (s Services) MarshalJSON() ([]byte, error) {
	var svcs SortableServices
	for svc := range s {
		svcs = append(svcs, svc)
	}
	sort.Sort(svcs)

	buffer := bytes.NewBufferString("[")
	length := len(svcs)
	count := 0
	for _, value := range svcs {
		jsonValueNamespace, err := json.Marshal(value.Namespace)
		if err != nil {
			return nil, err
		}
		jsonValueName, err := json.Marshal(value.Name)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("{\"namespace\":%s,\"name\":%s}", string(jsonValueNamespace), string(jsonValueName)))
		count++
		if count < length {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

type SortableServices []types.NamespacedName

func (s SortableServices) Len() int {
	return len(s)
}
func (s SortableServices) Less(i, j int) bool {
	if s[i].Namespace < s[j].Namespace {
		return true
	} else if s[i].Namespace == s[j].Namespace && s[i].Name < s[j].Name {
		return true
	}
	return false
}
func (s SortableServices) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type GraphEventIDs map[string]struct{}

func (e GraphEventIDs) MarshalJSON() ([]byte, error) {
	var ids []string
	for id := range e {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return json.Marshal(ids)
}
