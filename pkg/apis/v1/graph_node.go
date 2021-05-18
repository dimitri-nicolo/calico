// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/tigera/es-proxy/pkg/math"

	"github.com/projectcalico/libcalico-go/lib/set"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// The set of events correlated to this node. The json is rendered in the following format (which differs from the
	// struct definitions):
	// 			"events": [
	//				{
	//					"tiger_event_id": "abcde",
	//					"description": "A thing occurred, not sure when"
	//				},
	//				{
	//					"kubernetes_event_name": "n2",
	//					"kubernetes_event_namespace": "n",
	//					"description": "A k8s thing occurred",
	//					"time": "1973-03-14T00:00:00Z"
	//                }
	//			]
	// This contains two type of event:
	// Kubernetes events - id is using namespace and name
	// Tigera event - id is an id that corresponds to an entry in elasticsearch.
	//
	// This may be subject to change as the current event structure is poorly defined.
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
	if p == nil {
		return
	} else if n.AggregatedProtoPorts == nil {
		n.AggregatedProtoPorts = p
		return
	}

	// Combine the data. This is an approximation since the data is aggregated so was cannot say with any certainty
	// what the aggregated data contains and therefore how the two sets overlap. Just assume that entries that were
	// not in one set but in the other were the aggregated-out values.

	// Determine the full set of protocols that are explicitly defined.
	nodeProtoPorts := map[string]AggregatedPorts{}
	newProtoPorts := map[string]AggregatedPorts{}
	protoset := set.New()
	var protos []string
	for i := range p.ProtoPorts {
		newProtoPorts[p.ProtoPorts[i].Protocol] = p.ProtoPorts[i]
		if !protoset.Contains(p.ProtoPorts[i].Protocol) {
			protoset.Add(p.ProtoPorts[i].Protocol)
			protos = append(protos, p.ProtoPorts[i].Protocol)
		}
	}
	for i := range n.AggregatedProtoPorts.ProtoPorts {
		nodeProtoPorts[n.AggregatedProtoPorts.ProtoPorts[i].Protocol] = n.AggregatedProtoPorts.ProtoPorts[i]
		if !protoset.Contains(n.AggregatedProtoPorts.ProtoPorts[i].Protocol) {
			protoset.Add(n.AggregatedProtoPorts.ProtoPorts[i].Protocol)
			protos = append(protos, n.AggregatedProtoPorts.ProtoPorts[i].Protocol)
		}
	}
	sort.Strings(protos)

	// Iterate through all of the explicitly defined protocols
	nodeOtherProtos := n.AggregatedProtoPorts.NumOtherProtocols
	newOtherProtos := p.NumOtherProtocols
	agg := AggregatedProtoPorts{}
	for _, proto := range protos {
		nodePorts, nodeOk := nodeProtoPorts[proto]
		newPorts, newOk := newProtoPorts[proto]

		if !newOk {
			// The node has a protocol that the new set does not. Use the node value unchanged. Assume the protocol
			// is one of the other aggregated values so decrement the other protocols for the new set.
			agg.ProtoPorts = append(agg.ProtoPorts, nodePorts)
			newOtherProtos--
		} else if !nodeOk {
			// The new set has a protocol that the node does not. Use the new set unchanged. Assume the protocol
			// is one of the other aggregated values so decrement the other protocols for the node set.
			agg.ProtoPorts = append(agg.ProtoPorts, newPorts)
			nodeOtherProtos--
		} else {
			// Create a sorted superset of ranges. This will contain overlapping entries - we'll sort that out next.
			// Determine the total number of ports in each as we go.
			nodeTotalPorts := nodePorts.NumOtherPorts
			newTotalPorts := newPorts.NumOtherPorts
			allRanges := make([]PortRange, 0, len(nodePorts.PortRanges)+len(newPorts.PortRanges))
			for i := range nodePorts.PortRanges {
				allRanges = append(allRanges, nodePorts.PortRanges[i])
				nodeTotalPorts += nodePorts.PortRanges[i].Num()
			}
			for i := range newPorts.PortRanges {
				allRanges = append(allRanges, newPorts.PortRanges[i])
				newTotalPorts += newPorts.PortRanges[i].Num()
			}
			sort.Slice(allRanges, func(i, j int) bool {
				return allRanges[i].MinPort < allRanges[j].MinPort
			})

			var combinedRanges []PortRange
			var numPortsInRanges int
			var pr PortRange
			for i := range allRanges {
				if i == 0 {
					pr = allRanges[i]
					continue
				}

				if pr.MaxPort > allRanges[i].MaxPort {
					// The previous entry wholly covers this one, so skip.
					continue
				} else if pr.MaxPort < allRanges[i].MinPort-1 {
					// The ranges are not orverlapping nor contiguous, so add the previous and track the next.
					combinedRanges = append(combinedRanges, pr)
					numPortsInRanges += pr.Num()
					pr = allRanges[i]
				} else if pr.MaxPort < allRanges[i].MaxPort {
					// Ranges are either partially overlapping or contiguous and the next max is higher - so updatae
					// the max value.
					pr.MaxPort = allRanges[i].MaxPort
				}
			}
			if pr.MinPort > 0 {
				combinedRanges = append(combinedRanges, pr)
				numPortsInRanges += pr.Num()
			}

			// Recalculate the numer of other ports, and we'll take the larger of the two for the new data.
			otherPorts := math.MaxIntGtZero(nodeTotalPorts-numPortsInRanges, newTotalPorts-numPortsInRanges)
			agg.ProtoPorts = append(agg.ProtoPorts, AggregatedPorts{
				Protocol:      proto,
				PortRanges:    combinedRanges,
				NumOtherPorts: otherPorts,
			})
		}
	}

	// Set the guestimated number of other protocols.
	agg.NumOtherProtocols = math.MaxIntGtZero(nodeOtherProtos, newOtherProtos)

	// Update the node.
	n.AggregatedProtoPorts = &agg
}

