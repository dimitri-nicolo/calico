package servicegraph

import (
	"fmt"
	"strings"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"k8s.io/apimachinery/pkg/types"
)

// The following set of functions is used to construct node selectors.  These selectors can be used to query
// additional data for a node. The edge selectors is constructed from the node selectors in the graphconstructor.
//
// Selector format is similar to the kibana selector, e.g.
//   source_namespace == "namespace1 OR (dest_type == "wep" AND dest_namespace == "namespace2")

// Helper structs/methods for constructing the string for single "X == Y" selector and for ANDed sets of selectors.
// Note that ORed groups of selectors is handled in the GraphSelectors struct since that can handle the ORing of
// the full set in one operation.
type keyEqValue struct {
	key string
	val interface{}
}

func (kv keyEqValue) String() string {
	if s, ok := kv.val.(string); ok {
		return fmt.Sprintf("%s == \"%s\"", kv.key, s)
	}
	return fmt.Sprintf("%s == %v", kv.key, kv.val)
}

// keyEqValueOpAnd is used to collate ANDed groups of "X == Y" selectors.  Note that ORed groups is handled by the
// GraphSelectors methods.
type keyEqValueOpAnd []keyEqValue

func (kvs keyEqValueOpAnd) String() string {
	parts := make([]string, len(kvs))
	for i := range kvs {
		parts[i] = kvs[i].String()
	}
	if len(parts) == 1 {
		return parts[0]
	}

	// Enclose multiple and-ed key==val in parens.
	return "(" + strings.Join(parts, v1.OpAndWithSpaces) + ")"
}

//
func GetLayerNodeSelectors(layer string, view *ParsedViewIDs) v1.GraphSelectors {
	gs := v1.GraphSelectors{}
	for _, n := range view.Layers.LayerToNamespaces[layer] {
		gs = gs.Or(GetNamespaceNodeSelectors(n))
	}
	for _, sg := range view.Layers.LayerToServiceGroups[layer] {
		gs = gs.Or(GetServiceGroupNodeSelectors(sg))
	}
	for _, ep := range view.Layers.LayerToEndpoints[layer] {
		_, isAgg := mapGraphNodeTypeToRawType(ep.Type)
		if isAgg {
			gs = gs.Or(GetEndpointNodeSelectors(ep.Type, ep.Namespace, ep.NameAggr, ep.Proto, ep.Port, NoDirection))
		} else {
			gs = gs.Or(GetEndpointNodeSelectors(ep.Type, ep.Namespace, ep.Name, ep.Proto, ep.Port, NoDirection))
		}
	}
	return gs
}

func GetNamespaceNodeSelectors(namespace string) v1.GraphSelectors {
	return v1.GraphSelectors{
		L3Flows: v1.GraphSelector{
			Source: keyEqValue{"source_namespace", namespace}.String(),
			Dest:   keyEqValue{"dest_namespace", namespace}.String(),
		},
		L7Flows: v1.GraphSelector{
			Source: keyEqValue{"src_namespace", namespace}.String(),
			Dest:   keyEqValue{"dest_namespace", namespace}.String(),
		},
		DNSLogs: v1.GraphSelector{
			Source: keyEqValue{"client_namespace", namespace}.String(),
		},
	}
}

func GetServiceNodeSelectors(svc types.NamespacedName) v1.GraphSelectors {
	return v1.GraphSelectors{
		L3Flows: v1.GraphSelector{
			Dest: keyEqValueOpAnd{{"dest_service_namespace", svc.Namespace}, {"dest_service_name", svc.Name}}.String(),
		},
		L7Flows: v1.GraphSelector{
			Dest: keyEqValueOpAnd{{"dest_service_namespace", svc.Namespace}, {"dest_service_name", svc.Name}}.String(),
		},
	}
}

