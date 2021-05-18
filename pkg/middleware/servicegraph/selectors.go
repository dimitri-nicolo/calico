// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	maxSelectorItemsPerGroup = 50
)

// SelectorPairs contains source and dest pairs of graph node selectors.
// The source selector represents the selector used when an edge originates from that node.
// The dest selector represents the selector used when an edge terminates at that node.
//
// This is a convenience since most of the selectors can be split into source and dest related queries. It is not
// required for the API.
type SelectorPairs struct {
	Source v1.GraphSelectors
	Dest   v1.GraphSelectors
}

func (s SelectorPairs) ToNodeSelectors() v1.GraphSelectors {
	return s.Source.Or(s.Dest)
}

func (s SelectorPairs) ToEdgeSelectors() v1.GraphSelectors {
	return s.Source.And(s.Dest)
}

// And combines two sets of selectors by ANDing them together.
func (s SelectorPairs) And(s2 SelectorPairs) SelectorPairs {
	return SelectorPairs{
		Source: s.Source.And(s2.Source),
		Dest:   s.Dest.And(s2.Dest),
	}
}

// Or combines two sets of selectors by ORing them together.
func (s SelectorPairs) Or(s2 SelectorPairs) SelectorPairs {
	return SelectorPairs{
		Source: s.Source.Or(s2.Source),
		Dest:   s.Dest.Or(s2.Dest),
	}
}

func NewSelectorHelper(view *ParsedView, aggHelper AggregationHelper) *SelectorHelper {
	return &SelectorHelper{
		view:      view,
		aggHelper: aggHelper,
	}
}

type SelectorHelper struct {
	view      *ParsedView
	aggHelper AggregationHelper
}

// GetLayerNodeSelectors returns the selectors for a layer node (as specified on the request).
func (s *SelectorHelper) GetLayerNodeSelectors(layer string) SelectorPairs {
	gs := SelectorPairs{}
	for _, n := range s.view.Layers.LayerToNamespaces[layer] {
		gs = gs.Or(s.GetNamespaceNodeSelectors(n))
	}
	for _, sg := range s.view.Layers.LayerToServiceGroups[layer] {
		gs = gs.Or(s.GetServiceGroupNodeSelectors(sg))
	}
	for _, ep := range s.view.Layers.LayerToEndpoints[layer] {
		_, isAgg := mapGraphNodeTypeToRawType(ep.Type)
		if isAgg {
			gs = gs.Or(s.GetEndpointNodeSelectors(ep.Type, ep.Namespace, ep.NameAggr, ep.Proto, ep.Port, NoDirection))
		} else {
			gs = gs.Or(s.GetEndpointNodeSelectors(ep.Type, ep.Namespace, ep.Name, ep.Proto, ep.Port, NoDirection))
		}
	}
	return gs
}

// GetNamespaceNodeSelectors returns the selectors for a namespace node.
func (s *SelectorHelper) GetNamespaceNodeSelectors(namespace string) SelectorPairs {
	return SelectorPairs{
		Source: v1.GraphSelectors{
			L3Flows: v1.NewGraphSelector(v1.OpEqual, "source_namespace", namespace),
			L7Flows: v1.NewGraphSelector(v1.OpEqual, "src_namespace", namespace),
			DNSLogs: v1.NewGraphSelector(v1.OpEqual, "client_namespace", namespace),
		},
		Dest: v1.GraphSelectors{
			L3Flows: v1.NewGraphSelector(v1.OpEqual, "dest_namespace", namespace),
			L7Flows: v1.NewGraphSelector(v1.OpEqual, "dest_namespace", namespace),
		},
	}
}

// GetServiceNodeSelectors returns the selectors for a service node.
func (s *SelectorHelper) GetServiceNodeSelectors(svc types.NamespacedName) SelectorPairs {
	return SelectorPairs{
		Dest: v1.GraphSelectors{
			L3Flows: v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "dest_service_namespace", svc.Namespace),
				v1.NewGraphSelector(v1.OpEqual, "dest_service_name", svc.Name),
			),
			L7Flows: v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "dest_service_namespace", svc.Namespace),
				v1.NewGraphSelector(v1.OpEqual, "dest_service_name", svc.Name),
			),
		},
	}
}

