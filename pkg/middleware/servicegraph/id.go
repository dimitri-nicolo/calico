// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"k8s.io/apimachinery/pkg/types"
)

// This file provides the graph node ID handling. It defines an IDInfo struct that encapsulates all possible data that
// may be parsed from a graph node ID, or that may be used to construct a graph node ID.
//
// ParseGraphNodeID is used to parse a graph node ID and return an IDInfo.
// Create an IDInfo with appropriate data filled in, and use the various helper methods to construct an ID for a node.

const (
	NoPort      int       = 0
	NoDirection Direction = ""
	NoProto               = ""
)

const graphNodeTypeDirection = "dir"

type Direction string

const (
	DirectionIngress Direction = "ingress"
	DirectionEgress  Direction = "egress"
)

// IDInfo is used to construct or parse service graph node string ids.
type IDInfo struct {
	// The type parsed from the ID.
	ParsedIDType v1.GraphNodeType

	// The following is extracted from an ID, or used to construct an ID.
	Endpoint  FlowEndpoint
	Service   ServicePort
	Services  []types.NamespacedName
	Layer     string
	Direction Direction
}

// GetAggrEndpointID returns the aggregated endpoint ID used both internally by the script and externally by the
// service graph.
func (idf *IDInfo) GetAggrEndpointID() v1.GraphNodeID {
	switch idf.Endpoint.Type {
	case v1.GraphNodeTypeWorkload:
		return v1.GraphNodeID(fmt.Sprintf("%s/%s/%s", v1.GraphNodeTypeReplicaSet, idf.Endpoint.Namespace, idf.Endpoint.NameAggr))
	case v1.GraphNodeTypeReplicaSet:
		return v1.GraphNodeID(fmt.Sprintf("%s/%s/%s", v1.GraphNodeTypeReplicaSet, idf.Endpoint.Namespace, idf.Endpoint.NameAggr))
	case v1.GraphNodeTypeNetwork, v1.GraphNodeTypeHostEndpoint, v1.GraphNodeTypeNetworkSet:
		var id string
		if idf.Endpoint.Namespace == "" {
			id = fmt.Sprintf("%s/%s", idf.Endpoint.Type, idf.Endpoint.NameAggr)
		} else {
			id = fmt.Sprintf("%s/%s/%s", idf.Endpoint.Type, idf.Endpoint.Namespace, idf.Endpoint.NameAggr)
		}
		// If there is a service group then include the service group, otherwise if there is a Direction include that
		// (this effectively separates out sources and sinks.
		if svcGpId := idf.GetServiceGroupID(); svcGpId != "" {
			return v1.GraphNodeID(fmt.Sprintf("%s;%s", id, svcGpId))
		} else if dirId := idf.getDirectionID(); dirId != "" {
			return v1.GraphNodeID(fmt.Sprintf("%s;%s", id, dirId))
		}
		return v1.GraphNodeID(id)
	}
	return ""
}

// GetAggrEndpointType returns the aggregated endpoint type. This may be different from the Type in the structure
// if the endpoint is not aggregated. In particular if the endpoint is actually a pod (wep) then the aggregated type
// would be a replica set.
func (idf *IDInfo) GetAggrEndpointType() v1.GraphNodeType {
	return ConvertEndpointTypeToAggrEndpointType(idf.Endpoint.Type)
}

func ConvertEndpointTypeToAggrEndpointType(t v1.GraphNodeType) v1.GraphNodeType {
	if t == v1.GraphNodeTypeWorkload {
		return v1.GraphNodeTypeReplicaSet
	}
	return t
}

// GetEndpointID returns the ID of the non-aggregated endpoint. If the endpoint only has aggregated name data then this
// will return an empty string.
func (idf *IDInfo) GetEndpointID() v1.GraphNodeID {
	if idf.Endpoint.Type == v1.GraphNodeTypeWorkload {
		return v1.GraphNodeID(fmt.Sprintf("%s/%s/%s/%s", v1.GraphNodeTypeWorkload, idf.Endpoint.Namespace, idf.Endpoint.Name, idf.Endpoint.NameAggr))
	}
	return ""
}

// GetEndpointPortID returns the ID of the endpoint Port. This contains the parent endpoint ID embedded in it, or the
// aggregated endpoint ID if only the aggregated endpoint data is available. This returns an empty string if the
// node aggregated out endpoint information.
func (idf *IDInfo) GetEndpointPortID() v1.GraphNodeID {
	if idf.Endpoint.Port == 0 {
		return ""
	}
	epID := idf.GetEndpointID()
	if epID == "" {
		return idf.GetAggrEndpointPortID()
	}
	return v1.GraphNodeID(fmt.Sprintf("%s/%s/%d;%s", v1.GraphNodeTypePort, idf.Service.Proto, idf.Endpoint.Port, epID))
}

