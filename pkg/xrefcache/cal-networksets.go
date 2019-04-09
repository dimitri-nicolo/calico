// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"

	"github.com/tigera/compliance/pkg/internet"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

var (
	KindsNetworkSet = []metav1.TypeMeta{
		resources.TypeCalicoGlobalNetworkSets,
	}
)

// VersionedNetworkSetResource is an extension to the VersionedResource interface with some NetworkSet specific
// helper methods.
type VersionedNetworkSetResource interface {
	VersionedResource
	getV1NetworkSet() *model.NetworkSet
	isNamespaced() bool
}

// CacheEntryCalicoNetworkSet is a cache entry in the network set cache. Each entry implements the CacheEntry
// interface.
type CacheEntryCalicoNetworkSet struct {
	// The versioned network set resource.
	VersionedNetworkSetResource

	// Boolean values associated with this NetworkSet. Valid flags defined by CacheEntryFlagsNetworkSet.
	Flags CacheEntryFlags

	// The set of policy (allow) rule selectors that match this network set.
	PolicyRuleSelectors resources.Set

	// --- Internal data ---
	cacheEntryCommon
	clog *log.Entry
}

// getVersionedResource implements the CacheEntry interface.
func (c *CacheEntryCalicoNetworkSet) getVersionedResource() VersionedResource {
	return c.VersionedNetworkSetResource
}

// setVersionedResource implements the CacheEntry interface.
func (c *CacheEntryCalicoNetworkSet) setVersionedResource(r VersionedResource) {
	c.VersionedNetworkSetResource = r.(VersionedNetworkSetResource)
}

// versionedCalicoGlobalNetworkSet implements the VersionedNetworkSetResource for a Calico GlobalNetworkSet.
type versionedCalicoGlobalNetworkSet struct {
	*apiv3.GlobalNetworkSet
	v1 *model.NetworkSet
}

// getV3 implements the VersionedNetworkSetResource interface.
func (v *versionedCalicoGlobalNetworkSet) getV3() resources.Resource {
	return v.GlobalNetworkSet
}

// getV1 implements the VersionedNetworkSetResource interface.
func (v *versionedCalicoGlobalNetworkSet) getV1() interface{} {
	return v.v1
}

// getV1NetworkSet implements the VersionedNetworkSetResource interface.
func (v *versionedCalicoGlobalNetworkSet) getV1NetworkSet() *model.NetworkSet {
	return v.v1
}

// isNamespaced implements the VersionedNetworkSetResource interface.
func (v *versionedCalicoGlobalNetworkSet) isNamespaced() bool {
	return false
}

// newCalicoGlobalNetworkSetEngine creates a new engine used for the NetworkSet cache.
func newCalicoGlobalNetworkSetEngine() resourceCacheEngine {
	return &calicoNetworkSetEngine{}
}

// calicoNetworkSetEngine implements the resourceCacheEngine interface for the network set cache.
type calicoNetworkSetEngine struct {
	engineCache
}

// register implements the resourceCacheEngine interface.
func (c *calicoNetworkSetEngine) register(cache engineCache) {
	c.engineCache = cache

	// Register with the allow-rule label seletor so that we can track which allow rules are using this NetworkSet.
	c.NetworkSetLabelSelector().RegisterCallbacks(c.kinds(), c.selectorMatchStarted, c.selectorMatchStopped)
}

// kinds implements the resourceCacheEngine interface.
func (c *calicoNetworkSetEngine) kinds() []metav1.TypeMeta {
	return KindsNetworkSet
}

// newCacheEntry implements the resourceCacheEngine interface.
func (c *calicoNetworkSetEngine) newCacheEntry() CacheEntry {
	return &CacheEntryCalicoNetworkSet{
		PolicyRuleSelectors: resources.NewSet(),
	}
}

// resourceAdded implements the resourceCacheEngine interface.
func (c *calicoNetworkSetEngine) resourceAdded(id apiv3.ResourceID, entry CacheEntry) {
	entry.(*CacheEntryCalicoNetworkSet).clog = log.WithField("id", id)
	c.resourceUpdated(id, entry, nil)
}

