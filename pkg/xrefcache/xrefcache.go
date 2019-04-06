// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"container/heap"

	"github.com/tigera/compliance/pkg/dispatcher"
	"github.com/tigera/compliance/pkg/labelselector"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

// VersionedResource is an extension to the Resource interface to add some additional versioning
// (converting the original resource into the v3 Calico model and then the v1 Calico model).
type VersionedResource interface {
	resources.Resource
	getV3() resources.Resource
	getV1() interface{}
}

// All internal caches store types that implement the CacheEntry interface.
type CacheEntry interface {
	VersionedResource
	getVersionedResource() VersionedResource
	setVersionedResource(r VersionedResource)
	getUpdateTypes() syncer.UpdateType
	setUpdateTypes(syncer.UpdateType)
	resetUpdateTypes()
}

// cacheEntryCommon is embedded in each concrete CacheEntry type to provide the updateInProgress identifiers used
// by the cache processing to handle sending of updates only at the end of a syncer update.
type cacheEntryCommon struct {
	updateTypes syncer.UpdateType
}

// getUpdateTypes returns the accumulated update types for a resource that is being updated from a syncer update.
func (c *cacheEntryCommon) getUpdateTypes() syncer.UpdateType {
	return c.updateTypes
}

// setUpdateTypes adds the supplied update types to the accumlated set of updates for a resource that is being
// updated from a syncer update.
func (c *cacheEntryCommon) setUpdateTypes(u syncer.UpdateType) {
	c.updateTypes |= u
}

// resetUpdateTypes is called at the end of the syncer update processing to reset the accumlated set of updates
// for the resource being updated from a syncer update.
func (c *cacheEntryCommon) resetUpdateTypes() {
	c.updateTypes = 0
}

// XrefCache interface.
//
// This interface implements the SyncerCallbacks which is used to populate the cache from the raw K8s and Calico resource
// events.
type XrefCache interface {
	syncer.SyncerCallbacks
	Get(res resources.ResourceID) CacheEntry
	RegisterOnStatusUpdateHandler(callback dispatcher.DispatcherOnStatusUpdate)
	RegisterOnUpdateHandler(kind schema.GroupVersionKind, updateTypes syncer.UpdateType, callback dispatcher.DispatcherOnUpdate)
	GetCachedResourceIDs(kind schema.GroupVersionKind) []resources.ResourceID
}

// NewXrefCache creates a new cross-referenced XrefCache.
func NewXrefCache() XrefCache {
	// Create a dispatcher for use internally within the cross reference cache. The resources passed around in this
	// dispatcher will be augmented from the basic resource provided by the syncer.
	cacheDispatcher := dispatcher.NewDispatcher()

	// Create a dispatcher for the syncer. This simply fans out the original update to the appropriate cache responsible
	// for storing that resource kind.
	syncerDispatcher := dispatcher.NewDispatcher()

	// Register the cache dispatcher as a consumer of status events with the syncer dispatcher (basically this passes
	// status events straight through from the syncer to the xref caches.
	syncerDispatcher.RegisterOnStatusUpdateHandler(cacheDispatcher.OnStatusUpdate)

	// Label handler for endpoint matches.
	endpointLabelSelection := labelselector.NewLabelSelection()

	// Label handler for positive network set matches. We don't currently need to track negated matches.
	netsetLabelSelection := labelselector.NewLabelSelection()

	// Policy to rule selection manager
	networkPolicyRuleSelectorManager := NewNetworkPolicyRuleSelectorManager(syncerDispatcher.OnUpdate)

	// Create the various engines that underpin the separate resource caches. This list is ordered by recalculation
	// queue priority (highest index, highest priority).
	allEngines := []resourceCacheEngine{
		newEndpointsEngine(),
		newK8sServiceEndpointsEngine(),
		newK8sNamespacesEngine(),
		newK8sServiceAccountsEngine(),
		newNetworkPoliciesEngine(),
		newNetworkPolicyRuleSelectorsEngine(),
		newCalicoTiersEngine(),
		newCalicoGlobalNetworkSetEngine(),
	}

	c := &xrefCache{
		cacheDispatcher:                  cacheDispatcher,
		syncerDispatcher:                 syncerDispatcher,
		endpointLabelSelector:            endpointLabelSelection,
		networkSetLabelSelector:          netsetLabelSelection,
		networkPolicyRuleSelectorManager: networkPolicyRuleSelectorManager,
		caches:                           map[schema.GroupVersionKind]*resourceCache{},
		priorities:                       map[schema.GroupVersionKind]int8{},
	}
	// Initialise the priority queue used for handling asynchronous resource recalculations.
	heap.Init(&c.modified)

	// Create caches and determine priorities of resource types.
	for i, engine := range allEngines {
		cache := newResourceCache(engine)
		cache.register(c)
		for _, kind := range cache.kinds() {
			if _, ok := c.caches[kind]; ok {
				panic(fmt.Errorf("Resource kind registered with multiple caches: %s", kind))
			}
			c.caches[kind] = cache
			c.priorities[kind] = int8(i)
		}
	}

	return c
}