// GetAggrEndpointPortID returns the ID of the endpoint Port. This contains the parent aggregataed endpoint ID embedded
// in it. This returns an empty string if the node aggregated out endpoint information.
func (idf *IDInfo) GetAggrEndpointPortID() v1.GraphNodeID {
	if idf.Endpoint.Port == 0 {
		return ""
	}
	epID := idf.GetAggrEndpointID()
	if epID == "" {
		return ""
	}
	return v1.GraphNodeID(fmt.Sprintf("%s/%s/%d;%s", v1.GraphNodeTypePort, idf.Service.Proto, idf.Endpoint.Port, epID))
}

// GetServiceID returns the destination service ID of the service contained in this node.
func (idf *IDInfo) GetServiceID() v1.GraphNodeID {
	if idf.Service.Name == "" {
		return ""
	}
	return v1.GraphNodeID(idf.getServiceID(idf.Service.Namespace, idf.Service.Name))
}

// getServiceID returns the destination service ID of the service contained in this node.
func (idf *IDInfo) getServiceID(namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", v1.GraphNodeTypeService, namespace, name)
}

// GetServiceGroupID returns the internal service group ID for this node. If the internal service group index
// is not known and is not found by looking up the service name then this returns an empty string.
func (idf *IDInfo) GetServiceGroupID() v1.GraphNodeID {
	if len(idf.Services) == 0 {
		return ""
	}
	serviceIds := make([]string, len(idf.Services))
	for i, s := range idf.Services {
		serviceIds[i] = idf.getServiceID(s.Namespace, s.Name)
	}
	return v1.GraphNodeID(fmt.Sprintf("%s;%s", v1.GraphNodeTypeServiceGroup, strings.Join(serviceIds, ";")))
}

// GetServicePortID returns the ID of the service Port. This contains the parent service ID embedded in it. This returns
// an empty string if the service Port is not present.
func (idf *IDInfo) GetServicePortID() v1.GraphNodeID {
	if id := idf.GetServiceID(); id != "" {
		return v1.GraphNodeID(fmt.Sprintf("%s/%s/%s;%s", v1.GraphNodeTypeServicePort, idf.Service.Proto, idf.Service.Port, id))
	}
	return ""
}

// GetLayerID returns the ID of the layer that this endpoint is part of. This returns an empty string if the node
// is not in a layer.
func (idf *IDInfo) GetLayerID() v1.GraphNodeID {
	if idf.Layer == "" {
		return ""
	}
	return v1.GraphNodeID(fmt.Sprintf("%s/%s", v1.GraphNodeTypeLayer, idf.Layer))
}

// GetNamespaceID returns the ID of the Namespace that this endpoint is part of. This returns an empty string if the
// node is not namespaced.
func (idf *IDInfo) GetNamespaceID() v1.GraphNodeID {
	if idf.Endpoint.Namespace == "" {
		return ""
	}
	return v1.GraphNodeID(fmt.Sprintf("%s/%s", v1.GraphNodeTypeNamespace, idf.Endpoint.Namespace))
}

// getDirectionID() is an additional ID used to separate out ingress and egress.
func (idf *IDInfo) getDirectionID() v1.GraphNodeID {
	if idf.Direction == "" {
		return ""
	}
	return v1.GraphNodeID(fmt.Sprintf("%s/%s", graphNodeTypeDirection, idf.Direction))
}

type idp byte

const (
	idpType idp = iota
	idpLayer
	idpNamespace
	idpName
	idpNameAggr
	idpProtocol
	idpPort
	idpServiceNamespace
	idpServiceName
	idpServicePort
	idpServiceProtocol
	idpDirection
)

var (
	// For each type, this provides the field names of each segment of the ID. For some types there may be multiple
	// ways to unwrap the ID based on the number of segments in the ID.
	idMappings = map[v1.GraphNodeType][][]idp{
		v1.GraphNodeTypeLayer:        {{idpType, idpLayer}},
		v1.GraphNodeTypeNamespace:    {{idpType, idpNamespace}},
		v1.GraphNodeTypeServiceGroup: {{idpType}},
		v1.GraphNodeTypeReplicaSet:   {{idpType, idpNamespace, idpNameAggr}},
		v1.GraphNodeTypeHostEndpoint: {{idpType, idpNameAggr}},
		v1.GraphNodeTypeNetwork:      {{idpType, idpNameAggr}},
		v1.GraphNodeTypeNetworkSet:   {{idpType, idpNameAggr}, {idpType, idpNamespace, idpNameAggr}},
		v1.GraphNodeTypeWorkload:     {{idpType, idpNamespace, idpName, idpNameAggr}},
		v1.GraphNodeTypePort:         {{idpType, idpProtocol, idpPort}},
		v1.GraphNodeTypeService:      {{idpType, idpServiceNamespace, idpServiceName}},
		v1.GraphNodeTypeServicePort:  {{idpType, idpServiceProtocol, idpServicePort}},
		graphNodeTypeDirection:       {{idpType, idpDirection}},
	}

	// An ID may contain parent information to fully qualify it. This specifies which parent types are valid for a
	// specific type.
	allowedParentTypes = map[v1.GraphNodeType][]v1.GraphNodeType{
		v1.GraphNodeTypePort: {
			v1.GraphNodeTypeReplicaSet, v1.GraphNodeTypeWorkload, v1.GraphNodeTypeHostEndpoint,
			v1.GraphNodeTypeNetwork, v1.GraphNodeTypeNetworkSet,
		},
		v1.GraphNodeTypeNetwork:      {v1.GraphNodeTypeServiceGroup, graphNodeTypeDirection},
		v1.GraphNodeTypeNetworkSet:   {v1.GraphNodeTypeServiceGroup, graphNodeTypeDirection},
		v1.GraphNodeTypeHostEndpoint: {v1.GraphNodeTypeServiceGroup, graphNodeTypeDirection},
		v1.GraphNodeTypeServicePort:  {v1.GraphNodeTypeService},
		v1.GraphNodeTypeServiceGroup: {v1.GraphNodeTypeService},
		v1.GraphNodeTypeService:      {v1.GraphNodeTypeService},
	}

	// All segments should adhere to this simple regex. Further restrictions may be imposed on a field by field basis.
	valueRegex             = regexp.MustCompile("^[0-9a-zA-Z_.-]+$")
	valueAllowedEmptyRegex = regexp.MustCompile("^[0-9a-zA-Z_.-]*$")
	firstSplitRegex        = regexp.MustCompile("[;/]")
)

