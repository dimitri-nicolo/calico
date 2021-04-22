package servicegraph

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/set"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

// This file provides the final graph construction from a set of correlated (time-series) flows and the parsed view
// IDs.
//
// The flows are aggregated based on the layers and expanded nodes defined in the view. The graph is then pruned
// based on the focus and followed-nodes. A graph node is included if any of the following is true:
// - the node (or one of its child nodes) is in-focus
// - the node (or one of its child nodes) is connected directly to an in-focus node (in either connection Direction)
// - the node (or one of its child nodes) is connected indirectly to an in-focus node, respecting the Direction
//   of the connection (*)
// - the node (or one of its child nodes) is directly connected to an "included" node whose connections are being
//   explicitly "followed" in the appropriate connection Direction (*)
//
// (*) Suppose you have nodes A, B, C, D, E; C is directly in focus
//     If connections are: A-->B-->C-->D-->E then: A, B, C, D and E will all be included in the view.
//     If connections are: A<--B-->C-->D<--E then: B, C and D will be included in the view, and
//                                                 A will be included if the egress connections for B are being followed
//                                                 E will be included if the ingress connections for D are being followed

// GetServiceGraphResponse calculates the service graph from the flow data and parsed view ids.
func GetServiceGraphResponse(f *FilteredTimeSeriesFlowData, v *ParsedViewIDs) (*v1.ServiceGraphResponse, error) {
	sgr := &v1.ServiceGraphResponse{
		// Response should include the time range actually used to perform these queries.
		TimeIntervals: f.TimeIntervals,
	}
	s := newServiceGraphConstructor(f, v)

	// Iterate through the flows to track the nodes and edges.
	for i := range s.flowData.FilteredFlows {
		if err := s.trackFlow(&s.flowData.FilteredFlows[i]); err != nil {
			log.WithError(err).WithField("flow", s.flowData.FilteredFlows[i]).Errorf("Unable to process flow")
			continue
		}
	}

	getGroupNode := func(id string) *v1.GraphNode {
		n := s.nodesMap[id]
		for n.ParentID != "" {
			n = s.nodesMap[n.ParentID]
		}
		return n
	}

	// Determine which nodes are in view from the tracked data. If there is no view, this will return nil indicating
	// all nodes and edges should be included.
	nodesInView := s.getNodesInView()

	// Copy across edges that are in view, and update the nodes to indicate whether we are truncating the graph (i.e.
	// that the graph can be followed along it's ingress or egress connections).
	for id, edge := range s.edgesMap {
		sourceInFocus := nodesInView == nil || nodesInView.Contains(id.SourceNodeID)
		destInFocus := nodesInView == nil || nodesInView.Contains(id.DestNodeID)
		if sourceInFocus && destInFocus {
			sgr.Edges = append(sgr.Edges, *edge)
		} else if sourceInFocus {
			// Destination is not in Focus, but this means the egress can be Expanded for the source node. Mark this
			// on the group rather than the endpoint.
			getGroupNode(id.SourceNodeID).FollowEgress = true
		} else if destInFocus {
			// Source is not in Focus, but this means the ingress can be Expanded for the dest node. Mark this
			// on the group rather than the endpoint.
			getGroupNode(id.DestNodeID).FollowIngress = true
		}
	}

	if nodesInView == nil {
		// No view, copy all nodes.
		sgr.Nodes = make([]v1.GraphNode, 0, len(s.nodesMap))
		for _, n := range s.nodesMap {
			sgr.Nodes = append(sgr.Nodes, *n)
		}
	} else {
		// We have a view so copy across the nodes in the view.
		sgr.Nodes = make([]v1.GraphNode, 0, nodesInView.Len())
		nodesInView.Iter(func(item interface{}) error {
			id := item.(string)
			sgr.Nodes = append(sgr.Nodes, *s.nodesMap[id])
			return nil
		})
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
	id                 string
	ingress            set.Set
	egress             set.Set
	isInFocus          bool
	isFollowingEgress  bool
	isFollowingIngress bool
	children           []string
	processedIngress   bool
	processedEgress    bool
}

// newTrackedGroup creates a new trackedGroup, setting the focus/following info.
func newTrackedGroup(id string, isInFocus, isFollowingEgress, isFollowigIngress bool) *trackedGroup {
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
func (t *trackedGroup) update(child string, isInFocus, isFollowingEgress, isFollowingIngress bool) {
	t.children = append(t.children, child)
	t.isInFocus = t.isInFocus || isInFocus
	t.isFollowingEgress = t.isFollowingEgress || isFollowingEgress
	t.isFollowingIngress = t.isFollowingIngress || isFollowingIngress
}

// serviceGraphConstructionData is the transient data used to construct the final service graph.
type serviceGraphConstructionData struct {
	// The set of tracked groups keyed of the group node ID.
	groupsMap map[string]*trackedGroup

	// The full set of graph nodes keyed off the node ID.
	nodesMap map[string]*v1.GraphNode

	// The full set of graph edges keyed off the edge ID.
	edgesMap map[v1.GraphEdgeID]*v1.GraphEdge

	// The supplied flow data.
	flowData *FilteredTimeSeriesFlowData

	// The supplied view data.
	view *ParsedViewIDs
}

// newServiceGraphConstructor intializes a new serviceGraphConstructionData.
func newServiceGraphConstructor(f *FilteredTimeSeriesFlowData, v *ParsedViewIDs) *serviceGraphConstructionData {
	return &serviceGraphConstructionData{
		groupsMap: make(map[string]*trackedGroup),
		nodesMap:  make(map[string]*v1.GraphNode),
		edgesMap:  make(map[v1.GraphEdgeID]*v1.GraphEdge),
		flowData:  f,
		view:      v,
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
	srcGp, srcEp, _ := s.trackNodes(flow.Edge.Source, nil, DirectionEgress)
	dstGp, dstEp, servicePortDst := s.trackNodes(flow.Edge.Dest, flow.Edge.ServicePort, DirectionIngress)

	// Include any aggregated port proto info in either the group or the endpoint.  Include the service in the group if
	// it is not expanded.
	if dstGp != "" && dstEp == "" && servicePortDst == "" {
		if flow.AggregatedProtoPorts != nil {
			s.nodesMap[dstGp].IncludeAggregatedProtoPorts(flow.AggregatedProtoPorts)
		}
		if flow.Edge.ServicePort != nil {
			s.nodesMap[dstGp].IncludeService(flow.Edge.ServicePort.NamespacedName)
		}
	} else if dstEp != "" {
		if flow.AggregatedProtoPorts != nil {
			s.nodesMap[dstEp].IncludeAggregatedProtoPorts(flow.AggregatedProtoPorts)
		}
	}

	// If the source and dest group are the same then do not add edges, instead add the traffic stats to the group node
	// and just return the group node.
	if srcGp == dstGp {
		node := s.nodesMap[srcGp]
		node.Include(flow.TrafficStats)
		return nil
	}

	// Stitch together the source and dest nodes going via the service if present.
	if servicePortDst != "" {
		id := v1.GraphEdgeID{
			SourceNodeID: srcEp,
			DestNodeID:   servicePortDst,
		}
		log.Debugf("Tracking: %s", id)
		if edge, ok := s.edgesMap[id]; ok {
			edge.Include(flow.TrafficStats)
		} else {
			s.edgesMap[id] = &v1.GraphEdge{
				ID:           id,
				TrafficStats: flow.TrafficStats,
				Selectors:    v1.GetEdgeSelectors(s.nodesMap[srcEp].Selectors, s.nodesMap[servicePortDst].Selectors),
			}
		}

		id = v1.GraphEdgeID{
			SourceNodeID: servicePortDst,
			DestNodeID:   dstEp,
		}
		log.Debugf("Tracking: %s", id)
		if edge, ok := s.edgesMap[id]; ok {
			edge.Include(flow.TrafficStats)
		} else {
			s.edgesMap[id] = &v1.GraphEdge{
				ID:           id,
				TrafficStats: flow.TrafficStats,
				Selectors:    v1.GetEdgeSelectors(s.nodesMap[servicePortDst].Selectors, s.nodesMap[dstEp].Selectors),
			}
		}
	} else {
		id := v1.GraphEdgeID{
			SourceNodeID: srcEp,
			DestNodeID:   dstEp,
		}
		log.Debugf("Tracking: %s", id)
		if edge, ok := s.edgesMap[id]; ok {
			edge.Include(flow.TrafficStats)
		} else {
			s.edgesMap[id] = &v1.GraphEdge{
				ID:           id,
				TrafficStats: flow.TrafficStats,
				Selectors:    v1.GetEdgeSelectors(s.nodesMap[srcEp].Selectors, s.nodesMap[dstEp].Selectors),
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
) (groupId, endpointId, serviceId string) {
	// Determine if this endpoint is in a layer - most granular wins.
	var sg *ServiceGroup
	if svc != nil {
		sg = s.flowData.ServiceGroups.GetByService(svc.NamespacedName)
	} else {
		sg = s.flowData.ServiceGroups.GetByEndpoint(endpoint)
	}
	epg := GetServiceGroupFlowEndpointKey(endpoint)
	var layerName, layerNameEndpoint, layerNameServiceGroup, layerNameNamespace string
	if epg != nil {
		layerNameEndpoint = s.view.Layers.EndpointToLayer[*epg]
		layerName = layerNameEndpoint
	}
	if layerName == "" && sg != nil {
		layerNameServiceGroup = s.view.Layers.ServiceGroupToLayer[sg]
		layerName = layerNameServiceGroup
	}
	if layerName == "" && endpoint.Namespace != "" {
		layerNameNamespace = s.view.Layers.NamespaceToLayer[endpoint.Namespace]
		layerName = layerNameNamespace
	}

	// Create an ID handler.
	idi := IDInfo{
		Endpoint:  endpoint,
		Services:  nil,
		Layer:     layerName,
		Direction: dir,
	}
	if svc != nil {
		idi.Service = *svc
	}
	if sg != nil {
		idi.Services = sg.Services
	}

	// If the layer is not Expanded then return the layer node.
	if layerName != "" && !s.view.Expanded.Layers[layerName] {
		id := idi.GetLayerID()
		if _, ok := s.nodesMap[id]; !ok {
			s.nodesMap[id] = &v1.GraphNode{
				ID:         id,
				Type:       v1.GraphNodeTypeLayer,
				Name:       layerName,
				Expandable: true,
				Selectors:  GetLayerNodeSelectors(layerName, s.view),
			}
			s.groupsMap[id] = newTrackedGroup(
				id, s.view.Focus.Layers[layerName],
				s.view.FollowedEgress.Layers[layerName],
				s.view.FollowedIngress.Layers[layerName],
			)
		}
		return id, id, ""
	}

	// If the namespace is not Expanded then return the namespace node. If the namespace is part of an Expanded layer
	// then include the layer name.
	if endpoint.Namespace != "" && !s.view.Expanded.Namespaces[endpoint.Namespace] {
		id := idi.GetNamespaceID()
		if _, ok := s.nodesMap[id]; !ok {
			s.nodesMap[id] = &v1.GraphNode{
				ID:         id,
				Type:       v1.GraphNodeTypeNamespace,
				Name:       endpoint.Namespace,
				Layer:      layerNameNamespace,
				Expandable: true,
				Selectors:  GetNamespaceNodeSelectors(endpoint.Namespace),
			}
			s.groupsMap[id] = newTrackedGroup(
				id,
				s.view.Focus.Namespaces[endpoint.Namespace] || s.view.Focus.Layers[layerName],
				s.view.FollowedEgress.Namespaces[endpoint.Namespace] || s.view.FollowedEgress.Layers[layerName],
				s.view.FollowedIngress.Namespaces[endpoint.Namespace] || s.view.FollowedIngress.Layers[layerName],
			)
		}
		return id, id, ""
	}

	// If the service group is not Expanded then return the service group node. If the service group is part of an
	// Expanded layer then include the layer name.
	var parentId string
	if sg != nil {
		id := sg.ID
		if _, ok := s.nodesMap[id]; !ok {
			s.nodesMap[id] = &v1.GraphNode{
				ID:         id,
				Type:       v1.GraphNodeTypeServiceGroup,
				Namespace:  sg.Namespace,
				Name:       sg.Name,
				Layer:      layerNameServiceGroup,
				Expandable: true,
				Selectors:  GetServiceGroupNodeSelectors(sg),
			}
			s.groupsMap[id] = newTrackedGroup(
				id,
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
			return id, id, ""
		}

		groupId = id
		parentId = id

		// If there is a service we will need to add that node too, and return the ID.
		if svc != nil {
			serviceId := idi.GetServicePortID()
			if _, ok := s.nodesMap[id]; !ok {
				s.nodesMap[id] = &v1.GraphNode{
					ID:          serviceId,
					Type:        v1.GraphNodeTypeServicePort,
					Namespace:   svc.Namespace,
					Name:        svc.Name,
					ServicePort: svc.Port,
					ParentID:    parentId,
					Selectors:   GetServicePortNodeSelectors(*svc),
				}
				s.groupsMap[groupId].update(serviceId, false, false, false)
			}
		}
	}

	// Combine the aggregated endpoint node - this should  always be available for a flow.
	id := idi.GetAggrEndpointID()
	if _, ok := s.nodesMap[id]; !ok {
		s.nodesMap[id] = &v1.GraphNode{
			ID:         id,
			Type:       idi.GetAggrEndpointType(),
			Namespace:  endpoint.Namespace,
			Name:       endpoint.NameAggr,
			Layer:      layerNameEndpoint,
			Expandable: true,
			Selectors: GetEndpointNodeSelectors(idi.GetAggrEndpointType(), endpoint.Namespace, endpoint.NameAggr,
				NoProto, NoPort, idi.Direction),
		}

		var isInFocus, isFollowingEgress, isFollowingIngress bool
		if epg != nil {
			isInFocus = s.view.Focus.Endpoints[*epg] ||
				s.view.Focus.Namespaces[endpoint.Namespace] ||
				s.view.Focus.Layers[layerName]
			isFollowingEgress = s.view.FollowedEgress.Endpoints[*epg] ||
				s.view.FollowedEgress.Namespaces[endpoint.Namespace] ||
				s.view.FollowedEgress.Layers[layerName]
			isFollowingIngress = s.view.FollowedIngress.Endpoints[*epg] ||
				s.view.FollowedIngress.Namespaces[endpoint.Namespace] ||
				s.view.FollowedIngress.Layers[layerName]
		}

		if groupId == "" {
			// The endpoint is also the group, so track it.
			s.groupsMap[id] = newTrackedGroup(
				id,
				isInFocus,
				isFollowingEgress,
				isFollowingIngress,
			)
		} else {
			// The endpoint is not the group. Update the view details for the group and include the endpoint as
			// a child.
			s.groupsMap[groupId].update(id, isInFocus, isFollowingEgress, isFollowingIngress)
		}
	}
	parentId = id
	if groupId == "" {
		// If there is no service group for this endpoint, then the endpoint becomes the owning group.
		groupId = id
	}

	// If the endpoint is not Expanded, or is not expandable then add the port if present.
	nonAggEndpointId := idi.GetEndpointID()
	if epg == nil || !s.view.Expanded.Endpoints[*epg] || nonAggEndpointId == "" {
		if id := idi.GetAggrEndpointPortID(); id != "" {
			if _, ok := s.nodesMap[id]; !ok {
				s.nodesMap[id] = &v1.GraphNode{
					ID:       id,
					Type:     v1.GraphNodeTypePort,
					Port:     endpoint.Port,
					Protocol: endpoint.Proto,
					ParentID: parentId,
					Selectors: GetEndpointNodeSelectors(
						idi.GetAggrEndpointType(), endpoint.Namespace, endpoint.NameAggr,
						endpoint.Proto, endpoint.Port, idi.Direction,
					),
				}
				s.groupsMap[groupId].update(id, false, false, false)
			}
			endpointId = id
		} else {
			endpointId = parentId
		}
		return
	}

	if _, ok := s.nodesMap[nonAggEndpointId]; !ok {
		s.nodesMap[id] = &v1.GraphNode{
			ID:        nonAggEndpointId,
			Type:      idi.Endpoint.Type,
			Namespace: endpoint.Namespace,
			Name:      endpoint.Name,
			ParentID:  parentId,
			Selectors: GetEndpointNodeSelectors(
				idi.Endpoint.Type, endpoint.Namespace, endpoint.NameAggr, NoProto, NoPort, idi.Direction,
			),
		}
		s.groupsMap[groupId].update(nonAggEndpointId, false, false, false)
	}
	parentId = nonAggEndpointId

	if id := idi.GetEndpointPortID(); id != "" {
		if _, ok := s.nodesMap[nonAggEndpointId]; !ok {
			s.nodesMap[id] = &v1.GraphNode{
				ID:       id,
				Type:     v1.GraphNodeTypePort,
				Port:     endpoint.Port,
				Protocol: endpoint.Proto,
				ParentID: parentId,
				Selectors: GetEndpointNodeSelectors(
					idi.Endpoint.Type, endpoint.Namespace, endpoint.NameAggr,
					endpoint.Proto, endpoint.Port, idi.Direction,
				),
			}
			s.groupsMap[groupId].update(id, false, false, false)
		}
		endpointId = id
	} else {
		endpointId = parentId
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

	// Track which groups are in view.
	log.Debug("Calculating nodes in view")
	groupsInView := set.New()
	expandIngress := set.New()
	expandEgress := set.New()
	followed := set.New()
	for id, gp := range s.groupsMap {
		// Everything should be connected to directly in Focus nodes.
		if gp.isInFocus {
			log.Debugf("Group is in view: %s", id)
			if !gp.processedIngress {
				expandIngress.Add(gp)
			}
			if !gp.processedEgress {
				expandEgress.Add(gp)
			}
			groupsInView.Add(gp)
		}
	}

	// Keep expanding until we have processed all groups that are in-view.
	// Expand in-Focus nodes in ingress Direction.
	for expandIngress.Len() > 0 {
		expandIngress.Iter(func(item interface{}) error {
			gp := item.(*trackedGroup)
			groupsInView.Add(gp)
			gp.processedIngress = true
			log.Debugf("Adding group to view: %s", gp.id)
			gp.ingress.Iter(func(item interface{}) error {
				connectedGp := item.(*trackedGroup)
				if !connectedGp.processedIngress {
					log.Debugf("Expanding ingress to: %s", connectedGp.id)
					expandIngress.Add(connectedGp)
				}
				return nil
			})
			// If the egress is being followed add it to the followed set.
			if gp.isFollowingEgress {
				gp.egress.Iter(func(item interface{}) error {
					followedGp := item.(*trackedGroup)
					log.Debugf("Following egress to: %s", followedGp.id)
					followed.Add(followedGp)
					return nil
				})
			}
			return set.RemoveItem
		})
	}

	// Expand in-Focus nodes in egress Direction.
	for expandEgress.Len() > 0 {
		expandEgress.Iter(func(item interface{}) error {
			gp := item.(*trackedGroup)
			groupsInView.Add(gp)
			gp.processedEgress = true
			log.Debugf("Adding group to view: %s", gp.id)
			gp.egress.Iter(func(item interface{}) error {
				connectedGp := item.(*trackedGroup)
				if !connectedGp.processedEgress {
					log.Debugf("Expanding egress to: %s", connectedGp.id)
					expandEgress.Add(connectedGp)
				}
				return nil
			})
			// If the ingress is being followed add it to the followed set.
			if gp.isFollowingIngress {
				gp.ingress.Iter(func(item interface{}) error {
					followedGp := item.(*trackedGroup)
					log.Debugf("Following ingress to: %s", followedGp.id)
					followed.Add(followedGp)
					return nil
				})
			}
			return set.RemoveItem
		})
	}

	// Expand followed nodes. These nodes
	for followed.Len() > 0 {
		followed.Iter(func(item interface{}) error {
			gp := item.(*trackedGroup)
			groupsInView.Add(gp)
			log.Debugf("Adding group to view: %s", gp.id)
			if gp.isFollowingIngress && !gp.processedIngress {
				gp.processedIngress = true
				gp.ingress.Iter(func(item interface{}) error {
					followedGp := item.(*trackedGroup)
					log.Debugf("Following ingress to: %s", followedGp.id)
					followed.Add(followedGp)
					return nil
				})
			}
			if gp.isFollowingEgress && !gp.processedEgress {
				gp.processedEgress = true
				gp.egress.Iter(func(item interface{}) error {
					followedGp := item.(*trackedGroup)
					log.Debugf("Following egress to: %s", followedGp.id)
					followed.Add(followedGp)
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
			id := item.(string)
			log.Debugf("Including node: %s", id)
			return nil
		})
	}

	return nodes
}
