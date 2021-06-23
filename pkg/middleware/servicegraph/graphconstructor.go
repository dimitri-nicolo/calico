// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	log "github.com/sirupsen/logrus"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

// This file provides the final graph construction from a set of correlated (time-series) flows and the parsed view
// IDs.
//
// See v1.GraphView for details on aggregation, and which nodes will be included in the graph.

// GetServiceGraphResponse calculates the service graph from the flow data and parsed view ids.
func GetServiceGraphResponse(f *ServiceGraphData, v *ParsedView) (*v1.ServiceGraphResponse, error) {
	sgr := &v1.ServiceGraphResponse{
		// Response should include the time range actually used to perform these queries.
		TimeIntervals: f.TimeIntervals,
	}
	s := newServiceGraphConstructor(f, v)

	// Iterate through the flows to track the nodes and edges.
	for i := range s.sgd.FilteredFlows {
		if err := s.trackFlow(&s.sgd.FilteredFlows[i]); err != nil {
			log.WithError(err).WithField("flow", s.sgd.FilteredFlows[i]).Errorf("Unable to process flow")
			continue
		}
	}

	// Iterate through the collected services to fix up the selectors for edges originating from a service. These
	// selectors should be the ORed combination of all of the sources for edges destined for the service.
	for svcEp, ses := range s.serviceEdges {
		// OR together the source edge selectors for L3 flows.
		sourceEdgeSelector := v1.GraphSelectors{}
		for srcEp := range ses.sourceNodes {
			sourceEdgeSelector = sourceEdgeSelector.Or(srcEp.selectors.Source)
		}

		// It's only the L3 and DNS selectors that we care about for now. The L7 selectors will always have the service
		// available if it is available at all.
		sourceEdgeSelector = v1.GraphSelectors{
			L3Flows: sourceEdgeSelector.L3Flows,
			DNSLogs: sourceEdgeSelector.DNSLogs,
		}

		// Update the egress edges from the service to use the calculated selector.
		for dstEp := range ses.destNodes {
			edge := s.edgesMap[v1.GraphEdgeID{
				SourceNodeID: svcEp,
				DestNodeID:   dstEp.graphNode.ID,
			}]
			edge.Selectors = edge.Selectors.And(sourceEdgeSelector)
		}
	}

	// Get the nodes and parents that are in view.
	nodesInView := s.getNodesInView()

	// We no longer require the tracked nodes tracked groups - nilling out protects against these being re-used after
	// the set of in-view nodes has been determined.
	s.groupsMap = nil
	s.nodesMap = nil

	// Overlay the events on the in-view nodes.
	s.overlayEvents(nodesInView)

	// Overlay the dns client data on the in-view nodes.
	s.overlayDNS(nodesInView)

	if len(nodesInView) > 0 {
		// Copy across edges that are in view, and add the nodes to indicate whether we are truncating the graph (i.e.
		// that the graph can be followed along it's ingress or egress connections).
		for id, edge := range s.edgesMap {
			source := nodesInView[id.SourceNodeID]
			dest := nodesInView[id.DestNodeID]
			if source != nil && dest != nil {
				sgr.Edges = append(sgr.Edges, *edge)
			} else if source != nil {
				// Destination is not in view, but this means the egress can be Expanded for the source node. Mark this
				// on the group rather than the endpoint.
				source.graphNode.FollowEgress = true
			} else if dest != nil {
				// Source is not in view, but this means the ingress can be Expanded for the dest node. Mark this
				// on the group rather than the endpoint.
				dest.graphNode.FollowIngress = true
			}
		}

		// Copy across the nodes and parents that are in view.
		sgr.Nodes = make([]v1.GraphNode, 0, len(nodesInView))
		for _, n := range nodesInView {
			sgr.Nodes = append(sgr.Nodes, n.graphNode)
		}
	}

	// Trace out the nodes and edges if the log level is debug.
	if log.IsLevelEnabled(log.DebugLevel) {
		for _, node := range sgr.Nodes {
			log.Debugf("%s", node)
		}
		for _, edge := range sgr.Edges {
			log.Debugf("%s", edge)
		}
	}

	return sgr, nil
}

// trackedGroup is an internal struct used for tracking a node group (i.e. a node in the graph that does not have a
// parent). This is used to simplify the pruning algorithm since we only look at connectivity between these groups
// to determine if the node (and all its children, and its expanded parents) should be included or not.
type trackedGroup struct {
	node             *trackedNode
	parents          []*trackedNode
	children         []*trackedNode
	viewData         NodeViewData
	ingress          map[*trackedGroup]struct{}
	egress           map[*trackedGroup]struct{}
	processedIngress bool
	processedEgress  bool
}

