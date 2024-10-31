// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import (
	"fmt"
	"sort"

	"github.com/projectcalico/calico/libcalico-go/lib/set"
	"github.com/projectcalico/calico/ui-apis/pkg/math"
)

// AggregatedProtoPorts holds info about an aggregated set of protocols and ports.
type AggregatedProtoPorts struct {
	ProtoPorts        []AggregatedPorts `json:"proto_ports,omitempty"`
	NumOtherProtocols int               `json:"num_other_protocols,omitempty"`
}

func (a AggregatedProtoPorts) String() string {
	return fmt.Sprintf("Aggregated protocol and ports: %#v", a)
}

func (p *AggregatedProtoPorts) Combine(p2 *AggregatedProtoPorts) *AggregatedProtoPorts {
	if p2 == nil {
		return p
	} else if p == nil {
		return p2
	}

	// Combine the data. This is an approximation since the data is aggregated so was cannot say with any certainty
	// what the aggregated data contains and therefore how the two sets overlap. Just assume that entries that were
	// not in one set but in the other were the aggregated-out values.

	// Determine the full set of protocols that are explicitly defined.
	nodeProtoPorts := map[string]AggregatedPorts{}
	newProtoPorts := map[string]AggregatedPorts{}
	protoset := set.New[string]()
	var protos []string
	for i := range p2.ProtoPorts {
		newProtoPorts[p2.ProtoPorts[i].Protocol] = p2.ProtoPorts[i]
		if !protoset.Contains(p2.ProtoPorts[i].Protocol) {
			protoset.Add(p2.ProtoPorts[i].Protocol)
			protos = append(protos, p2.ProtoPorts[i].Protocol)
		}
	}
	for i := range p.ProtoPorts {
		nodeProtoPorts[p.ProtoPorts[i].Protocol] = p.ProtoPorts[i]
		if !protoset.Contains(p.ProtoPorts[i].Protocol) {
			protoset.Add(p.ProtoPorts[i].Protocol)
			protos = append(protos, p.ProtoPorts[i].Protocol)
		}
	}
	sort.Strings(protos)

	// Iterate through all of the explicitly defined protocols
	nodeOtherProtos := p.NumOtherProtocols
	newOtherProtos := p2.NumOtherProtocols
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

	return &agg
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
