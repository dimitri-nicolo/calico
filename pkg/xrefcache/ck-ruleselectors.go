// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

// This file implements a RuleSelector cache. This is a pseudo-resource, implemented to allow rule selectors to be
// managed and accessed as if they were a resource type. Creation of these pseudo-resources is managed via the
// NetworkPolicy cache which tracks which selectors need to be created/deleted based on policy configuration events.

var (
	// Internal resource kind to encapsulate a selector. This is a bit of a hack since our label-selector interface
	// assumes all links are resource types, however we want to track selector/netset links so we create a fake
	// selector kind.
	kindSelector = schema.GroupVersionKind{
		Kind:  "selector",
		Group: "internal",
	}

	// The network policy cache is populated by both Kubernetes and Calico policy types. Include kindSelector in here so
	// the queued recalculation processing knows where to send those updates.
	KindsNetworkPolicyRuleSelectors = []schema.GroupVersionKind{
		kindSelector,
	}
)

func selectorIDToSelector(id resources.ResourceID) string {
	return id.Name
}

func selectorToSelectorID(sel string) resources.ResourceID {
	return resources.ResourceID{
		GroupVersionKind: kindSelector,
		NameNamespace: resources.NameNamespace{
			Name: sel,
		},
	}
}

// CacheEntryNetworkPolicyRuleSelector is a cache entry in the NetworkPolicyRuleSelector cache. Each entry implements
// the CacheEntry interface.
type CacheEntryNetworkPolicyRuleSelector struct {
	// The versioned policy resource.
	VersionedResource

	// The effective NetworkSet CacheEntryFlags (i.e. the combination of the set of selected NetworkSets for this
	// selector.
	NetworkSetFlags CacheEntryFlags

	// Internally managed references.
	networkSets resources.Set
	policies    resources.Set

	// --- Internal data ---
	cacheEntryCommon
}

// getVersionedResource implements the CacheEntry interface.
func (c *CacheEntryNetworkPolicyRuleSelector) getVersionedResource() VersionedResource {
	return c.VersionedResource
}

// setVersionedResource implements the CacheEntry interface.
func (c *CacheEntryNetworkPolicyRuleSelector) setVersionedResource(r VersionedResource) {
	c.VersionedResource = r
}

// newNetworkPolicyRuleSelectorsEngine creates a new engine used for the NetworkPolicy cache.
func newNetworkPolicyRuleSelectorsEngine() resourceCacheEngine {
	return &networkPolicyRuleSelectorsEngine{}
}

// networkPolicyRuleSelectorsEngine implements the resourceCacheEngine interface for the NetworkPolicy rule selector.
type networkPolicyRuleSelectorsEngine struct {
	engineCache
}

// register implements the resourceCacheEngine interface.
func (c *networkPolicyRuleSelectorsEngine) register(cache engineCache) {
	c.engineCache = cache

	// Register with the netset label selectors for notification of match start/stops.
	c.NetworkSetLabelSelector().RegisterCallbacks(c.kinds(), c.netsetMatchStarted, c.netsetMatchStopped)
	c.NetworkPolicyRuleSelectorManager().RegisterCallbacks(c.policyMatchStarted, c.policyMatchStopped)
}

// register implements the resourceCacheEngine interface.
func (c *networkPolicyRuleSelectorsEngine) kinds() []schema.GroupVersionKind {
	return KindsNetworkPolicy
}

// newCacheEntry implements the resourceCacheEngine interface.
func (c *networkPolicyRuleSelectorsEngine) newCacheEntry() CacheEntry {
	return &CacheEntryNetworkPolicyRuleSelector{
		networkSets: resources.NewSet(),
		policies:    resources.NewSet(),
	}
}

// resourceAdded implements the resourceCacheEngine interface.
func (c *networkPolicyRuleSelectorsEngine) resourceAdded(id resources.ResourceID, entry CacheEntry) {
	// Just call through to our update processsing.
	c.resourceUpdated(id, entry, nil)
}