// newTrackedGroup creates a new trackedGroup, setting the focus/following info.
func newTrackedGroup(node *trackedNode, parents []*trackedNode) *trackedGroup {
	vd := node.viewData
	for _, parent := range parents {
		vd = vd.Combine(parent.viewData)
	}

	tg := &trackedGroup{
		node:     node,
		parents:  parents,
		ingress:  make(map[*trackedGroup]struct{}),
		egress:   make(map[*trackedGroup]struct{}),
		viewData: vd,
	}

	return tg
}

// addChild adds a child node to a tracked group. This updates the groups focus/following info from the child specific
// values - this data is additive.
func (t *trackedGroup) addChild(child *trackedNode) {
	t.children = append(t.children, child)
	t.viewData = t.viewData.Combine(child.viewData)
}

// trackedNode encapsulates details of a node returned by the API, and additional data required to do some post
// graph-construction updates.
type trackedNode struct {
	graphNode v1.GraphNode
	parent    *trackedNode
	selectors SelectorPairs
	viewData  NodeViewData
}

// Track the source and dest nodes for each service node. We need to do this to generate the edge selectors for
// edges from the service to each dest. The source for each is the ORed combination of the sources.
type serviceEdges struct {
	sourceNodes map[*trackedNode]struct{}
	destNodes   map[*trackedNode]struct{}
}

func newServiceEdges() *serviceEdges {
	return &serviceEdges{
		sourceNodes: make(map[*trackedNode]struct{}),
		destNodes:   make(map[*trackedNode]struct{}),
	}
}

// serviceGraphConstructionData is the transient data used to construct the final service graph.
type serviceGraphConstructionData struct {
	// The set of tracked groups keyed of the group node ID.
	groupsMap map[v1.GraphNodeID]*trackedGroup

	// The full set of graph nodes keyed off the node ID.
	nodesMap map[v1.GraphNodeID]*trackedNode

	// The full set of graph edges keyed off the edge ID.
	edgesMap map[v1.GraphEdgeID]*v1.GraphEdge

	// The mapping between service and edges connected to the service.
	serviceEdges map[v1.GraphNodeID]*serviceEdges

	// The supplied service graph data.
	sgd *ServiceGraphData

	// The supplied view data.
	view *ParsedView

	// The selector helper used to construct selectors.
	selh *SelectorHelper
}

// newServiceGraphConstructor intializes a new serviceGraphConstructionData.
func newServiceGraphConstructor(sgd *ServiceGraphData, v *ParsedView) *serviceGraphConstructionData {
	return &serviceGraphConstructionData{
		groupsMap:    make(map[v1.GraphNodeID]*trackedGroup),
		nodesMap:     make(map[v1.GraphNodeID]*trackedNode),
		edgesMap:     make(map[v1.GraphEdgeID]*v1.GraphEdge),
		serviceEdges: make(map[v1.GraphNodeID]*serviceEdges),
		sgd:          sgd,
		view:         v,
		selh:         NewSelectorHelper(v, sgd.NameHelper, sgd.ServiceGroups),
	}
}