func GetServicePortNodeSelectors(sp ServicePort) v1.GraphSelectors {
	l3Parts := keyEqValueOpAnd{{"proto", sp.Proto}, {"dest_service_namespace", sp.Namespace}, {"dest_service_name", sp.Name}}
	if sp.Port != "" {
		l3Parts = append(l3Parts, keyEqValue{"dest_service_port", sp.Port})
	}

	var l7Parts keyEqValueOpAnd
	if sp.Proto == "tcp" {
		l7Parts = keyEqValueOpAnd{{"dest_service_namespace", sp.Namespace}, {"dest_service_name", sp.Name}}
	}

	return v1.GraphSelectors{
		// L3 source selector is calculated in graphconstructor by ORing together all of the ingress sources to the
		// service.  We need to do this because flows arae recorded at source and dest and may therefore not have
		// service information available.
		L3Flows: v1.GraphSelector{
			Dest: l3Parts.String(),
		},
		// L3 service selector is the same for source and dest.
		L7Flows: v1.GraphSelector{
			Source: l7Parts.String(),
			Dest:   l7Parts.String(),
		},
	}
}

func GetServiceGroupNodeSelectors(sg *ServiceGroup) v1.GraphSelectors {
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

	gs := v1.GraphSelectors{}
	for svc := range allSvcs {
		gs = gs.Or(GetServiceNodeSelectors(svc))
	}
	for ep := range allEps {
		_, isAgg := mapGraphNodeTypeToRawType(ep.Type)
		if isAgg {
			gs = gs.Or(GetEndpointNodeSelectors(ep.Type, ep.Namespace, ep.NameAggr, ep.Proto, ep.Port, NoDirection))
		} else {
			gs = gs.Or(GetEndpointNodeSelectors(ep.Type, ep.Namespace, ep.Name, ep.Proto, ep.Port, NoDirection))
		}
	}
	return gs
}

func GetEndpointNodeSelectors(epType v1.GraphNodeType, namespace, name, proto string, port int, dir Direction) v1.GraphSelectors {
	rawType, isAgg := mapGraphNodeTypeToRawType(epType)
	namespace = blankToSingleDash(namespace)

	var l3, l7, dns v1.GraphSelector
	if rawType == "wep" {
		// DNS logs are only recorded for wep types.
		if isAgg {
			dns.Source = keyEqValueOpAnd{{"client_namespace", namespace}, {"client_name_aggr", name}}.String()
		} else {
			dns.Source = keyEqValueOpAnd{{"client_namespace", namespace}, {"client_name", name}}.String()
		}

		// Similarly, L7 logs are only recorded for wep types and also only with aggregated names. If the protocol is
		// known then only include for TCP.
		if isAgg && (proto == "" || proto == "tcp") {
			l7 = v1.GraphSelector{
				Source: keyEqValueOpAnd{{"src_namespace", namespace}, {"src_name_aggr", name}}.String(),
				Dest:   keyEqValueOpAnd{{"dest_namespace", namespace}, {"dest_name_aggr", name}}.String(),
			}
		}
	}

	var l3SrcParts, l3DstParts keyEqValueOpAnd
	if isAgg {
		l3SrcParts = keyEqValueOpAnd{{"source_type", rawType}, {"source_namespace", namespace}, {"source_name_aggr", name}}
		l3DstParts = keyEqValueOpAnd{{"dest_type", rawType}, {"dest_namespace", namespace}, {"dest_name_aggr", name}}
	} else {
		l3SrcParts = keyEqValueOpAnd{{"source_type", rawType}, {"source_namespace", namespace}, {"source_name", name}}
		l3DstParts = keyEqValueOpAnd{{"dest_type", rawType}, {"dest_namespace", namespace}, {"dest_name", name}}
	}
	if port != 0 {
		l3DstParts = append(l3DstParts, keyEqValue{"dest_port", port})
	}
	if proto != "" {
		l3SrcParts = append(l3SrcParts, keyEqValue{"proto", proto})
		l3DstParts = append(l3DstParts, keyEqValue{"proto", proto})
	}
	l3 = v1.GraphSelector{
		Source: l3SrcParts.String(),
		Dest:   l3DstParts.String(),
	}

	// If a direction has been specified, blank out the source or dest values appropriately.  It's easier to do this
	// once here rather than in each of the branches above.
	if dir == DirectionIngress {
		l3.Source = ""
		l7.Source = ""
		dns.Source = ""
	} else if dir == DirectionEgress {
		l3.Dest = ""
		l7.Dest = ""
		dns.Dest = ""
	}

	return v1.GraphSelectors{
		L3Flows: l3,
		L7Flows: l7,
		DNSLogs: dns,
	}
}