// GetServicePortNodeSelectors returns the selectors for a service port node.
func (s *SelectorHelper) GetServicePortNodeSelectors(sp ServicePort) SelectorPairs {
	l3Sel := v1.NewGraphSelector(v1.OpAnd,
		v1.NewGraphSelector(v1.OpEqual, "dest_service_namespace", sp.Namespace),
		v1.NewGraphSelector(v1.OpEqual, "dest_service_name", sp.Name),
	)
	if sp.Port != "" {
		l3Sel = v1.NewGraphSelector(v1.OpAnd,
			l3Sel,
			v1.NewGraphSelector(v1.OpEqual, "dest_service_port", sp.Port),
		)
	}

	if sp.Port != "" {
		l3Sel = v1.NewGraphSelector(v1.OpAnd,
			l3Sel,
			v1.NewGraphSelector(v1.OpEqual, "dest_service_port", sp.Port),
		)
	}

	var l7Sel *v1.GraphSelector
	if sp.Proto == "tcp" {
		l7Sel = v1.NewGraphSelector(v1.OpAnd,
			v1.NewGraphSelector(v1.OpEqual, "dest_service_namespace", sp.Namespace),
			v1.NewGraphSelector(v1.OpEqual, "dest_service_name", sp.Name),
		)
	}

	return SelectorPairs{
		Source: v1.GraphSelectors{
			// L3 source selector is calculated in graphconstructor by ORing together all of the ingress sources to the
			// service.  We need to do this because flows are recorded at source and dest and may not have service
			// information available in the dest recorded flows.
			// L7 service selector is the same for source and dest.
			L7Flows: l7Sel,
		},
		Dest: v1.GraphSelectors{
			L3Flows: l3Sel,
			L7Flows: l7Sel,
		},
	}
}

// GetServiceGroupNodeSelectors returns the selectors for a service group node.
func (s *SelectorHelper) GetServiceGroupNodeSelectors(sg *ServiceGroup) SelectorPairs {
	// Selectors depend on whether the service endpoints record the flow. If only the source records the flow then we
	// limit the search based on the service selectors.
	allSvcs := make(map[types.NamespacedName]struct{})
	allEps := make(map[FlowEndpoint]struct{})

	for sp, eps := range sg.ServicePorts {
		for ep := range eps {
			switch ep.Type {
			case v1.GraphNodeTypeHostEndpoint, v1.GraphNodeTypeWorkload, v1.GraphNodeTypeReplicaSet:
				allEps[ep] = struct{}{}
			default:
				allSvcs[sp.NamespacedName] = struct{}{}
			}
		}
	}

	var gs SelectorPairs
	for svc := range allSvcs {
		gs = gs.Or(s.GetServiceNodeSelectors(svc))
	}
	for ep := range allEps {
		_, isAgg := mapGraphNodeTypeToRawType(ep.Type)
		if isAgg {
			gs = gs.Or(s.GetEndpointNodeSelectors(ep.Type, ep.Namespace, ep.NameAggr, ep.Proto, ep.Port, NoDirection))
		} else {
			gs = gs.Or(s.GetEndpointNodeSelectors(ep.Type, ep.Namespace, ep.Name, ep.Proto, ep.Port, NoDirection))
		}
	}
	return gs
}