// trackFlow converts a flow into a set of graph nodes and edges. Each flow may be converted into one or more
// nodes (with parent relationships), and either zero, one or two edges.
//
// This tracks the graph node and edge data, aggregating the traffic stats as required. This also tracks connectivity
// between the endpoint groups to simplify graph pruning (we only consider connectivity between groups).
func (s *serviceGraphConstructionData) trackFlow(flow *TimeSeriesFlow) error {
	// Create the source and dest graph nodes. Note that if the source and dest nodes have a common root then add
	// the appropriate intra-node statistics. Note source will not include a service Port since that is an ingress
	// only concept.
	log.Debugf("Processing: %s", flow)
	var egress, ingress Direction
	if s.view.SplitIngressEgress {
		egress, ingress = DirectionEgress, DirectionIngress
	}

	srcGp, srcEp, _ := s.trackNodes(flow.Edge.Source, nil, egress)
	dstGp, dstEp, servicePortDst := s.trackNodes(flow.Edge.Dest, flow.Edge.ServicePort, ingress)

	// Include any aggregated port proto info in either the group or the endpoint.  If there is a service, include in
	// the group.
	if flow.AggregatedProtoPorts != nil {
		dstEp.graphNode.IncludeAggregatedProtoPorts(flow.AggregatedProtoPorts)
	}
	if flow.Edge.ServicePort != nil {
		dstGp.node.graphNode.IncludeService(flow.Edge.ServicePort.NamespacedName)
	}

	// If any of the source and dest nodes (including parents) are the same, add the stats to those nodes. Note the
	// hierarchy should be small, so doing this cross-product of checks will not be heavyweight.
	for srcNode := srcEp; srcNode != nil; srcNode = srcNode.parent {
		for dstNode := dstEp; dstNode != nil; dstNode = dstNode.parent {
			if srcNode == dstNode {
				srcNode.graphNode.IncludeStats(flow.Stats)
				break
			}
		}
	}

	// If the source and dest group are the same then do not add edges since this overly complicates the graph.
	// TODO(rlb): We could add an option to return these edges.
	if srcGp == dstGp {
		return nil
	}

	// Stitch together the source and dest nodes going via the service if present.
	if servicePortDst != nil {
		// There is a service port, so we have src->svc->dest
		var sourceEdge, destEdge *v1.GraphEdge
		var ok bool

		id := v1.GraphEdgeID{
			SourceNodeID: srcEp.graphNode.ID,
			DestNodeID:   servicePortDst.graphNode.ID,
		}
		log.Debugf("Tracking: %s", id)
		if sourceEdge, ok = s.edgesMap[id]; ok {
			sourceEdge.IncludeStats(flow.Stats)
		} else {
			sourceEdge = &v1.GraphEdge{
				ID:        id,
				Stats:     flow.Stats,
				Selectors: srcEp.selectors.Source.And(dstEp.selectors.Dest),
			}
			s.edgesMap[id] = sourceEdge
		}

		id = v1.GraphEdgeID{
			SourceNodeID: servicePortDst.graphNode.ID,
			DestNodeID:   dstEp.graphNode.ID,
		}
		log.Debugf("Tracking: %s", id)
		if destEdge, ok = s.edgesMap[id]; ok {
			destEdge.IncludeStats(flow.Stats)
		} else {
			destEdge = &v1.GraphEdge{
				ID:        id,
				Stats:     flow.Stats,
				Selectors: servicePortDst.selectors.Source.And(dstEp.selectors.Dest),
			}
			s.edgesMap[id] = destEdge
		}

		// Track the edges associated with a service node - we need to do this to fix up the service selectors since
		// it is not possible to match on service for destination based flows.
		se := s.serviceEdges[servicePortDst.graphNode.ID]
		if se == nil {
			se = newServiceEdges()
			s.serviceEdges[servicePortDst.graphNode.ID] = se
		}
		se.sourceNodes[srcEp] = struct{}{}
		se.destNodes[dstEp] = struct{}{}
	} else {
		// No service port, so direct src->dst
		id := v1.GraphEdgeID{
			SourceNodeID: srcEp.graphNode.ID,
			DestNodeID:   dstEp.graphNode.ID,
		}
		log.Debugf("Tracking: %s", id)
		if edge, ok := s.edgesMap[id]; ok {
			edge.IncludeStats(flow.Stats)
		} else {
			s.edgesMap[id] = &v1.GraphEdge{
				ID:        id,
				Stats:     flow.Stats,
				Selectors: srcEp.selectors.Source.And(dstEp.selectors.Dest),
			}
		}
	}

	// Track group interconnectivity for pruning purposes.
	srcGp.egress[dstGp] = struct{}{}
	dstGp.ingress[srcGp] = struct{}{}

	return nil
}

