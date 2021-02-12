// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

// GraphView provides the configuration for what is included in the service graph response.
//
// The flows are aggregated based on the layers and expanded nodes defined in this view. The graph is then pruned
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
//                                                 A will be included iff the egress connections for B are being followed
//                                                 E will be included iff the ingress connections for D are being followed
type GraphView struct {
	// The view is the set of nodes that are the focus of the graph. All nodes returned by the service graph query
	// will be connected to at least one of these nodes. If this is empty, then all nodes will be returned.
	Focus []string `json:"focus,omitempty"`

	// Expanded nodes.
	Expanded []string `json:"expanded,omitempty"`

	// Followed nodes. These are nodes on the periphery of the graph that we are follow further out of the scope of the
	// graph focus.
	FollowedEgress  []string `json:"followed_egress,omitempty"`
	FollowedIngress []string `json:"followed_ingress,omitempty"`

	// The layers - this is the set of nodes that will be aggregated into a single layer. If a layer is also
	// flagged as "expanded" then the nodes will not be aggregated into the layer, but the nodes will be flagged as
	// being contained in the layer.
	Layers Layers `json:"layers,omitempty"`
}

type Layers map[string][]string
