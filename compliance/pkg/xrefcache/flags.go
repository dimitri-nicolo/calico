// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import "github.com/projectcalico/calico/compliance/pkg/syncer"

// The set of events that may be registered for. Each event type is only valid for a sub-set of resources.
//
// Some of the event type flags are re-used as cache entry flags to store the equivalent boolean value. For example
// the event type EventProtectedIngress is identical to the cache entry flag FlagProtectedIngress. This flag indicates
// whether the resource has ingress protection or not. Using the same set of values makes it easy to use bit-wise
// operations to track changes and to notify of the equivalent events:
// -  If a policy has cache entry flags X which are then updated to Y
// -  The event flags correspond one-to-one with the cache flags, so therefore the event flags associated with
//    configuration change X->Y can be determined using syncer.UpdateType(X ^ Y).
//
// Events that do not correspond to boolean configuration values do not have equivalents defined for the CacheEntry
// values.
//
// The boolean values are chosen where possible to be additive so that if a resource depends on multiple other
// resources, then we can simply OR the values together to determine effective configuration. For example:
// - a Pod has multiple applied Policies
// - if any of those Policies has ProtectedIngress then the Pod as a whole has ProtectedIngress.
// If the field was "UnprotectedIngress" then we'd need to AND the values of the Policies together to determine the pods
// UnprotectedIngress value.
const (
	// Valid for Policies, pods and host endpoints
	EventProtectedIngress syncer.UpdateType = 1 << iota
	EventProtectedEgress
	// Valid for network sets
	EventInternetExposed
	// Valid for Policies, pods and host endpoints
	EventInternetExposedIngress
	EventInternetExposedEgress
	// Valid for Policies, pods and host endpoints
	EventOtherNamespaceExposedIngress
	EventOtherNamespaceExposedEgress
	// Valid for pods
	EventEnvoyEnabled

	// ----- Non boolean configuration values -----
	// The following event flags do not have equivalent CacheEntry flags.
	EventPolicyRuleSelectorMatchStarted
	EventPolicyRuleSelectorMatchStopped
	EventPolicyMatchStarted
	EventPolicyMatchStopped
	EventNetsetMatchStarted
	EventNetsetMatchStopped
	EventServiceAdded
	EventServiceDeleted

	// ----- Added by the generic cache processing -----
	EventResourceAdded
	EventResourceModified
	EventResourceDeleted
	EventInScope

	// ----- Valid for policies -----
	EventNodeAssigned
	EventNodeRemoved
	EventEndpointMatchStarted
	EventEndpointMatchStopped
)

const (
	// For a cache to indicate that the settings have changed, it queues itself for recalculation. However, not all
	// updates require a recalculation since the data may be updated as part of another callback. The following events
	// do not actually require a recalculation, and therefore the recalculation step can be omitted if the update
	// flags only belong in this group.
	EventsNotRequiringRecalculation = EventServiceAdded | EventServiceDeleted | EventInScope
)

// The CacheEntryFlags are the set of flags maintained by each cache entry. Only certain flags are valid for certain
// resource types.
//
// See note above regarding the relationship between the event flags and the cache entry flags.
type CacheEntryFlags syncer.UpdateType

const (
	CacheEntryProtectedIngress             = CacheEntryFlags(EventProtectedIngress)
	CacheEntryProtectedEgress              = CacheEntryFlags(EventProtectedEgress)
	CacheEntryInternetExposed              = CacheEntryFlags(EventInternetExposed)
	CacheEntryInternetExposedIngress       = CacheEntryFlags(EventInternetExposedIngress)
	CacheEntryInternetExposedEgress        = CacheEntryFlags(EventInternetExposedEgress)
	CacheEntryOtherNamespaceExposedIngress = CacheEntryFlags(EventOtherNamespaceExposedIngress)
	CacheEntryOtherNamespaceExposedEgress  = CacheEntryFlags(EventOtherNamespaceExposedEgress)
	CacheEntryEnvoyEnabled                 = CacheEntryFlags(EventEnvoyEnabled)
	CacheEntryNodeAssigned                 = CacheEntryFlags(EventNodeAssigned)
	CacheEntryNodeRemoved                  = CacheEntryFlags(EventNodeRemoved)
	CacheEntryEndpointMatchStarted         = CacheEntryFlags(EventEndpointMatchStarted)
	CacheEntryEndpointMatchStopped         = CacheEntryFlags(EventEndpointMatchStopped)
)

const (
	CacheEntryFlagsEndpoint = CacheEntryProtectedIngress |
		CacheEntryProtectedEgress |
		CacheEntryInternetExposedIngress |
		CacheEntryInternetExposedEgress |
		CacheEntryOtherNamespaceExposedIngress |
		CacheEntryOtherNamespaceExposedEgress |
		CacheEntryEnvoyEnabled

	CacheEntryFlagsNetworkPolicy = CacheEntryProtectedIngress |
		CacheEntryProtectedEgress |
		CacheEntryInternetExposedIngress |
		CacheEntryInternetExposedEgress |
		CacheEntryOtherNamespaceExposedIngress |
		CacheEntryOtherNamespaceExposedEgress |
		CacheEntryNodeAssigned |
		CacheEntryNodeRemoved |
		CacheEntryEndpointMatchStarted |
		CacheEntryEndpointMatchStopped

	CacheEntryFlagsNetworkSets = CacheEntryInternetExposed
)