// trackNodes converts a FlowEndpoint and service to a set of hierarchical nodes. This is where we determine whether
// a node is aggregated into a layer, namespace, service group or aggregated endpoint set - only creating the nodes
// required based on the aggregation.
//
// This method updates the groupsMap and nodesMap, and returns the IDs of the group, the endpoint (to which the
// edge is connected) and the service port (which will be an additional hop).
func (s *serviceGraphConstructionData) trackNodes(
	endpoint FlowEndpoint, svc *ServicePort, dir Direction,
) (group *trackedGroup, endpointPortNode, servicePortNode *trackedNode) {
	// Track the expanded container parents and the current parent ID.
	var expandedParents []*trackedNode
	getParent := func() *trackedNode {
		if len(expandedParents) == 0 {
			return nil
		}
		return expandedParents[len(expandedParents)-1]
	}
	getParentID := func() v1.GraphNodeID {
		parent := getParent()
		if parent == nil {
			return ""
		}
		return parent.graphNode.ID
	}

	// Determine if this endpoint is in a layer - most granular wins.
	var sg *ServiceGroup
	if svc != nil {
		sg = s.sgd.ServiceGroups.GetByService(svc.NamespacedName)
	} else {
		sg = s.sgd.ServiceGroups.GetByEndpoint(endpoint)
	}
	// Create an ID handler.
	idi := IDInfo{
		Endpoint:     endpoint,
		ServiceGroup: sg,
		Direction:    dir,
	}
	if svc != nil {
		idi.Service = *svc
	}

	// Determine the group namespace for this node. If this node is part of service group then we use the namespace
	// associated with the service group, otherwise this is just the endpoint namespace. Note that the service group
	// namespace may be an aggregated name - this is fine - in this case the service group does not belong in a single
	// namespace. Layer selection and namespace expansion is based on this group namespace.
	groupNamespace := endpoint.Namespace
	if sg != nil {
		groupNamespace = sg.Namespace
	}

	// Determine the aggregated and full endpoint IDs and check if contained in a layer.
	nonAggrEndpointId := idi.GetEndpointID()
	aggrEndpointId := idi.GetAggrEndpointID()

	// If the endpoint/service group was directly part of a layer, and that layer is expanded then we effectively
	// bypass the namespace grouping. Keep track of whether the namespace layer should be skipped.
	var layerName string
	var skipNamespace bool

	// The parsed layer will only consist of sensible groups of endpoints and will not separate endpoints that cannot
	// be sensibly removed from other groups.
	if nonAggrEndpointId != "" {
		if layerName = s.view.Layers.EndpointToLayer[nonAggrEndpointId]; layerName != "" {
			skipNamespace = true
		}
	}

	if layerName == "" && aggrEndpointId != "" {
		if layerName = s.view.Layers.EndpointToLayer[aggrEndpointId]; layerName != "" {
			skipNamespace = true
		}
	}

	if layerName == "" && sg != nil {
		if layerName = s.view.Layers.ServiceGroupToLayer[sg]; layerName != "" {
			skipNamespace = true
		}
	}

	if layerName == "" && groupNamespace != "" {
		layerName = s.view.Layers.NamespaceToLayer[groupNamespace]
	}

	if layerName != "" {
		idi.Layer = layerName
		layerId := idi.GetLayerID()
		var layer *trackedNode
		if layer = s.nodesMap[layerId]; layer == nil {
			sel := s.selh.GetLayerNodeSelectors(layerName)
			layer = &trackedNode{
				graphNode: v1.GraphNode{
					ID:         layerId,
					Type:       v1.GraphNodeTypeLayer,
					Name:       layerName,
					Expandable: true,
					Selectors:  sel.ToNodeSelectors(),
				},
				selectors: sel,
				viewData:  s.view.NodeViewData[layerId],
			}

			// Set whether the node is expanded (we could do this as part of creating the node, but this reduces the
			// NodeViewData lookups), and then store the node.
			layer.graphNode.Expanded = layer.viewData.Expanded
			s.nodesMap[layerId] = layer
		}

		if !layer.viewData.Expanded {
			// Layer is not expanded. Track the layer as the group used for graph pruning, and return the layer as
			// both the group and the endpoint.
			if group = s.groupsMap[layerId]; group == nil {
				group = newTrackedGroup(layer, expandedParents)
				s.groupsMap[layerId] = group
			}
			return group, layer, nil
		}

		// Add the layer to the set of parent nodes.
		expandedParents = append(expandedParents, layer)
	}

	// If there is a namespace and we are not skipping the namespace (because the endpoint or service group are in a
	// layer which has been expanded) then add the namespace.
	if groupNamespace != "" && !skipNamespace {
		namespaceId := idi.GetNamespaceID()
		var namespace *trackedNode
		if namespace = s.nodesMap[namespaceId]; namespace == nil {
			sel := s.selh.GetNamespaceNodeSelectors(groupNamespace)
			namespace = &trackedNode{
				graphNode: v1.GraphNode{
					Type:       v1.GraphNodeTypeNamespace,
					ID:         namespaceId,
					ParentID:   getParentID(),
					Name:       groupNamespace,
					Expandable: true,
					Selectors:  sel.ToNodeSelectors(),
				},
				parent:    getParent(),
				selectors: sel,
				viewData:  s.view.NodeViewData[namespaceId],
			}

			// Set whether the node is expanded (we could do this as part of creating the node, but this reduces the
			// NodeViewData lookups), and then store the node.
			namespace.graphNode.Expanded = namespace.viewData.Expanded
			s.nodesMap[namespaceId] = namespace
		}

		if !namespace.viewData.Expanded {
			// Namespace is not expanded. Track the namespace as the group used for graph pruning, and return the
			// namespace as both the group and the endpoint.
			if group = s.groupsMap[namespaceId]; group == nil {
				group = newTrackedGroup(namespace, expandedParents)
				s.groupsMap[namespaceId] = group
			}
			return group, namespace, nil
		}

		// Add the namespace to the set of parent nodes.
		expandedParents = append(expandedParents, namespace)
	}

	// The graph constructor assumes the following are the least divisible units - in that we cannot split out child
	// nodes from these nodes:
	// - Service Group.  If, for example an endpoint is added to a layer, then the whole service group will be added
	//                   to the layer.  The endpoint will never appear as a node without the service group in its
	//                   parentage.
	// - Aggregated Endpoint.  If an endpoint is not associated with a service group, but is part of an aggregated
	//                   endpoint then similar rules apply as per Service Group. If, for example an endpoint is added
	//                   to a layer, then the aggregated endpoint group will be added to the layer.  The endpoint will
	//                   never appear as a node without the aggregated endpoint group in it's parentage.
	//
	// These rules exist to ensure groups of related endpoints are never split up since that would confuse the
	// metrics aggregation and hide important details about endpoint relationship.

	// If there is a service group then add the service group.
	if sg != nil {
		var serviceGroup *trackedNode
		if serviceGroup = s.nodesMap[sg.ID]; serviceGroup == nil {
			sel := s.selh.GetServiceGroupNodeSelectors(sg)
			serviceGroup = &trackedNode{
				graphNode: v1.GraphNode{
					Type:       v1.GraphNodeTypeServiceGroup,
					ID:         sg.ID,
					ParentID:   getParentID(),
					Namespace:  sg.Namespace,
					Name:       sg.Name,
					Expandable: true,
					Selectors:  sel.ToNodeSelectors(),
				},
				parent:    getParent(),
				selectors: sel,
				viewData:  s.view.NodeViewData[sg.ID],
			}

			// Set whether the node is expanded (we could do this as part of creating the node, but this reduces the
			// NodeViewData lookups), and then store the node.
			serviceGroup.graphNode.Expanded = serviceGroup.viewData.Expanded
			s.nodesMap[sg.ID] = serviceGroup
		}

		// Since there is a service group - we always track this as the tracking group even if the service group is
		// expanded.
		if group = s.groupsMap[sg.ID]; group == nil {
			group = newTrackedGroup(serviceGroup, expandedParents)
			s.groupsMap[sg.ID] = group
		}

		if !serviceGroup.viewData.Expanded {
			// If the service group is not expanded then return this as both the group and the endpoint.
			return group, serviceGroup, nil
		}

		// Add the service group to the set of expanded parents.
		expandedParents = append(expandedParents, serviceGroup)

		// If there is a service we will need to add that node and the service port. We return the service port ID since
		// this is an ingress point.
		if svc != nil {
			serviceId := idi.GetServiceID()
			var service *trackedNode
			if service = s.nodesMap[serviceId]; service == nil {
				sel := s.selh.GetServiceNodeSelectors(svc.NamespacedName)
				service = &trackedNode{
					graphNode: v1.GraphNode{
						Type:      v1.GraphNodeTypeService,
						ID:        serviceId,
						ParentID:  sg.ID,
						Namespace: svc.Namespace,
						Name:      svc.Name,
						Selectors: sel.ToNodeSelectors(),
					},
					parent:    serviceGroup,
					selectors: sel,
					viewData:  s.view.NodeViewData[serviceId],
				}
				s.nodesMap[serviceId] = service
				group.addChild(service)
			}

			servicePortId := idi.GetServicePortID()
			if servicePortNode = s.nodesMap[servicePortId]; servicePortNode == nil {
				sel := s.selh.GetServicePortNodeSelectors(*svc)
				servicePortNode = &trackedNode{
					graphNode: v1.GraphNode{
						Type:      v1.GraphNodeTypeServicePort,
						ID:        servicePortId,
						ParentID:  serviceId,
						Name:      svc.Port,
						Selectors: sel.ToNodeSelectors(),
					},
					parent:    service,
					selectors: sel,
					viewData:  s.view.NodeViewData[servicePortId],
				}
				s.nodesMap[servicePortId] = servicePortNode
				group.addChild(servicePortNode)
			}
		}
	}

	// Combine the aggregated endpoint node - this should always be available for a flow.
	var aggrEndpoint *trackedNode
	if aggrEndpoint = s.nodesMap[aggrEndpointId]; aggrEndpoint == nil {
		sel := s.selh.GetEndpointNodeSelectors(
			idi.GetAggrEndpointType(),
			endpoint.Namespace,
			endpoint.Name,
			endpoint.NameAggr,
			NoProto,
			NoPort, idi.Direction,
		)
		aggrEndpoint = &trackedNode{
			graphNode: v1.GraphNode{
				Type:       idi.GetAggrEndpointType(),
				ID:         aggrEndpointId,
				ParentID:   getParentID(),
				Namespace:  endpoint.Namespace,
				Name:       endpoint.NameAggr,
				Expandable: nonAggrEndpointId != "",
				Selectors:  sel.ToNodeSelectors(),
			},
			parent:    getParent(),
			selectors: sel,
			viewData:  s.view.NodeViewData[aggrEndpointId],
		}

		// Set whether the node is expanded (we could do this as part of creating the node, but this reduces the
		// NodeViewData lookups), and then store the node.
		if aggrEndpoint.graphNode.Expandable {
			aggrEndpoint.graphNode.Expanded = aggrEndpoint.viewData.Expanded
		}
		s.nodesMap[aggrEndpointId] = aggrEndpoint
	}

	if group == nil {
		// There is no outer group for this aggregated endpoint, so the aggregated endpoint is also the group.
		if group = s.groupsMap[aggrEndpointId]; group == nil {
			group = newTrackedGroup(aggrEndpoint, expandedParents)
			s.groupsMap[aggrEndpointId] = group
		}
	} else {
		// There is an outer group for this aggregated endpoint, so add this endpoint to the group.
		group.addChild(aggrEndpoint)
	}

	// If the endpoint is expanded then add the port if present.
	if !aggrEndpoint.graphNode.Expanded {
		log.Debugf("Group is not expanded or not expandable: %s; %s, %s", group.node.graphNode.ID, aggrEndpointId, nonAggrEndpointId)

		if aggrEndpointPortId := idi.GetAggrEndpointPortID(); aggrEndpointPortId != "" {
			var aggrEndpointPort *trackedNode
			if aggrEndpointPort = s.nodesMap[aggrEndpointPortId]; aggrEndpointPort == nil {
				sel := s.selh.GetEndpointNodeSelectors(
					idi.GetAggrEndpointType(),
					endpoint.Namespace,
					endpoint.Name,
					endpoint.NameAggr,
					endpoint.Proto,
					endpoint.Port,
					idi.Direction,
				)
				aggrEndpointPort = &trackedNode{
					graphNode: v1.GraphNode{
						Type:      v1.GraphNodeTypePort,
						ID:        aggrEndpointPortId,
						ParentID:  aggrEndpointId,
						Port:      endpoint.Port,
						Protocol:  endpoint.Proto,
						Selectors: sel.ToNodeSelectors(),
					},
					parent:    aggrEndpoint,
					selectors: sel,
					viewData:  s.view.NodeViewData[aggrEndpointPortId],
				}
				s.nodesMap[aggrEndpointPortId] = aggrEndpointPort
				group.addChild(aggrEndpointPort)
			}
			return group, aggrEndpointPort, servicePortNode
		}
		return group, aggrEndpoint, servicePortNode
	}

	// The endpoint is expanded and expandable.
	var nonAggrEndpoint *trackedNode
	if nonAggrEndpoint = s.nodesMap[nonAggrEndpointId]; nonAggrEndpoint == nil {
		sel := s.selh.GetEndpointNodeSelectors(
			idi.Endpoint.Type,
			endpoint.Namespace,
			endpoint.Name,
			endpoint.NameAggr,
			NoProto,
			NoPort,
			idi.Direction,
		)
		nonAggrEndpoint = &trackedNode{
			graphNode: v1.GraphNode{
				Type:      idi.Endpoint.Type,
				ID:        nonAggrEndpointId,
				ParentID:  aggrEndpointId,
				Namespace: endpoint.Namespace,
				Name:      endpoint.Name,
				Selectors: sel.ToNodeSelectors(),
			},
			parent:    aggrEndpoint,
			selectors: sel,
			viewData:  s.view.NodeViewData[nonAggrEndpointId],
		}
		s.nodesMap[nonAggrEndpointId] = nonAggrEndpoint
		group.addChild(nonAggrEndpoint)
	}

	if nonAggrEndpointPortId := idi.GetEndpointPortID(); nonAggrEndpointPortId != "" {
		var nonAggrEndpointPort *trackedNode
		if nonAggrEndpointPort = s.nodesMap[nonAggrEndpointId]; nonAggrEndpointPort == nil {
			sel := s.selh.GetEndpointNodeSelectors(
				idi.Endpoint.Type,
				endpoint.Namespace,
				endpoint.Name,
				endpoint.NameAggr,
				endpoint.Proto,
				endpoint.Port,
				idi.Direction,
			)
			nonAggrEndpointPort = &trackedNode{
				graphNode: v1.GraphNode{
					Type:      v1.GraphNodeTypePort,
					ID:        nonAggrEndpointPortId,
					ParentID:  nonAggrEndpointId,
					Port:      endpoint.Port,
					Protocol:  endpoint.Proto,
					Selectors: sel.ToNodeSelectors(),
				},
				parent:    nonAggrEndpoint,
				selectors: sel,
				viewData:  s.view.NodeViewData[nonAggrEndpointPortId],
			}
			s.nodesMap[nonAggrEndpointPortId] = nonAggrEndpointPort
			group.addChild(nonAggrEndpointPort)
		}
		return group, nonAggrEndpointPort, servicePortNode
	}

	return group, nonAggrEndpoint, servicePortNode
}