// xc implements the GlobalCache interface.
type xrefCache struct {
	syncerDispatcher                 dispatcher.Dispatcher
	cacheDispatcher                  dispatcher.Dispatcher
	endpointLabelSelector            labelselector.Interface
	networkSetLabelSelector          labelselector.Interface
	networkPolicyRuleSelectorManager NetworkPolicyRuleSelectorManager
	caches                           map[schema.GroupVersionKind]*resourceCache
	priorities                       map[schema.GroupVersionKind]int8
	modified                         resources.PriorityQueue
	inSync                           bool
}

func (c *xrefCache) OnStatusUpdate(status syncer.StatusUpdate) {
	log.Infof("Processing status update: %#o", status.Type)

	// Notify the syncer dispatcher first.
	c.syncerDispatcher.OnStatusUpdate(status)

	// If we are now in-sync then dump the entire contents of the cache through the cache dispatcher.
	if status.Type == syncer.StatusTypeInSync {
		for kind, cache := range c.caches {
			log.Debugf("Dumping cache: %s", kind)
			cache.dumpResourcesAsUpdate()
		}
		c.inSync = true
	}

	// Finally, notify the cache dispatcher of the status update.
	c.cacheDispatcher.OnStatusUpdate(status)
}

func (c *xrefCache) OnUpdate(update syncer.Update) {
	c.syncerDispatcher.OnUpdate(update)
	for c.modified.Len() > 0 {
		id := heap.Pop(&c.modified).(*resources.QueueItem).ResourceID

		cache := c.caches[id.GroupVersionKind]
		entry := cache.get(id)
		if entry == nil {
			log.Errorf("Resource queued for recalculation, but is no longer in cache: %s", id)
			continue
		}

		// Recalculate the entry, combine the response with the existing set of update types.
		updates := cache.engine.recalculate(id, entry) | entry.getUpdateTypes()

		// If we are in-sync then send a notification via the cache dispatcher for this entry.
		if c.inSync {
			c.cacheDispatcher.OnUpdate(syncer.Update{
				Type:       updates,
				ResourceID: id,
				Resource:   entry,
			})
		}

		// Reset the update types.
		entry.resetUpdateTypes()
	}
}

func (c *xrefCache) Get(id resources.ResourceID) CacheEntry {
	return c.caches[id.GroupVersionKind].get(id)
}

func (c *xrefCache) RegisterOnStatusUpdateHandler(callback dispatcher.DispatcherOnStatusUpdate) {
	c.cacheDispatcher.RegisterOnStatusUpdateHandler(callback)
}

func (c *xrefCache) RegisterOnUpdateHandler(kind schema.GroupVersionKind, updateTypes syncer.UpdateType, callback dispatcher.DispatcherOnUpdate) {
	c.cacheDispatcher.RegisterOnUpdateHandler(kind, updateTypes, callback)
}

func (c *xrefCache) GetCachedResourceIDs(kind schema.GroupVersionKind) []resources.ResourceID {
	var ids []resources.ResourceID
	cache := c.caches[kind]
	for k := range cache.resources {
		if k.GroupVersionKind == kind {
			ids = append(ids, k)
		}
	}
	return ids
}

func (c *xrefCache) queueRecalculation(id resources.ResourceID, entry CacheEntry, update syncer.UpdateType) {
	if update == 0 {
		log.Errorf("Update type should always be specified for resource recalculation: %s", id)
		return
	}

	if entry == nil {
		entry = c.Get(id)
		if entry == nil {
			log.Errorf("Queue recalculation request for resource that is no longer in cache: %s", id)
			return
		}
	}

	queue := entry.getUpdateTypes() == 0
	entry.setUpdateTypes(update)
	if queue {
		// There are no other recalculations pending for this resource
		log.WithField("id", id).Debug("Queue recalculation of resource")
		item := &resources.QueueItem{
			ResourceID: id,
			Priority:   c.priorities[id.GroupVersionKind],
		}
		heap.Push(&c.modified, item)
	}
}
