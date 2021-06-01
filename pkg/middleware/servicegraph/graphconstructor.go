// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/set"

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
		ses.sourceNodes.Iter(func(item interface{}) error {
			srcEp := item.(v1.GraphNodeID)
			sourceEdgeSelector = sourceEdgeSelector.Or(s.nodesMap[srcEp].Selectors.Source)
			return nil
		})

		// It's only the L3 and DNS selectors that we care about for now. The L7 selectors will always have the service
		// available if it is available at all.
		sourceEdgeSelector = v1.GraphSelectors{
			L3Flows: sourceEdgeSelector.L3Flows,
			DNSLogs: sourceEdgeSelector.DNSLogs,
		}

		// Update the egress edges from the service to use the calculated selector.
		ses.destNodes.Iter(func(item interface{}) error {
			dstEp := item.(v1.GraphNodeID)
			edge := s.edgesMap[v1.GraphEdgeID{
				SourceNodeID: svcEp, DestNodeID: dstEp,
			}]
			edge.Selectors = edge.Selectors.And(sourceEdgeSelector)
			return nil
		})
	}

	getGroupNode := func(id v1.GraphNodeID) *v1.GraphNode {
		n := s.nodesMap[id]
		for n.Node.ParentID != "" {
			n = s.nodesMap[n.Node.ParentID]
		}
		return n.Node
	}

	nodesInView := s.getNodesInView()

	// Overlay the events on the in-view nodes.
	s.overlayEvents(nodesInView)

	if nodesInView != nil && nodesInView.Len() > 0 {
		// Copy across edges that are in view, and update the nodes to indicate whether we are truncating the graph (i.e.
		// that the graph can be followed along it's ingress or egress connections).
		for id, edge := range s.edgesMap {
			sourceInView := nodesInView.Contains(id.SourceNodeID)
			destInView := nodesInView.Contains(id.DestNodeID)
			if sourceInView && destInView {
				sgr.Edges = append(sgr.Edges, *edge)
			} else if sourceInView {
				// Destination is not in view, but this means the egress can be Expanded for the source node. Mark this
				// on the group rather than the endpoint.
				getGroupNode(id.SourceNodeID).FollowEgress = true
			} else if destInView {
				// Source is not in view, but this means the ingress can be Expanded for the dest node. Mark this
				// on the group rather than the endpoint.
				getGroupNode(id.DestNodeID).FollowIngress = true
			}
		}

		// Copy across the nodes that are in view.
		sgr.Nodes = make([]v1.GraphNode, 0, nodesInView.Len())
		nodesInView.Iter(func(item interface{}) error {
			id := item.(v1.GraphNodeID)
			sgr.Nodes = append(sgr.Nodes, *s.nodesMap[id].Node)
			return nil
		})
	} else if nodesInView == nil && len(s.nodesMap) > 0 {
		// There is no focus, which means everything is in view, and there are some nodes to return.  Include them all
		// and all edges.
		sgr.Nodes = make([]v1.GraphNode, 0, len(s.nodesMap))
		for _, n := range s.nodesMap {
			sgr.Nodes = append(sgr.Nodes, *n.Node)
		}
		sgr.Edges = make([]v1.GraphEdge, 0, len(s.edgesMap))
		for _, edge := range s.edgesMap {
			sgr.Edges = append(sgr.Edges, *edge)
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
// parent). This is used to simplify the pruning algorithm since we only look at connectivity between node groups
// to determine if the node (and all it's children) should be included or not.
type trackedGroup struct {
	id                 v1.GraphNodeID
	ingress            set.Set
	egress             set.Set
	isInFocus          bool
	isFollowingEgress  bool
	isFollowingIngress bool
	children           []v1.GraphNodeID
	processedIngress   bool
	processedEgress    bool
}

// newTrackedGroup creates a new trackedGroup, setting the focus/following info.
func newTrackedGroup(id v1.GraphNodeID, isInFocus, isFollowingEgress, isFollowigIngress bool) *trackedGroup {
	return &trackedGroup{
		id:                 id,
		ingress:            set.New(),
		egress:             set.New(),
		isInFocus:          isInFocus,
		isFollowingEgress:  isFollowingEgress,
		isFollowingIngress: isFollowigIngress,
	}
}

// update updates the focus/following info for a node group. This data is additive.
func (t *trackedGroup) update(child v1.GraphNodeID, isInFocus, isFollowingEgress, isFollowingIngress bool) {
	t.children = append(t.children, child)
	t.isInFocus = t.isInFocus || isInFocus
	t.isFollowingEgress = t.isFollowingEgress || isFollowingEgress
	t.isFollowingIngress = t.isFollowingIngress || isFollowingIngress
}

// trackedNode encapsulates details of a node returned by the API, and additional data required to do some post
// graph-construction updates.
type trackedNode struct {
	Node      *v1.GraphNode
	Selectors SelectorPairs
}

// Track the source and dest nodes for each service node. We need to do this to generate the edge selectors for
// edges from the service to each dest. The source for each is the ORed combination of the sources.
type serviceEdges struct {
	sourceNodes set.Set
	destNodes   set.Set
}

func newServiceEdges() *serviceEdges {
	return &serviceEdges{
		sourceNodes: set.New(),
		destNodes:   set.New(),
	}
}

// serviceGraphConstructionData is the transient data used to construct the final service graph.
type serviceGraphConstructionData struct {
	// The set of tracked groups keyed of the group node ID.
	groupsMap map[v1.GraphNodeID]*trackedGroup

	// The full set of graph nodes keyed off the node ID.
	nodesMap map[v1.GraphNodeID]trackedNode

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
		nodesMap:     make(map[v1.GraphNodeID]trackedNode),
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
	// Create the source and dest graph nodes. Note that if the source and dest nodes have a common root then update
	// the appropriate intra-node statistics. Note source will not include a service Port since that is an ingress
	// only concept.
	log.Debugf("Processing: %s", flow)
	var egress, ingress Direction
	if s.view.SplitIngressEgress {
		egress, ingress = DirectionEgress, DirectionIngress
	}

	srcGp, srcEp, _ := s.trackNodes(flow.Edge.Source, nil, egress)
	dstGp, dstEp, servicePortDst := s.trackNodes(flow.Edge.Dest, flow.Edge.ServicePort, ingress)

	// Include any aggregated port proto info in either the group or the endpoint.  Include the service in the group if
	// it is not expanded.
	if flow.AggregatedProtoPorts != nil {
		s.nodesMap[dstEp].Node.IncludeAggregatedProtoPorts(flow.AggregatedProtoPorts)
	}
	if dstGp == dstEp {
		if flow.Edge.ServicePort != nil {
			s.nodesMap[dstGp].Node.IncludeService(flow.Edge.ServicePort.NamespacedName)
		}
	}

	// If the source and dest group are the same then do not add edges, instead add the traffic stats and processes to
	// the group node.
	if srcGp == dstGp {
		s.nodesMap[srcGp].Node.IncludeStats(flow.Stats)
		return nil
	}

	// Stitch together the source and dest nodes going via the service if present.
	if servicePortDst != "" {
		// There is a service port, so we have src->svc->dest
		var sourceEdge, destEdge *v1.GraphEdge
		var ok bool

		id := v1.GraphEdgeID{
			SourceNodeID: srcEp,
			DestNodeID:   servicePortDst,
		}
		log.Debugf("Tracking: %s", id)
		if sourceEdge, ok = s.edgesMap[id]; ok {
			sourceEdge.IncludeStats(flow.Stats)
		} else {
			sourceEdge = &v1.GraphEdge{
				ID:        id,
				Stats:     flow.Stats,
				Selectors: s.nodesMap[srcEp].Selectors.Source.And(s.nodesMap[servicePortDst].Selectors.Dest),
			}
			s.edgesMap[id] = sourceEdge
		}

		id = v1.GraphEdgeID{
			SourceNodeID: servicePortDst,
			DestNodeID:   dstEp,
		}
		log.Debugf("Tracking: %s", id)
		if destEdge, ok = s.edgesMap[id]; ok {
			destEdge.IncludeStats(flow.Stats)
		} else {
			destEdge = &v1.GraphEdge{
				ID:        id,
				Stats:     flow.Stats,
				Selectors: s.nodesMap[servicePortDst].Selectors.Source.And(s.nodesMap[dstEp].Selectors.Dest),
			}
			s.edgesMap[id] = destEdge
		}

		// Track the edges associated with a service node - we need to do this to fix up the service selectors since
		// it is not possible to match on service for destination based flows.
		se := s.serviceEdges[servicePortDst]
		if se == nil {
			se = newServiceEdges()
			s.serviceEdges[servicePortDst] = se
		}
		se.sourceNodes.Add(srcEp)
		se.destNodes.Add(dstEp)
	} else {
		// No service port, so direct src->dst
		id := v1.GraphEdgeID{
			SourceNodeID: srcEp,
			DestNodeID:   dstEp,
		}
		log.Debugf("Tracking: %s", id)
		if edge, ok := s.edgesMap[id]; ok {
			edge.IncludeStats(flow.Stats)
		} else {
			s.edgesMap[id] = &v1.GraphEdge{
				ID:        id,
				Stats:     flow.Stats,
				Selectors: s.nodesMap[srcEp].Selectors.Source.And(s.nodesMap[dstEp].Selectors.Dest),
			}
		}
	}

	// Track group interconnectivity for pruning purposes.
	srcGpNode := s.groupsMap[srcGp]
	dstGpNode := s.groupsMap[dstGp]
	srcGpNode.egress.Add(dstGpNode)
	dstGpNode.ingress.Add(srcGpNode)

	return nil
}

// trackNodes converts a FlowEndpoint and service to a set of hierarchical nodes. This is where we determine whether
// a node is aggregated into a layer, namespace, service group or aggregated endpoint set - only creating the nodes
// required based on the aggregation.
//
// This method updates the groupsMap and nodesMap, and returns the IDs of the group, the endpoint (to which the edge is
// connected) and the service port (which will be an additional hop).
func (s *serviceGraphConstructionData) trackNodes(
	endpoint FlowEndpoint, svc *ServicePort, dir Direction,
) (groupId, endpointId, serviceId v1.GraphNodeID) {
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

	// Determine the aggregated and full endpoint IDs and check if contained in a layer.
	nonAggrEndpointId := idi.GetEndpointID()
	aggrEndpointId := idi.GetAggrEndpointID()

	var layerName, layerNameEndpoint, layerNameAggrEndpoint, layerNameServiceGroup, layerNameNamespace string

	if nonAggrEndpointId != "" {
		layerNameEndpoint = s.view.Layers.EndpointToLayer[nonAggrEndpointId]
		layerName = layerNameEndpoint
	}

	if layerName == "" && aggrEndpointId != "" {
		layerNameAggrEndpoint = s.view.Layers.EndpointToLayer[aggrEndpointId]
		layerName = layerNameAggrEndpoint
	}

	if layerName == "" && sg != nil {
		layerNameServiceGroup = s.view.Layers.ServiceGroupToLayer[sg]
		layerName = layerNameServiceGroup
	}

	if layerName == "" && endpoint.Namespace != "" {
		layerNameNamespace = s.view.Layers.NamespaceToLayer[endpoint.Namespace]
		layerName = layerNameNamespace
	}

	// If the layer is not Expanded then return the layer node.
	if layerName != "" && !s.view.Expanded.Layers[layerName] {
		idi.Layer = layerName
		layerId := idi.GetLayerID()
		if _, ok := s.nodesMap[layerId]; !ok {
			sel := s.selh.GetLayerNodeSelectors(layerName)
			s.nodesMap[layerId] = trackedNode{
				Node: &v1.GraphNode{
					ID:         layerId,
					Type:       v1.GraphNodeTypeLayer,
					Name:       layerName,
					Expandable: true,
					Selectors:  sel.ToNodeSelectors(),
				},
				Selectors: sel,
			}
			s.groupsMap[layerId] = newTrackedGroup(
				layerId, s.view.Focus.Layers[layerName],
				s.view.FollowedEgress.Layers[layerName],
				s.view.FollowedIngress.Layers[layerName],
			)
		}
		return layerId, layerId, ""
	}

	// If the namespace is not Expanded then return the namespace node. If the namespace is part of an Expanded layer
	// then include the layer name.
	if endpoint.Namespace != "" && !s.view.Expanded.Namespaces[endpoint.Namespace] {
		namespaceId := idi.GetNamespaceID()
		if _, ok := s.nodesMap[namespaceId]; !ok {
			sel := s.selh.GetNamespaceNodeSelectors(endpoint.Namespace)
			s.nodesMap[namespaceId] = trackedNode{
				Node: &v1.GraphNode{
					Type:       v1.GraphNodeTypeNamespace,
					ID:         namespaceId,
					Name:       endpoint.Namespace,
					Layer:      layerNameNamespace,
					Expandable: true,
					Selectors:  sel.ToNodeSelectors(),
				},
				Selectors: sel,
			}
			s.groupsMap[namespaceId] = newTrackedGroup(
				namespaceId,
				s.view.Focus.Namespaces[endpoint.Namespace] || s.view.Focus.Layers[layerName],
				s.view.FollowedEgress.Namespaces[endpoint.Namespace] || s.view.FollowedEgress.Layers[layerName],
				s.view.FollowedIngress.Namespaces[endpoint.Namespace] || s.view.FollowedIngress.Layers[layerName],
			)
		}
		return namespaceId, namespaceId, ""
	}

	// If the service group is not Expanded then return the service group node. If the service group is part of an
	// Expanded layer then include the layer name.
	if sg != nil {
		if _, ok := s.nodesMap[sg.ID]; !ok {
			sel := s.selh.GetServiceGroupNodeSelectors(sg)
			s.nodesMap[sg.ID] = trackedNode{
				Node: &v1.GraphNode{
					Type:       v1.GraphNodeTypeServiceGroup,
					ID:         sg.ID,
					Namespace:  sg.Namespace,
					Name:       sg.Name,
					Layer:      layerNameServiceGroup,
					Expandable: true,
					Selectors:  sel.ToNodeSelectors(),
				},
				Selectors: sel,
			}
			s.groupsMap[sg.ID] = newTrackedGroup(
				sg.ID,
				s.view.Focus.ServiceGroups[sg] ||
					s.view.Focus.Namespaces[endpoint.Namespace] ||
					s.view.Focus.Layers[layerName],
				s.view.FollowedEgress.ServiceGroups[sg] ||
					s.view.FollowedEgress.Namespaces[endpoint.Namespace] ||
					s.view.FollowedEgress.Layers[layerName],
				s.view.FollowedIngress.ServiceGroups[sg] ||
					s.view.FollowedIngress.Namespaces[endpoint.Namespace] ||
					s.view.FollowedIngress.Layers[layerName],
			)
		}

		if !s.view.Expanded.ServiceGroups[sg] {
			return sg.ID, sg.ID, ""
		}

		groupId = sg.ID

		// If there is a service we will need to add that node too, and return the ID.
		if svc != nil {
			serviceId = idi.GetServicePortID()
			if _, ok := s.nodesMap[serviceId]; !ok {
				sel := s.selh.GetServicePortNodeSelectors(*svc)
				s.nodesMap[serviceId] = trackedNode{
					Node: &v1.GraphNode{
						Type:        v1.GraphNodeTypeServicePort,
						ID:          serviceId,
						ParentID:    sg.ID,
						Namespace:   svc.Namespace,
						Name:        svc.Name,
						ServicePort: svc.Port,
						Selectors:   sel.ToNodeSelectors(),
					},
					Selectors: sel,
				}
				s.groupsMap[groupId].update(serviceId, false, false, false)
			}
		}
	}

	// Determine if the non-aggregated node is expanded or expandable.
	aggrEndpointExpandable := nonAggrEndpointId != ""
	aggrEndpointExpanded := s.view.Expanded.Endpoints[aggrEndpointId]

	// Combine the aggregated endpoint node - this should  always be available for a flow.
	if _, ok := s.nodesMap[aggrEndpointId]; !ok {
		sel := s.selh.GetEndpointNodeSelectors(
			idi.GetAggrEndpointType(),
			endpoint.Namespace,
			endpoint.Name,
			endpoint.NameAggr,
			NoProto,
			NoPort, idi.Direction,
		)
		s.nodesMap[aggrEndpointId] = trackedNode{
			Node: &v1.GraphNode{
				Type:       idi.GetAggrEndpointType(),
				ID:         aggrEndpointId,
				ParentID:   groupId,
				Namespace:  endpoint.Namespace,
				Name:       endpoint.NameAggr,
				Layer:      layerNameEndpoint,
				Expandable: aggrEndpointExpandable && !aggrEndpointExpanded,
				Selectors:  sel.ToNodeSelectors(),
			},
			Selectors: sel,
		}

		isInFocus := s.view.Focus.Endpoints[aggrEndpointId] ||
			s.view.Focus.Namespaces[endpoint.Namespace] ||
			s.view.Focus.Layers[layerName]
		isFollowingEgress := s.view.FollowedEgress.Endpoints[aggrEndpointId] ||
			s.view.FollowedEgress.Namespaces[endpoint.Namespace] ||
			s.view.FollowedEgress.Layers[layerName]
		isFollowingIngress := s.view.FollowedIngress.Endpoints[aggrEndpointId] ||
			s.view.FollowedIngress.Namespaces[endpoint.Namespace] ||
			s.view.FollowedIngress.Layers[layerName]

		if groupId == "" {
			// The endpoint is also the group, so track it.
			s.groupsMap[aggrEndpointId] = newTrackedGroup(
				aggrEndpointId,
				isInFocus,
				isFollowingEgress,
				isFollowingIngress,
			)
		} else {
			// The endpoint is not the group. Update the view details for the group and include the endpoint as
			// a child.
			s.groupsMap[groupId].update(aggrEndpointId, isInFocus, isFollowingEgress, isFollowingIngress)
		}
	}
	if groupId == "" {
		// If there is no service group for this endpoint, then the endpoint becomes the owning group.
		groupId = aggrEndpointId
	}

	// If the endpoint is not expandable or expanded then add the port if present.
	if !aggrEndpointExpandable || !aggrEndpointExpanded {
		log.Debugf("Group is not expanded or not expandable: %s; %s, %s, %#v", groupId, aggrEndpointId, nonAggrEndpointId, s.view.Expanded.Endpoints)

		if aggrEndpointPortId := idi.GetAggrEndpointPortID(); aggrEndpointPortId != "" {
			if _, ok := s.nodesMap[aggrEndpointPortId]; !ok {
				sel := s.selh.GetEndpointNodeSelectors(
					idi.GetAggrEndpointType(),
					endpoint.Namespace,
					endpoint.Name,
					endpoint.NameAggr,
					endpoint.Proto,
					endpoint.Port,
					idi.Direction,
				)
				s.nodesMap[aggrEndpointPortId] = trackedNode{
					Node: &v1.GraphNode{
						Type:      v1.GraphNodeTypePort,
						ID:        aggrEndpointPortId,
						ParentID:  aggrEndpointId,
						Port:      endpoint.Port,
						Protocol:  endpoint.Proto,
						Selectors: sel.ToNodeSelectors(),
					},
					Selectors: sel,
				}
				s.groupsMap[groupId].update(aggrEndpointPortId, false, false, false)
			}
			endpointId = aggrEndpointPortId
		} else {
			endpointId = aggrEndpointId
		}
		return
	}

	// The endpoint is expanded and expandable.
	if _, ok := s.nodesMap[nonAggrEndpointId]; !ok {
		sel := s.selh.GetEndpointNodeSelectors(
			idi.Endpoint.Type,
			endpoint.Namespace,
			endpoint.Name,
			endpoint.NameAggr,
			NoProto,
			NoPort,
			idi.Direction,
		)
		s.nodesMap[nonAggrEndpointId] = trackedNode{
			Node: &v1.GraphNode{
				Type:      idi.Endpoint.Type,
				ID:        nonAggrEndpointId,
				ParentID:  aggrEndpointId,
				Namespace: endpoint.Namespace,
				Name:      endpoint.Name,
				Selectors: sel.ToNodeSelectors(),
			},
			Selectors: sel,
		}
		s.groupsMap[groupId].update(nonAggrEndpointId, false, false, false)
	}

	if nonAggrEndpointPortId := idi.GetEndpointPortID(); nonAggrEndpointPortId != "" {
		if _, ok := s.nodesMap[nonAggrEndpointId]; !ok {
			sel := s.selh.GetEndpointNodeSelectors(
				idi.Endpoint.Type,
				endpoint.Namespace,
				endpoint.Name,
				endpoint.NameAggr,
				endpoint.Proto,
				endpoint.Port,
				idi.Direction,
			)
			s.nodesMap[nonAggrEndpointPortId] = trackedNode{
				Node: &v1.GraphNode{
					Type:      v1.GraphNodeTypePort,
					ID:        nonAggrEndpointPortId,
					ParentID:  nonAggrEndpointId,
					Port:      endpoint.Port,
					Protocol:  endpoint.Proto,
					Selectors: sel.ToNodeSelectors(),
				},
				Selectors: sel,
			}
			s.groupsMap[groupId].update(nonAggrEndpointPortId, false, false, false)
		}
		endpointId = nonAggrEndpointPortId
	} else {
		endpointId = nonAggrEndpointId
	}

	return
}

// getNodeInView determines which nodes are in view. This is returned as a Set of node IDs. This is then used to select
// the final set of nodes and edges for the service graph.
func (s *serviceGraphConstructionData) getNodesInView() set.Set {
	if s.view.Focus.isEmpty() {
		// In full view, so return all tracked nodes and edges. The calling code will handle this case separately.
		log.Debug("No view selected - return all nodes and edges")
		return nil
	}

	// Keep expanding until we have processed all groups that are in-view.  There are three parts to this expansion:
	// - Expand the in-focus nodes in both directions
	// - If connection direction is being following, carry on expanding ingress and egress directions outwards from
	//   in-focus nodes
	// - If a node connection is being explicitly followed, keep expanding until all expansion points are exhausted.

	log.Debug("Expanding nodes explicitly in view")
	groupsInView := set.New()
	expandIngress := set.New()
	expandEgress := set.New()
	expandFollowing := set.New()
	for id, gp := range s.groupsMap {
		if gp.isInFocus {
			log.Debugf("Expand ingress and egress for in-focus node: %s", id)
			groupsInView.Add(gp)
			expandIngress.Add(gp)
			expandEgress.Add(gp)
		}
	}

	// Expand in-Focus nodes in ingress Direction and possibly follow connection direction.
	for expandIngress.Len() > 0 {
		expandIngress.Iter(func(item interface{}) error {
			gp := item.(*trackedGroup)
			if gp.processedIngress {
				return set.RemoveItem
			}

			// Add ingress nodes for this group.
			gp.processedIngress = true
			gp.ingress.Iter(func(item interface{}) error {
				connectedGp := item.(*trackedGroup)
				log.Debugf("Including ingress expanded group: %s -> %s", connectedGp.id, gp.id)
				groupsInView.Add(connectedGp)
				if s.view.FollowConnectionDirection {
					expandIngress.Add(connectedGp)
				} else if connectedGp.isFollowingEgress || connectedGp.isFollowingIngress {
					log.Debugf("Following ingress and/or egress direction from: %s", connectedGp.id)
					expandFollowing.Add(connectedGp)
				}
				return nil
			})
			return set.RemoveItem
		})
	}

	// Expand in-Focus nodes in ingress Direction and possibly follow connection direction.
	for expandEgress.Len() > 0 {
		expandEgress.Iter(func(item interface{}) error {
			gp := item.(*trackedGroup)
			if gp.processedEgress {
				return set.RemoveItem
			}

			// Add egress nodes for this group.
			gp.processedEgress = true
			gp.egress.Iter(func(item interface{}) error {
				connectedGp := item.(*trackedGroup)
				log.Debugf("Including egress expanded group: %s -> %s", gp.id, connectedGp.id)
				groupsInView.Add(connectedGp)

				if s.view.FollowConnectionDirection {
					expandEgress.Add(connectedGp)
				} else if connectedGp.isFollowingEgress || connectedGp.isFollowingIngress {
					log.Debugf("Following ingress and/or egress direction from: %s", connectedGp.id)
					expandFollowing.Add(connectedGp)
				}
				return nil
			})
			return set.RemoveItem
		})
	}

	// Expand followed nodes.
	for expandFollowing.Len() > 0 {
		expandFollowing.Iter(func(item interface{}) error {
			gp := item.(*trackedGroup)
			if gp.isFollowingIngress && !gp.processedIngress {
				gp.processedIngress = true
				gp.ingress.Iter(func(item interface{}) error {
					followedGp := item.(*trackedGroup)
					log.Debugf("Following ingress from %s to %s", gp.id, followedGp.id)
					groupsInView.Add(followedGp)
					expandFollowing.Add(followedGp)
					return nil
				})
			}
			if gp.isFollowingEgress && !gp.processedEgress {
				gp.processedEgress = true
				gp.egress.Iter(func(item interface{}) error {
					followedGp := item.(*trackedGroup)
					log.Debugf("Following egress from %s to %s", gp.id, followedGp.id)
					groupsInView.Add(followedGp)
					expandFollowing.Add(followedGp)
					return nil
				})
			}
			return set.RemoveItem
		})
	}

	// Create the full set of nodes that are in view.
	nodes := set.New()
	groupsInView.Iter(func(item interface{}) error {
		gp := item.(*trackedGroup)
		nodes.Add(gp.id)
		nodes.AddAll(gp.children)
		return nil
	})

	// Log the full set of nodes in view.
	if log.IsLevelEnabled(log.DebugLevel) {
		nodes.Iter(func(item interface{}) error {
			id := item.(v1.GraphNodeID)
			log.Debugf("Including node: %s", id)
			return nil
		})
	}

	return nodes
}

// overlayEvents iterates through all the events and overlays them on the existing graph nodes. This never adds more
// nodes to the graph.
func (s *serviceGraphConstructionData) overlayEvents(nodesInView set.Set) {
	for _, event := range s.sgd.Events {
		log.Debugf("Checking event %#v", event)
		for _, ep := range event.Endpoints {
			log.Debugf("  - Checking event endpoint: %#v", ep)
			switch ep.Type {
			case v1.GraphNodeTypeService:
				fep := FlowEndpoint{
					Type:      ep.Type,
					Namespace: ep.Namespace,
				}
				sg := s.sgd.ServiceGroups.GetByService(v1.NamespacedName{
					Namespace: ep.Namespace, Name: ep.Name,
				})
				s.maybeOverlayEventID(nodesInView, event, fep, sg)
			default:
				sg := s.sgd.ServiceGroups.GetByEndpoint(ep)
				s.maybeOverlayEventID(nodesInView, event, ep, sg)
			}
		}
	}
}

// maybeOverlayEventID attempts to overlay an event on to one of the graph nodes. It applies the event to the top
// level parent containing the impacted endpoint. If there is no node visible the event is discarded.
func (s *serviceGraphConstructionData) maybeOverlayEventID(nodesInView set.Set, event Event, ep FlowEndpoint, sg *ServiceGroup) {
	// Determine which layer this endpoint might be part of.
	var layerName, layerNameEndpoint, layerNameAggrEndpoint, layerNameServiceGroup, layerNameNamespace string

	idi := IDInfo{
		Endpoint:     ep,
		ServiceGroup: sg,
		Direction:    "",
	}

	aggrEndpointId := idi.GetAggrEndpointID()
	endpointId := idi.GetEndpointID()
	if aggrEndpointId != "" {
		layerNameEndpoint = s.view.Layers.EndpointToLayer[aggrEndpointId]
		layerName = layerNameEndpoint
	}
	if layerName == "" && endpointId != "" {
		layerNameAggrEndpoint = s.view.Layers.EndpointToLayer[endpointId]
		layerName = layerNameAggrEndpoint
	}
	if layerName == "" && sg != nil {
		layerNameServiceGroup = s.view.Layers.ServiceGroupToLayer[sg]
		layerName = layerNameServiceGroup
	}
	if layerName == "" && ep.Namespace != "" {
		layerNameNamespace = s.view.Layers.NamespaceToLayer[ep.Namespace]
		layerName = layerNameNamespace
	}

	// Set the layer.
	idi.Layer = layerName

	// Check if the layer is in view, if so add the event to the layer.
	if layerName != "" {
		layerId := idi.GetLayerID()
		if _, ok := s.nodesMap[layerId]; ok && (nodesInView == nil || nodesInView.Contains(layerId)) {
			log.Debugf("  - Including event in node %s", layerId)
			s.nodesMap[layerId].Node.IncludeEvent(event.ID, event.Details)
			return
		}
	}

	// Check if the namespace is in view, if so add the event to the namespace.
	if ep.Namespace != "" {
		namespaceId := idi.GetNamespaceID()
		if _, ok := s.nodesMap[namespaceId]; ok && (nodesInView == nil || nodesInView.Contains(namespaceId)) {
			log.Debugf("  - Including event in node %s", namespaceId)
			s.nodesMap[namespaceId].Node.IncludeEvent(event.ID, event.Details)
			return
		}
	}

	// Check if service group is in view, if so add the event to the service group.
	if sg != nil {
		if _, ok := s.nodesMap[sg.ID]; ok && (nodesInView == nil || nodesInView.Contains(sg.ID)) {
			log.Debugf("  - Including event in node %s", sg.ID)
			s.nodesMap[sg.ID].Node.IncludeEvent(event.ID, event.Details)
			return
		}
	}
	// Check if endpoint is in view, if so add the event to the endpoint.
	if endpointId != "" {
		if _, ok := s.nodesMap[endpointId]; ok && (nodesInView == nil || nodesInView.Contains(endpointId)) {
			log.Debugf("  - Including event in node %s", endpointId)
			s.nodesMap[endpointId].Node.IncludeEvent(event.ID, event.Details)
			return
		}
	}
	if aggrEndpointId != "" {
		if _, ok := s.nodesMap[aggrEndpointId]; ok && (nodesInView == nil || nodesInView.Contains(aggrEndpointId)) {
			log.Debugf("  - Including event in node %s", aggrEndpointId)
			s.nodesMap[aggrEndpointId].Node.IncludeEvent(event.ID, event.Details)
			return
		}
	}
	log.Debug("  - No matching node in graph")
}