func (n *GraphNode) IncludeService(s types.NamespacedName) {
	if n.Services == nil {
		n.Services = make(Services)
	}
	n.Services[s] = struct{}{}
}

func (n *GraphNode) IncludeEvent(id GraphEventID, event GraphEvent) {
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

// AggregatedProtoPorts holds info about an aggregated set of protocols and ports.
type AggregatedProtoPorts struct {
	ProtoPorts        []AggregatedPorts `json:"proto_ports,omitempty"`
	NumOtherProtocols int               `json:"num_other_protocols,omitempty"`
}

func (a AggregatedProtoPorts) String() string {
	return fmt.Sprintf("Aggregated protocol and ports: %#v", a)
}

type AggregatedPorts struct {
	Protocol      string      `json:"protocol,omitempty"`
	PortRanges    []PortRange `json:"port_ranges,omitempty"`
	NumOtherPorts int         `json:"num_other_ports,omitempty"`
}

type PortRange struct {
	MinPort int `json:"min_port,omitempty"`
	MaxPort int `json:"max_port,omitempty"`
}

func (p PortRange) Num() int {
	return p.MaxPort - p.MinPort + 1
}

// Services is the set of services associated with a node. This is JSON marshaled as a slice.
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

// SortableServices is used to sort a set of services.
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

// The ID for the event.  This allows the particular event to be cross referenced with the event source.
type GraphEventID struct {
	TigeraEventID     string
	KubernetesEventID types.NamespacedName
}

// Details of the event. This does not contain the full event details, but the original event may be cross referenced
// from the ID.
type GraphEvent struct {
	Description string       `json:"description,omitempty"`
	Timestamp   *metav1.Time `json:"time,omitempty"`
}

// GraphEvents is used to store event details. Stored as a map to handle deduplication, this is JSON marshaled as a
// slice.
type GraphEvents map[GraphEventID]GraphEvent

type graphEventWithID struct {
	TigeraEventID            string `json:"tiger_event_id,omitempty"`
	KubernetesEventName      string `json:"kubernetes_event_name,omitempty"`
	KubernetesEventNamespace string `json:"kubernetes_event_namespace,omitempty"`
	GraphEvent               `json:",inline"`
}

func (e GraphEvents) MarshalJSON() ([]byte, error) {
	var ids []graphEventWithID
	for id, ev := range e {
		ids = append(ids, graphEventWithID{
			TigeraEventID:            id.TigeraEventID,
			KubernetesEventName:      id.KubernetesEventID.Name,
			KubernetesEventNamespace: id.KubernetesEventID.Namespace,
			GraphEvent:               ev,
		})
	}
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].Timestamp == nil && ids[j].Timestamp != nil {
			return true
		}
		return ids[i].Timestamp.Before(ids[j].Timestamp)
	})
	return json.Marshal(ids)
}