// GetEndpointNodeSelectors returns the selectors for an endpoint node.
func (s *SelectorHelper) GetEndpointNodeSelectors(epType v1.GraphNodeType, namespace, name, proto string, port int, dir Direction) SelectorPairs {
	rawType, isAgg := mapGraphNodeTypeToRawType(epType)
	namespace = blankToSingleDash(namespace)

	var l3Dest, l7Dest, l3Source, l7Source, dnsSource *v1.GraphSelector
	if rawType == "wep" {
		// DNS logs are only recorded for wep types.
		if isAgg {
			dnsSource = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "client_namespace", namespace),
				v1.NewGraphSelector(v1.OpEqual, "client_name_aggr", name),
			)
		} else {
			dnsSource = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "client_namespace", namespace),
				v1.NewGraphSelector(v1.OpEqual, "client_name", name),
			)
		}

		// Similarly, L7 logs are only recorded for wep types and also only with aggregated names. If the protocol is
		// known then only include for TCP.
		if isAgg && (proto == "" || proto == "tcp") {
			l7Source = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "src_namespace", namespace),
				v1.NewGraphSelector(v1.OpEqual, "src_name_aggr", name),
			)
			l7Dest = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "dest_namespace", namespace),
				v1.NewGraphSelector(v1.OpEqual, "dest_name_aggr", name),
			)
		}
	}

	if epType == v1.GraphNodeTypeHosts {
		// Handle hosts separately. We provide an internal aggregation for these types, so when constructing a selector
		// we have do do a rather brutal list of all host endpoints. We can at least skip namespace since hep types
		// are only non-namespaced.
		hosts := s.aggHelper.GetHostNamesFromAggregatedName(name)
		if len(hosts) > maxSelectorItemsPerGroup {
			// Too many individual items. Don't filter on the hosts.
			l3Source = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "source_type", rawType),
			)
			l3Dest = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "dest_type", rawType),
				v1.NewGraphSelector(v1.OpEqual, "dest_name_aggr", name),
			)
		} else if len(hosts) == 1 {
			// Only one host, just use equals.
			l3Source = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "source_type", rawType),
				v1.NewGraphSelector(v1.OpEqual, "source_name_aggr", name),
			)
			l3Dest = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "dest_type", rawType),
				v1.NewGraphSelector(v1.OpEqual, "dest_name_aggr", name),
			)
		} else {
			// Multiple (or no) host names, use "in" operator.  The in operator will not include a zero length
			// comparison (which would be the case if there are no node selectors specified).
			l3Source = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "source_type", rawType),
				v1.NewGraphSelector(v1.OpIn, "source_name_aggr", hosts),
			)
			l3Dest = v1.NewGraphSelector(v1.OpAnd,
				v1.NewGraphSelector(v1.OpEqual, "dest_type", rawType),
				v1.NewGraphSelector(v1.OpIn, "dest_name_aggr", hosts),
			)
		}
	} else if isAgg {
		l3Source = v1.NewGraphSelector(v1.OpAnd,
			v1.NewGraphSelector(v1.OpEqual, "source_type", rawType),
			v1.NewGraphSelector(v1.OpEqual, "source_namespace", namespace),
			v1.NewGraphSelector(v1.OpEqual, "source_name_aggr", name),
		)
		l3Dest = v1.NewGraphSelector(v1.OpAnd,
			v1.NewGraphSelector(v1.OpEqual, "dest_type", rawType),
			v1.NewGraphSelector(v1.OpEqual, "dest_namespace", namespace),
			v1.NewGraphSelector(v1.OpEqual, "dest_name_aggr", name),
		)
	} else {
		l3Source = v1.NewGraphSelector(v1.OpAnd,
			v1.NewGraphSelector(v1.OpEqual, "source_type", rawType),
			v1.NewGraphSelector(v1.OpEqual, "source_namespace", namespace),
			v1.NewGraphSelector(v1.OpEqual, "source_name", name),
		)
		l3Dest = v1.NewGraphSelector(v1.OpAnd,
			v1.NewGraphSelector(v1.OpEqual, "dest_type", rawType),
			v1.NewGraphSelector(v1.OpEqual, "dest_namespace", namespace),
			v1.NewGraphSelector(v1.OpEqual, "dest_name", name),
		)
	}
	if port != 0 {
		l3Dest = v1.NewGraphSelector(v1.OpAnd,
			v1.NewGraphSelector(v1.OpEqual, "dest_port", port),
			l3Dest,
		)
	}
	if proto != "" {
		l3Source = v1.NewGraphSelector(v1.OpAnd,
			v1.NewGraphSelector(v1.OpEqual, "proto", proto),
			l3Source,
		)
		l3Dest = v1.NewGraphSelector(v1.OpAnd,
			v1.NewGraphSelector(v1.OpEqual, "proto", proto),
			l3Dest,
		)
	}

	gsp := SelectorPairs{
		Source: v1.GraphSelectors{},
		Dest:   v1.GraphSelectors{},
	}

	// If a direction has been specified then we only include one side of the flow.
	if dir != DirectionIngress {
		gsp.Source = v1.GraphSelectors{
			L3Flows: l3Source,
			L7Flows: l7Source,
			DNSLogs: dnsSource,
		}
	}
	if dir != DirectionEgress {
		gsp.Dest = v1.GraphSelectors{
			L3Flows: l3Dest,
			L7Flows: l7Dest,
		}
	}

	return gsp
}