// getNodeInView determines which nodes are in view. This returns the set of trackedNodes that are in view, and the
// set of parent nodes that are expanded and associated with the in-view children.
//
// This is then used to select the final set of nodes and edges for the service graph.
func (s *serviceGraphConstructionData) getNodesInView() (nodes map[v1.GraphNodeID]*trackedNode) {
	nodes = make(map[v1.GraphNodeID]*trackedNode)

	// Special case when focus is empty - this indicates full view when everything is visible.
	if s.view.EmptyFocus {
		log.Debug("No view selected - return all nodes")

		nodes = s.nodesMap
		if log.IsLevelEnabled(log.DebugLevel) {
			for n := range nodes {
				log.Debugf("Including node: %s", n)
			}
		}
		return nodes
	}

	// Keep expanding until we have processed all groups that are in-view.  There are three parts to this expansion:
	// - Expand the in-focus nodes in both directions
	// - If connection direction is being following, carry on expanding ingress and egress directions outwards from
	//   in-focus nodes
	// - If a node connection is being explicitly followed, keep expanding until all expansion points are exhausted.

	log.Debug("Expanding nodes explicitly in view")
	groupsInView := make(map[*trackedGroup]struct{})
	expandIngress := make(map[*trackedGroup]struct{})
	expandEgress := make(map[*trackedGroup]struct{})
	expandFollowing := make(map[*trackedGroup]struct{})
	for id, gp := range s.groupsMap {
		if gp.viewData.InFocus {
			log.Debugf("Expand ingress and egress for in-focus node: %s", id)
			groupsInView[gp] = struct{}{}
			expandIngress[gp] = struct{}{}
			expandEgress[gp] = struct{}{}
		}
	}

	// Expand in-Focus nodes in ingress Direction and possibly follow connection direction.
	for len(expandIngress) > 0 {
		for gp := range expandIngress {
			if gp.processedIngress {
				delete(expandIngress, gp)
				continue
			}

			// Add ingress nodes for this group.
			gp.processedIngress = true
			for connectedGp := range gp.ingress {
				log.Debugf("Including ingress expanded group: %s -> %s", connectedGp.node.graphNode.ID, gp.node.graphNode.ID)
				groupsInView[connectedGp] = struct{}{}
				if s.view.FollowConnectionDirection {
					expandIngress[connectedGp] = struct{}{}
				} else if connectedGp.viewData.FollowedEgress || connectedGp.viewData.FollowedIngress {
					log.Debugf("Following ingress and/or egress direction from: %s", connectedGp.node.graphNode.ID)
					expandFollowing[connectedGp] = struct{}{}
				}
			}

			delete(expandIngress, gp)
		}
	}

	// Expand in-Focus nodes in ingress Direction and possibly follow connection direction.
	for len(expandEgress) > 0 {
		for gp := range expandEgress {
			if gp.processedEgress {
				delete(expandEgress, gp)
				continue
			}

			// Add egress nodes for this group.
			gp.processedEgress = true
			for connectedGp := range gp.egress {
				log.Debugf("Including egress expanded group: %s -> %s", gp.node.graphNode.ID, connectedGp.node.graphNode.ID)
				groupsInView[connectedGp] = struct{}{}

				if s.view.FollowConnectionDirection {
					expandEgress[connectedGp] = struct{}{}
				} else if connectedGp.viewData.FollowedEgress || connectedGp.viewData.FollowedIngress {
					log.Debugf("Following ingress and/or egress direction from: %s", connectedGp.node.graphNode.ID)
					expandFollowing[connectedGp] = struct{}{}
				}
			}

			delete(expandEgress, gp)
		}
	}

	// Expand followed nodes.
	for len(expandFollowing) > 0 {
		for gp := range expandFollowing {
			if gp.viewData.FollowedIngress && !gp.processedIngress {
				gp.processedIngress = true
				for followedGp := range gp.ingress {
					log.Debugf("Following ingress from %s to %s", gp.node.graphNode.ID, followedGp.node.graphNode.ID)
					groupsInView[followedGp] = struct{}{}
					expandFollowing[followedGp] = struct{}{}
				}
			}
			if gp.viewData.FollowedEgress && !gp.processedEgress {
				gp.processedEgress = true
				for followedGp := range gp.egress {
					log.Debugf("Following egress from %s to %s", gp.node.graphNode.ID, followedGp.node.graphNode.ID)
					groupsInView[followedGp] = struct{}{}
					expandFollowing[followedGp] = struct{}{}
				}
			}

			delete(expandFollowing, gp)
		}
	}

	// Create the full set of nodes that are in view.
	for gp := range groupsInView {
		nodes[gp.node.graphNode.ID] = gp.node
		for _, child := range gp.children {
			nodes[child.graphNode.ID] = child
		}
		for _, parent := range gp.parents {
			nodes[parent.graphNode.ID] = parent
		}
	}

	// Log the full set of nodes in view.
	if log.IsLevelEnabled(log.DebugLevel) {
		for n := range nodes {
			log.Debugf("Including node: %s", n)
		}
	}

	return nodes
}