// ParseGraphNodeID parses an external node ID and returns the data in an ID.
func ParseGraphNodeID(id v1.GraphNodeID) (*IDInfo, error) {
	parts := firstSplitRegex.Split(string(id), 2)

	// Names are hierarchical in nature, with components separated by semicolons: sub-component -> parent component.
	// Update the type as we go along.
	idf := &IDInfo{
		ParsedIDType: v1.GraphNodeType(parts[0]),
	}
	var previousType v1.GraphNodeType
	var isServiceGroup bool
	for _, component := range strings.Split(string(id), ";") {
		parts := strings.Split(component, "/")
		thisType := v1.GraphNodeType(parts[0])

		// Check the type one of the allowed parent types.
		if len(previousType) != 0 {
			var allowed bool
			for _, allowedParentType := range allowedParentTypes[previousType] {
				if allowedParentType == thisType {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, fmt.Errorf("invalid format for ID: %s", id)
			}
		}

		if thisType == v1.GraphNodeTypeServiceGroup {
			isServiceGroup = true
		}

		// If the current type is an endpoint type then update the endpoint info. Each ID should have at most one
		// endpoint specified.
		switch thisType {
		case v1.GraphNodeTypeReplicaSet,
			v1.GraphNodeTypeHostEndpoint,
			v1.GraphNodeTypeNetwork,
			v1.GraphNodeTypeNetworkSet,
			v1.GraphNodeTypeWorkload:
			idf.Endpoint.Type = thisType
		}

		// Locate the mapping for the endpoint type and copy the values into the response.
		var foundMapping bool
		for _, mappings := range idMappings[thisType] {
			if len(mappings) != len(parts) {
				continue
			}
			foundMapping = true
			for idx, field := range mappings {
				// Check the segment syntax. Only the service Port is allowed to be empty.
				switch field {
				case idpServicePort:
					if !valueAllowedEmptyRegex.MatchString(parts[idx]) {
						return nil, fmt.Errorf("invalid format of graph node ID %s: unexpected empty segment", id)
					}
				default:
					if !valueRegex.MatchString(parts[idx]) {
						return nil, fmt.Errorf("invalid format of graph node ID %s: badly formatted segment", id)
					}
				}

				switch field {
				case idpType:
					// Already extracted the type.
				case idpNamespace:
					idf.Endpoint.Namespace = parts[idx]
				case idpName:
					idf.Endpoint.Name = parts[idx]
				case idpNameAggr:
					idf.Endpoint.NameAggr = parts[idx]
				case idpLayer:
					idf.Layer = parts[idx]
				case idpProtocol:
					idf.Endpoint.Proto = parts[idx]
				case idpServiceProtocol:
					idf.Service.Proto = parts[idx]
				case idpPort:
					val, err := strconv.Atoi(parts[idx])
					if err != nil {
						return nil, fmt.Errorf("invalid format of graph node ID %s: Port is not a number: %v", id, err)
					}
					idf.Endpoint.Port = val
				case idpServiceNamespace:
					idf.Service.Namespace = parts[idx]
				case idpServiceName:
					idf.Service.Name = parts[idx]
				case idpServicePort:
					idf.Service.Port = parts[idx]
				case idpDirection:
					idf.Direction = Direction(parts[idx])
				default:
					return nil, fmt.Errorf("invalid format of graph node ID %s: unexpected node type", id)
				}
			}
			break
		}

		// If we are parsing a service group and the last segment was a service then append to the set of services
		// in the group.
		if isServiceGroup && thisType == v1.GraphNodeTypeService {
			idf.Services = append(idf.Services, idf.Service.NamespacedName)
			idf.Service = ServicePort{}
		}

		if !foundMapping {
			return nil, fmt.Errorf("invalid format of graph node ID %s", id)
		}

		previousType = thisType
	}

	return idf, nil
}
