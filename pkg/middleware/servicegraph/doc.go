// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

// Service Graph
//
// This middleware provides an API for returning flow related data aggregated into configurable aggregation
// groups to make the data more consumable. The graph automatically handles grouping of related sets of services
// and Endpoints, aggregation of user-configurable Layers of arbitrary nodes, Namespace and service group
// aggregation, expansion of replica sets when the flow logs has pod-level details.
//
// Layers
//
// Aggregation is performed in a hierarchical way depending on requested configuration.  The order of hierarchy is:
//
// - Layer
// - Namespace
// - AggregatedEndpoint
// - Endpoint
// - Port
//
// Not all of these are always applicable. If not applicable the parent/child hierarchy will skip over the aggregation
// level.
//
// By default, everything is aggregated at the Namespace level, and into Layers (if defined).
//
// See comment at the top of each file for additional details.