// resourceUpdated implements the resourceCacheEngine interface.
func (c *calicoNetworkSetEngine) resourceUpdated(id apiv3.ResourceID, entry CacheEntry, prev VersionedResource) {
	// Use the V1 labels to register with the label selection handler.
	x := entry.(*CacheEntryCalicoNetworkSet)

	// Update the labels for this network set. Always update the labels first so that each cache can get a view of the
	// links before we start sending updates.
	c.NetworkSetLabelSelector().UpdateLabels(id, x.getV1NetworkSet().Labels, nil)
}

// resourceDeleted implements the resourceCacheEngine interface.
func (c *calicoNetworkSetEngine) resourceDeleted(id apiv3.ResourceID, entry CacheEntry) {
	c.NetworkSetLabelSelector().DeleteLabels(id)
}

// recalculate implements the resourceCacheEngine interface.
func (c *calicoNetworkSetEngine) recalculate(id apiv3.ResourceID, entry CacheEntry) syncer.UpdateType {
	x := entry.(*CacheEntryCalicoNetworkSet)

	// Determine whether this network set contains any internet addresses.
	changed := c.scanNets(x)
	x.clog.Debugf("Recalculated, returning update %d, flags now: %d", changed, x.Flags)
	return changed
}

// convertToVersioned implements the resourceCacheEngine interface.
func (c *calicoNetworkSetEngine) convertToVersioned(res resources.Resource) (VersionedResource, error) {
	in := res.(*apiv3.GlobalNetworkSet)

	v1, err := updateprocessors.ConvertGlobalNetworkSetV3ToV1(&model.KVPair{
		Key: model.ResourceKey{
			Kind: apiv3.KindGlobalNetworkSet,
			Name: in.Name,
		},
		Value: in,
	})
	if err != nil {
		return nil, err
	}

	return &versionedCalicoGlobalNetworkSet{
		GlobalNetworkSet: in,
		v1:               v1.Value.(*model.NetworkSet),
	}, nil
}

// scanNets checks the nets in the resource for certain properties (currently just if it contains any non-private
// CIDRs.
func (c *calicoNetworkSetEngine) scanNets(x *CacheEntryCalicoNetworkSet) syncer.UpdateType {
	old := x.Flags
	// Toggle the InternetAddressExposed flag
	x.Flags &^= CacheEntryInternetExposed
	if internet.NetsContainInternetAddr(x.getV1NetworkSet().Nets) {
		x.Flags |= CacheEntryInternetExposed
	}

	// Determine flags that have changed, and convert to an update type. See notes in flags.go.
	changed := syncer.UpdateType(old ^ x.Flags)

	// Return which flags have changed and return as an update type. See notes in flags.go.
	return changed
}

// selectorMatchStarted is called synchronously from the rule selector or network set resource update methods when a
// selector<->netset match has started. We update our set of matched selectors.
func (c *calicoNetworkSetEngine) selectorMatchStarted(selId, netsetId apiv3.ResourceID) {
	x, ok := c.GetFromOurCache(netsetId).(*CacheEntryCalicoNetworkSet)
	if !ok {
		// This is called synchronously from the resource update methods, so we don't expect the entries to have been
		// removed from the cache at this point.
		log.Errorf("Match started on NetworkSet, but NetworkSet is not in cache: %s matches %s", selId, netsetId)
		return
	}
	// Update the selector set in our network set data. No need to queue an async recalculation since this won't affect
	// our settings *and* we don't notify the cache listeners about this event type.
	x.clog.Debugf("Adding %s to policyRuleSelectors for %s", selId, netsetId)
	x.PolicyRuleSelectors.Add(selId)
}

// selectorMatchStopped is called synchronously from the rule selector or network set resource update methods when a
// selector<->netset match has stopped. We update our set of matched selectors.
func (c *calicoNetworkSetEngine) selectorMatchStopped(selId, netsetId apiv3.ResourceID) {
	x, ok := c.GetFromOurCache(netsetId).(*CacheEntryCalicoNetworkSet)
	if !ok {
		// This is called synchronously from the resource update methods, so we don't expect the entries to have been
		// removed from the cache at this point.
		log.Errorf("Match started on NetworkSet, but NetworkSet is not in cache: %s matches %s", selId, netsetId)
		return
	}
	// Update the selector set in our network set data. No need to queue an async recalculation since this won't affect
	// our settings *and* we don't notify the cache listeners about this event type.
	x.clog.Debugf("Removing %s from policyRuleSelectors for %s", selId, netsetId)
	x.PolicyRuleSelectors.Discard(selId)
}