// resourceUpdated implements the resourceCacheEngine interface.
func (c *networkPolicyRuleSelectorsEngine) resourceUpdated(id resources.ResourceID, entry CacheEntry, prev VersionedResource) syncer.UpdateType {
	c.NetworkSetLabelSelector().UpdateSelector(id, selectorIDToSelector(id))
	return 0
}

// resourceDeleted implements the resourceCacheEngine interface.
func (c *networkPolicyRuleSelectorsEngine) resourceDeleted(id resources.ResourceID, res CacheEntry) {
	c.NetworkSetLabelSelector().DeleteSelector(id)
}

// recalculate implements the resourceCacheEngine interface.
func (c *networkPolicyRuleSelectorsEngine) recalculate(id resources.ResourceID, entry CacheEntry) syncer.UpdateType {
	sel := entry.(*CacheEntryNetworkPolicyRuleSelector)

	// Store and clear the effective set of Netset flags.
	oldFlags := sel.NetworkSetFlags
	sel.NetworkSetFlags = 0
	sel.networkSets.Iter(func(nsid resources.ResourceID) error {
		netset := c.GetFromXrefCache(nsid)
		if netset == nil {
			log.Errorf("Cannot find referenced NetworkSet in cache when recalculating rule selector flags")
			return nil
		}
		sel.NetworkSetFlags |= netset.(*CacheEntryCalicoNetworkSet).Flags
		return nil
	})

	changed := syncer.UpdateType(oldFlags ^ sel.NetworkSetFlags)

	//TODO(rlb): This should really be done by registered callbacks
	if changed != 0 {
		// The effective settings of the NetworkSet flags for this rule selector have changed. Trigger a recalculation
		// of affected policies.
		sel.policies.Iter(func(pid resources.ResourceID) error {
			c.QueueRecalculation(pid, nil, changed)
			return nil
		})
	}
	return changed
}

// convertToVersioned implements the resourceCacheEngine interface.
func (c *networkPolicyRuleSelectorsEngine) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	return nil, nil
}

func (c *networkPolicyRuleSelectorsEngine) netsetMatchStarted(ns, sel resources.ResourceID) {
	s, ok := c.GetFromOurCache(sel).(*CacheEntryNetworkPolicyRuleSelector)
	if !ok {
		log.Errorf("Match started on selector, but selector is not in cache: %s matches %s", sel, ns)
		return
	}
	s.networkSets.Add(ns)
	c.QueueRecalculation(sel, nil, EventNetsetMatchStarted)
}

func (c *networkPolicyRuleSelectorsEngine) netsetMatchStopped(ns, sel resources.ResourceID) {
	s, ok := c.GetFromOurCache(sel).(*CacheEntryNetworkPolicyRuleSelector)
	if !ok {
		log.Errorf("Match stopped on selector, but selector is not in cache: %s matches %s", sel, ns)
		return
	}
	s.networkSets.Discard(ns)
	c.QueueRecalculation(sel, nil, EventNetsetMatchStopped)
}

func (c *networkPolicyRuleSelectorsEngine) policyMatchStarted(pol, sel resources.ResourceID) {
	s, ok := c.GetFromOurCache(sel).(*CacheEntryNetworkPolicyRuleSelector)
	if !ok {
		log.Errorf("Match started on selector, but selector is not in cache: %s matches %s", sel, pol)
		return
	}
	s.policies.Add(pol)
}

func (c *networkPolicyRuleSelectorsEngine) policyMatchStopped(pol, sel resources.ResourceID) {
	s, ok := c.GetFromOurCache(sel).(*CacheEntryNetworkPolicyRuleSelector)
	if !ok {
		log.Errorf("Match stopped on selector, but selector is not in cache: %s matches %s", sel, pol)
		return
	}
	s.policies.Discard(pol)
}