// overlayEvents iterates through all the events and overlays them on the existing graph nodes. This never adds more
// nodes to the graph.
func (s *serviceGraphConstructionData) overlayEvents(nodesInView map[v1.GraphNodeID]*trackedNode) {
	if nodesInView == nil {
		return
	}

	for _, event := range s.sgd.Events {
		log.Debugf("Checking event %#v", event)
		for _, ep := range event.Endpoints {
			log.Debugf("  - Checking event endpoint: %#v", ep)
			var node *trackedNode
			switch ep.Type {
			case v1.GraphNodeTypeService:
				fep := FlowEndpoint{
					Type:      ep.Type,
					Namespace: ep.Namespace,
				}
				sg := s.sgd.ServiceGroups.GetByService(v1.NamespacedName{
					Namespace: ep.Namespace, Name: ep.Name,
				})
				if node = s.getMostGranularNodeInView(nodesInView, fep, sg); node != nil {
					node.graphNode.IncludeEvent(event.ID, event.Details)
				}
			default:
				sg := s.sgd.ServiceGroups.GetByEndpoint(ep)
				if node = s.getMostGranularNodeInView(nodesInView, ep, sg); node != nil {
					node.graphNode.IncludeEvent(event.ID, event.Details)
				}
			}
		}
	}
}

