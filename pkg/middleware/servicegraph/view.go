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
	Focus           *ParsedNodes
	Expanded        *ParsedNodes
	FollowedIngress *ParsedNodes
	FollowedEgress  *ParsedNodes
	Layers          *ParsedLayers
}

// ParsedNodes contains details about a set of parsed node IDs in the view. The different aggregation levels are split
// out for easier lookup in the graph constructor.
type ParsedNodes struct {
	Layers        map[string]bool
	Namespaces    map[string]bool
	ServiceGroups map[*ServiceGroup]bool
	Endpoints     map[FlowEndpoint]bool
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
		Endpoints:     make(map[FlowEndpoint]bool),
	}
}

// ParsedLayers contains the details about the parsed layers. The different aggregation levels are split
// out for easier lookup in the graph constructor.
type ParsedLayers struct {
	NamespaceToLayer    map[string]string
	ServiceGroupToLayer map[*ServiceGroup]string
	EndpointToLayer     map[FlowEndpoint]string
}

// newParsedLayers initializes a new ParsedLayers struct.
func newParsedLayers() *ParsedLayers {
	return &ParsedLayers{
		NamespaceToLayer:    make(map[string]string),
		ServiceGroupToLayer: make(map[*ServiceGroup]string),
		EndpointToLayer:     make(map[FlowEndpoint]string),
	}
}

// ParseViewIDs converts the IDs contained in the view to a ParsedViewIDs.
func ParseViewIDs(sgr *v1.ServiceGraphRequest, sgs ServiceGroups) (*ParsedViewIDs, error) {
	// Parse the Focus and Expanded node IDs.
	log.Debug("Parse view data")
	p := &ParsedViewIDs{}
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

func parseNodes(ids []string, sgs ServiceGroups) (pn *ParsedNodes, err error) {
	pn = newParsedNodes()
	for _, id := range ids {
		log.Debugf("Processing ID in view: %s", id)
		if pid, err := ParseGraphNodeID(id); err != nil {
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
				for _, s := range pid.Services {
					if sg := sgs.GetByService(s); sg != nil {
						pn.ServiceGroups[sg] = true
					}
				}
			default:
				// Otherwise assume it's the endpoint we parsed - and in that case we use the service group
				// endpoint key - which may or may not include port and protocol based on what is available.
				if ep := GetServiceGroupFlowEndpointKey(pid.Endpoint); ep != nil {
					pn.Endpoints[*ep] = true
				}
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
			if pid, err := ParseGraphNodeID(id); err != nil {
				return nil, fmt.Errorf("invalid id '%s': %v", id, err)
			} else {
				switch pid.ParsedIDType {
				case v1.GraphNodeTypeNamespace:
					pn.NamespaceToLayer[pid.Endpoint.Namespace] = layer
				case v1.GraphNodeTypeService, v1.GraphNodeTypeServicePort:
					if sg := sgs.GetByService(pid.Service.NamespacedName); sg != nil {
						pn.ServiceGroupToLayer[sg] = layer
					}
				case v1.GraphNodeTypeServiceGroup:
					for _, s := range pid.Services {
						if sg := sgs.GetByService(s); sg != nil {
							pn.ServiceGroupToLayer[sg] = layer
						}
					}
				default:
					// Otherwise assume it's the endpoint we parsed - and in that case we use the service group
					// endpoint key - which may or may not include port and protocol based on what is available.
					if ep := GetServiceGroupFlowEndpointKey(pid.Endpoint); ep != nil {
						pn.EndpointToLayer[*ep] = layer
					}
				}
			}
		}
	}
	return
}
