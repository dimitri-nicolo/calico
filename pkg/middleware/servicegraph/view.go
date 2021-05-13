// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

// This file is used to process the view data supplied in the service graph request and convert it to sets of parsed
// data in a format that makes it more useful for the graph constructor and other internal components.

// ParsedViewIDs contains the view specified in the service graph request, parsed into an internally useful format.
type ParsedViewIDs struct {
	Focus                     *ParsedNodes
	Expanded                  *ParsedNodes
	FollowedIngress           *ParsedNodes
	FollowedEgress            *ParsedNodes
	Layers                    *ParsedLayers
	FollowConnectionDirection bool
	SplitIngressEgress        bool
}

// ParsedNodes contains details about a set of parsed node IDs in the view. The different aggregation levels are split
// out for easier lookup in the graph constructor.
type ParsedNodes struct {
	Layers        map[string]bool
	Namespaces    map[string]bool
	ServiceGroups map[*ServiceGroup]bool
	Endpoints     map[v1.GraphNodeID]bool
}

// isEmpty returns true if there are no entries in the data.
func (pn *ParsedNodes) isEmpty() bool {
	return len(pn.Layers) == 0 &&
		len(pn.Namespaces) == 0 &&
		len(pn.ServiceGroups) == 0 &&
		len(pn.Endpoints) == 0
}

// newParsedNodes initializes a new ParsedNode struct.
func newParsedNodes() *ParsedNodes {
	return &ParsedNodes{
		Layers:        make(map[string]bool),
		Namespaces:    make(map[string]bool),
		ServiceGroups: make(map[*ServiceGroup]bool),
		Endpoints:     make(map[v1.GraphNodeID]bool),
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
		NamespaceToLayer:    make(map[string]string),
		ServiceGroupToLayer: make(map[*ServiceGroup]string),
		EndpointToLayer:     make(map[v1.GraphNodeID]string),
	}
}

// ParseViewIDs converts the IDs contained in the view to a ParsedViewIDs.
func ParseViewIDs(sgr *v1.ServiceGraphRequest, sgs ServiceGroups) (*ParsedViewIDs, error) {
	// Parse the Focus and Expanded node IDs.
	log.Debug("Parse view data")
	p := &ParsedViewIDs{
		FollowConnectionDirection: sgr.SelectedView.FollowConnectionDirection,
		SplitIngressEgress:        sgr.SelectedView.SplitIngressEgress,
	}
	var err error
	if p.Focus, err = parseNodes(sgr.SelectedView.Focus, sgs); err != nil {
		return nil, err
	} else if p.Expanded, err = parseNodes(sgr.SelectedView.Expanded, sgs); err != nil {
		return nil, err
	} else if p.FollowedEgress, err = parseNodes(sgr.SelectedView.FollowedEgress, sgs); err != nil {
		return nil, err
	} else if p.FollowedIngress, err = parseNodes(sgr.SelectedView.FollowedIngress, sgs); err != nil {
		return nil, err
	} else if p.Layers, err = parseLayers(sgr.SelectedView.Layers, sgs); err != nil {
		return nil, err
	}

	return p, nil
}

func parseNodes(ids []v1.GraphNodeID, sgs ServiceGroups) (pn *ParsedNodes, err error) {
	pn = newParsedNodes()
	for _, id := range ids {
		log.Debugf("Processing ID in view: %s", id)
		if pid, err := ParseGraphNodeID(id, sgs); err != nil {
			return nil, fmt.Errorf("invalid id '%s': %v", id, err)
		} else {
			switch pid.ParsedIDType {
			case v1.GraphNodeTypeLayer:
				pn.Layers[pid.Layer] = true
			case v1.GraphNodeTypeNamespace:
				pn.Namespaces[pid.Endpoint.Namespace] = true
			case v1.GraphNodeTypeService, v1.GraphNodeTypeServicePort:
				if sg := sgs.GetByService(pid.Service.NamespacedName); sg != nil {
					pn.ServiceGroups[sg] = true
				}
			case v1.GraphNodeTypeServiceGroup:
				pn.ServiceGroups[pid.ServiceGroup] = true
			default:
				pn.Endpoints[pid.GetNormalizedID()] = true
			}
		}
	}
	return
}

func parseLayers(layers v1.Layers, sgs ServiceGroups) (pn *ParsedLayers, err error) {
	pn = newParsedLayers()
	for layer, ids := range layers {
		for _, id := range ids {
			log.Debugf("Processing ID in view: %s", id)
			if pid, err := ParseGraphNodeID(id, sgs); err != nil {
				return nil, fmt.Errorf("invalid id '%s': %v", id, err)
			} else {
				switch pid.ParsedIDType {
				case v1.GraphNodeTypeNamespace:
					if _, ok := pn.NamespaceToLayer[pid.Endpoint.Namespace]; !ok {
						pn.NamespaceToLayer[pid.Endpoint.Namespace] = layer
						pn.LayerToNamespaces[layer] = append(pn.LayerToNamespaces[layer], pid.Endpoint.Namespace)
					}
				case v1.GraphNodeTypeService, v1.GraphNodeTypeServicePort:
					if sg := sgs.GetByService(pid.Service.NamespacedName); sg != nil {
						if _, ok := pn.ServiceGroupToLayer[sg]; !ok {
							pn.ServiceGroupToLayer[sg] = layer
							pn.LayerToServiceGroups[layer] = append(pn.LayerToServiceGroups[layer], sg)
						}
					}
				case v1.GraphNodeTypeServiceGroup:
					if _, ok := pn.ServiceGroupToLayer[pid.ServiceGroup]; !ok {
						pn.ServiceGroupToLayer[pid.ServiceGroup] = layer
						pn.LayerToServiceGroups[layer] = append(pn.LayerToServiceGroups[layer], pid.ServiceGroup)
					}
				default:
					// Otherwise assume it's the endpoint we parsed. In this case we also need to include the service
					// group to disambiguate.
					id := pid.GetNormalizedID()
					if _, ok := pn.EndpointToLayer[id]; !ok {
						pn.EndpointToLayer[id] = layer
						pn.LayerToEndpoints[layer] = append(pn.LayerToEndpoints[layer], pid.Endpoint)
					}
				}
			}
		}
	}
	return
}
