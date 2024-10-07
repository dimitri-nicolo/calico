// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/lma/pkg/httputils"

	v1 "github.com/projectcalico/calico/ui-apis/pkg/apis/v1"
)

// This file is used to process the view data supplied in the service graph request and convert it to sets of parsed
// data in a format that makes it more useful for the graph constructor and other internal components.

// ParsedView contains the view specified in the service graph request, parsed into an internally useful format.
type ParsedView struct {
	NodeViewData              map[v1.GraphNodeID]NodeViewData
	Layers                    *ParsedLayers
	FollowConnectionDirection bool
	SplitIngressEgress        bool
	ExpandPorts               bool
	EmptyFocus                bool
}

type NodeViewData struct {
	InFocus         bool
	Expanded        bool
	FollowedIngress bool
	FollowedEgress  bool
}

func (d NodeViewData) Combine(d2 NodeViewData) NodeViewData {
	return NodeViewData{
		InFocus:         d.InFocus || d2.InFocus,
		Expanded:        d.Expanded || d2.Expanded,
		FollowedIngress: d.FollowedIngress || d2.FollowedIngress,
		FollowedEgress:  d.FollowedEgress || d2.FollowedEgress,
	}
}

// ParsedLayers contains the details about the parsed layers. The different aggregation levels are split
// out for easier lookup in the graph constructor.
type ParsedLayers struct {
	NamespaceToLayer    map[string]string
	ServiceGroupToLayer map[*ServiceGroup]string
	EndpointToLayer     map[v1.GraphNodeID]string

	// Store layer contents - we use this for constructing selector strings.
	LayerToNamespaces    map[string][]string
	LayerToServiceGroups map[string][]*ServiceGroup
	LayerToEndpoints     map[string][]FlowEndpoint
}

// newParsedLayers initializes a new ParsedLayers struct.
func newParsedLayers() *ParsedLayers {
	return &ParsedLayers{
		NamespaceToLayer:     make(map[string]string),
		ServiceGroupToLayer:  make(map[*ServiceGroup]string),
		EndpointToLayer:      make(map[v1.GraphNodeID]string),
		LayerToNamespaces:    make(map[string][]string),
		LayerToServiceGroups: make(map[string][]*ServiceGroup),
		LayerToEndpoints:     make(map[string][]FlowEndpoint),
	}
}

// ParseViewIDs converts the IDs contained in the view to a ParsedView.
func ParseViewIDs(rd *RequestData, sgs ServiceGroups) (*ParsedView, error) {
	// Parse the Focus and Expanded node IDs.
	log.Debug("Parse view data")
	p := &ParsedView{
		NodeViewData:              make(map[v1.GraphNodeID]NodeViewData),
		FollowConnectionDirection: rd.ServiceGraphRequest.SelectedView.FollowConnectionDirection,
		ExpandPorts:               rd.ServiceGraphRequest.SelectedView.ExpandPorts,
		SplitIngressEgress:        rd.ServiceGraphRequest.SelectedView.SplitIngressEgress,
		EmptyFocus:                len(rd.ServiceGraphRequest.SelectedView.Focus) == 0,
	}
	if ids, err := parseNodes(rd.ServiceGraphRequest.SelectedView.Expanded, sgs, p.SplitIngressEgress); err != nil {
		return nil, httputils.NewHttpStatusErrorBadRequest("Request body contains an invalid expanded node: "+err.Error(), err)
	} else {
		for _, id := range ids {
			p.NodeViewData[id] = p.NodeViewData[id].Combine(NodeViewData{Expanded: true})
		}
	}
	if ids, err := parseNodes(rd.ServiceGraphRequest.SelectedView.FollowedIngress, sgs, p.SplitIngressEgress); err != nil {
		return nil, httputils.NewHttpStatusErrorBadRequest("Request body contains an invalid followed_ingress node: "+err.Error(), err)
	} else {
		for _, id := range ids {
			p.NodeViewData[id] = p.NodeViewData[id].Combine(NodeViewData{FollowedIngress: true})
		}
	}
	if ids, err := parseNodes(rd.ServiceGraphRequest.SelectedView.FollowedEgress, sgs, p.SplitIngressEgress); err != nil {
		return nil, httputils.NewHttpStatusErrorBadRequest("Request body contains an invalid followed_egress node: "+err.Error(), err)
	} else {
		for _, id := range ids {
			p.NodeViewData[id] = p.NodeViewData[id].Combine(NodeViewData{FollowedEgress: true})
		}
	}

	if layers, err := parseLayers(rd.ServiceGraphRequest.SelectedView.Layers, sgs, p.SplitIngressEgress); err != nil {
		return nil, httputils.NewHttpStatusErrorBadRequest("Request body contains an invalid layer node: "+err.Error(), err)
	} else {
		p.Layers = layers
	}

	// Now parse the focus nodes. We need to ensure that the set of expanded nodes contains all of the required nodes
	// to view the focus nodes.
	if ids, err := parseNodes(rd.ServiceGraphRequest.SelectedView.Focus, sgs, p.SplitIngressEgress); err != nil {
		return nil, httputils.NewHttpStatusErrorBadRequest("Request body contains an invalid focus node: "+err.Error(), err)
	} else {
		for _, id := range ids {
			p.NodeViewData[id] = p.NodeViewData[id].Combine(NodeViewData{InFocus: true})

			expandedNodes := getExpandedNodesForNode(id, p.Layers, sgs)
			for _, expandedNodeId := range expandedNodes {
				p.NodeViewData[expandedNodeId] = p.NodeViewData[expandedNodeId].Combine(NodeViewData{Expanded: true})
			}
		}
	}

	return p, nil
}