// overlayDNS iterates through all the DNS logs and overlays them on the existing graph nodes. The stats are added
// to the endpoint and all parent nodes in the hierarchy.
func (s *serviceGraphConstructionData) overlayDNS(nodesInView map[v1.GraphNodeID]*trackedNode) {
	if nodesInView == nil {
		return
	}

	for _, dl := range s.sgd.FilteredDNSClientLogs {
		log.Debugf("Checking DNS log for endpoint %#v", dl.Endpoint)
		sg := s.sgd.ServiceGroups.GetByEndpoint(dl.Endpoint)

		for node := s.getMostGranularNodeInView(nodesInView, dl.Endpoint, sg); node != nil; node = node.parent {
			node.graphNode.IncludeStats(dl.Stats)
		}
	}
}

// getMostGranularNodeInView returns the most granular node that is in view for a given endpoint.
//
// Note: This duplicates a lot of the processing in trackNodes, so we might want to think about just using that
//       to locate the nodes. This processing is, however, a little more lightweight since it only needs to
//       consider nodes that already exist - and can just track the most granular node visible rather than involved
//       in the node expansion processing.
func (s *serviceGraphConstructionData) getMostGranularNodeInView(
	nodesInView map[v1.GraphNodeID]*trackedNode, ep FlowEndpoint, sg *ServiceGroup,
) *trackedNode {
	idi := IDInfo{
		Endpoint:     ep,
		ServiceGroup: sg,
		Direction:    "",
	}

	// Start with the most granular up to service group.
	// The non-aggregated endpoint.
	nonAggrEndpointId := idi.GetEndpointID()
	if nonAggrEndpointId != "" {
		log.Debugf("Checking if endpoint exists: %s", nonAggrEndpointId)
		if nonAggrEndpoint := nodesInView[nonAggrEndpointId]; nonAggrEndpoint != nil {
			log.Debug("Endpoint exists")
			return nonAggrEndpoint
		}
	}

	// Aggregated endpoint.
	aggrEndpointId := idi.GetAggrEndpointID()
	if aggrEndpointId != "" {
		log.Debugf("Checking if aggr endpoint exists: %s", aggrEndpointId)
		if aggrEndpoint := nodesInView[aggrEndpointId]; aggrEndpoint != nil {
			log.Debug("Aggr endpoint exists")
			return aggrEndpoint
		}
	}

	// Service group.
	if sg != nil {
		log.Debugf("Checking if service group exists: %s", sg.ID)
		if serviceGroup := nodesInView[sg.ID]; serviceGroup != nil {
			log.Debug("Service Group exists")
			return serviceGroup
		}
	}

	// Check layer first - if the endpoint or service group are part of a layer then check if the layer is in view.
	if idi.Layer != "" && nonAggrEndpointId != "" {
		if idi.Layer = s.view.Layers.EndpointToLayer[nonAggrEndpointId]; idi.Layer != "" {
			return nodesInView[idi.GetLayerID()]
		}
	}
	if idi.Layer != "" && aggrEndpointId != "" {
		if idi.Layer = s.view.Layers.EndpointToLayer[aggrEndpointId]; idi.Layer != "" {
			return nodesInView[idi.GetLayerID()]
		}
	}
	if idi.Layer != "" && sg != nil {
		if idi.Layer = s.view.Layers.ServiceGroupToLayer[sg]; idi.Layer != "" {
			return nodesInView[idi.GetLayerID()]
		}
	}

	// Now finally, check if the namespace is in view, and if not in view whether the namespace is part of a layer.
	groupNamespace := idi.GetEffectiveNamespace()
	if groupNamespace == "" {
		return nil
	}

	namespaceId := idi.GetNamespaceID()
	if namespace := nodesInView[namespaceId]; namespace != nil {
		log.Debug("Namespace exists")
		return namespace
	}

	if idi.Layer = s.view.Layers.NamespaceToLayer[groupNamespace]; idi.Layer == "" {
		return nil
	}

	return nodesInView[idi.GetLayerID()]
}