func parseNodes(ids []v1.GraphNodeID, sgs ServiceGroups, splitIngressEgress bool) ([]v1.GraphNodeID, error) {
	var pn []v1.GraphNodeID
	for _, id := range ids {
		log.Debugf("Processing ID in view: %s", id)
		if pids, err := GetNormalizedIDs(id, sgs, splitIngressEgress); err != nil {
			return nil, err
		} else {
			pn = append(pn, pids...)
		}
	}
	return pn, nil
}

func parseLayers(layers []v1.Layer, sgs ServiceGroups, splitIngressEgress bool) (pn *ParsedLayers, err error) {
	pn = newParsedLayers()
	for _, layer := range layers {
		for _, id := range layer.Nodes {
			log.Debugf("Processing ID in view: %s", id)
			ids, err := GetNormalizedIDs(id, sgs, splitIngressEgress)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				if pid, err := ParseGraphNodeID(id, sgs); err != nil {
					return nil, err
				} else {
					switch pid.ParsedIDType {
					case v1.GraphNodeTypeNamespace:
						if _, ok := pn.NamespaceToLayer[pid.Endpoint.Namespace]; !ok {
							pn.NamespaceToLayer[pid.Endpoint.Namespace] = layer.Name
							pn.LayerToNamespaces[layer.Name] = append(pn.LayerToNamespaces[layer.Name], pid.Endpoint.Namespace)
						}
					default:
						// Otherwise assume it's a service group or endpoint we parsed.  Note that we always include the
						// group of related endpoints in the layer so that we are not trying to pick apart endpoints in a
						// service, or endpoints in an aggregated endpoint group. This is important because expanding a
						// layer could end up with some very odd layouts.
						if pid.ServiceGroup != nil {
							if _, ok := pn.ServiceGroupToLayer[pid.ServiceGroup]; !ok {
								pn.ServiceGroupToLayer[pid.ServiceGroup] = layer.Name
								pn.LayerToServiceGroups[layer.Name] = append(pn.LayerToServiceGroups[layer.Name], pid.ServiceGroup)
							}
						} else if id := pid.GetAggrEndpointID(); id != "" {
							// There is no service group associated with this endpoint - include the aggregated endpoint in
							// the layer. We may need
							if _, ok := pn.EndpointToLayer[id]; !ok {
								pn.EndpointToLayer[id] = layer.Name
								pn.LayerToEndpoints[layer.Name] = append(pn.LayerToEndpoints[layer.Name], pid.Endpoint)
							}
						}
					}
				}
			}
		}
	}
	return
}

// getExpandedNodesForNode returns the set of expanded nodes required to view the node specified by `id`.
func getExpandedNodesForNode(id v1.GraphNodeID, layers *ParsedLayers, sgs ServiceGroups) []v1.GraphNodeID {
	var nodes []v1.GraphNodeID
	pid, err := ParseGraphNodeID(id, sgs)
	if err != nil {
		// We've already parsed the node so should never hit an error.
		return nodes
	}

	if id := pid.GetEndpointID(); id != "" {
		// This is an endpoint.  If this is part of a layer then include the layer in the expansion.
		if layer := layers.EndpointToLayer[id]; layer != "" {
			pid.Layer = layer
			nodes = append(nodes, pid.GetLayerID())
			return nodes
		}

		// The endpoint is in view, so make sure the aggregated endpoint is expanded.
		if id := pid.GetAggrEndpointID(); id != "" {
			nodes = append(nodes, id)
		}
	}

	if id := pid.GetAggrEndpointID(); id != "" {
		// This is an aggregated endpoint (or contained within one).  If this is part of a layer then include the layer
		// in the expansion.
		if layer := layers.EndpointToLayer[id]; layer != "" {
			pid.Layer = layer
			nodes = append(nodes, pid.GetLayerID())
			return nodes
		}

		// The aggregated endpoint is in view, so make sure the service group or namespace (whichever is appropriate)
		// is expanded.
		if id := pid.GetServiceGroupID(); id != "" {
			nodes = append(nodes, id)
		} else if ns := pid.GetNamespaceID(); ns != "" {
			nodes = append(nodes, id)
		}
	}

	if pid.ServiceGroup != nil {
		// This is a service group (or contained within one).  If this is part of a layer then include the layer in the
		// expansion.
		if layer := layers.ServiceGroupToLayer[pid.ServiceGroup]; layer != "" {
			pid.Layer = layer
			nodes = append(nodes, pid.GetLayerID())
			return nodes
		}

		// The service group is in view so make sure the namespace is expanded.
		if id := pid.GetNamespaceID(); id != "" {
			nodes = append(nodes, id)
		}
	}

	if ns := pid.GetEffectiveNamespace(); ns != "" {
		// This is a namespace (or contained within one).  If this is part of a layer then include the layer in the
		// expansion.
		if layer := layers.NamespaceToLayer[ns]; layer != "" {
			pid.Layer = layer
			nodes = append(nodes, pid.GetLayerID())
			return nodes
		}
	}

	return nodes
}
