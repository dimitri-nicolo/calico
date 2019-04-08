// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	"container/heap"
	"fmt"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime/schema"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/dispatcher"
	"github.com/tigera/compliance/pkg/keyselector"
	"github.com/tigera/compliance/pkg/labelselector"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

// NewXrefCache creates a new cross-referenced XrefCache.
func NewXrefCache() XrefCache {
	// Create a dispatcher for use internally within the cross reference cache. The resources passed around in this
	// dispatcher will be augmented from the basic resource provided by the syncer.
	cacheDispatcher := dispatcher.NewDispatcher()

	// Create a dispatcher for the syncer. This simply fans out the original update to the appropriate cache responsible
	// for storing that resource kind.
	syncerDispatcher := dispatcher.NewDispatcher()

	// Create a dispatcher sending events to the consumer of the cross reference cache.
	consumerDispatcher := dispatcher.NewDispatcher()

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
		consumerDispatcher:               consumerDispatcher,
		endpointLabelSelector:            endpointLabelSelection,
		networkSetLabelSelector:          netsetLabelSelection,
		networkPolicyRuleSelectorManager: networkPolicyRuleSelectorManager,
		ipManager:                        keyselector.New(),
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
	consumerDispatcher               dispatcher.Dispatcher
	endpointLabelSelector            labelselector.Interface
	networkSetLabelSelector          labelselector.Interface
	networkPolicyRuleSelectorManager NetworkPolicyRuleSelectorManager
	ipManager                        keyselector.Interface
	caches                           map[schema.GroupVersionKind]*resourceCache
	priorities                       map[schema.GroupVersionKind]int8
	modified                         resources.PriorityQueue
	inSync                           bool
}

// OnStatusUpdate implements the XrefCache interface.
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

	// Finally, notify the cache dispatcher and the consumer dispatcher of the status update.
	c.cacheDispatcher.OnStatusUpdate(status)
	c.consumerDispatcher.OnStatusUpdate(status)
}

// OnUpdate implements the XrefCache interface.
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

		// Reset the update types now, just incase by some oddity the recalculation attempts to re-add itself to the
		// queue (which is fine, but odd).
		updates := entry.getUpdateTypes()
		entry.resetUpdateTypes()

		if updates&EventsNotRequiringRecalculation != updates {
			// The set of updates that have been queued do require some recalculation, therefore recalculate the entry,
			// combine the response with the existing set of update types.
			updates |= cache.engine.recalculate(id, entry)
		}

		// If we are in-sync then send a notification via the cache dispatcher for this entry. Always include the
		// in-scope flag if the entry is in-scope.
		update := syncer.Update{
			Type:       updates | entry.getInScopeFlag(),
			ResourceID: id,
			Resource:   entry,
		}
		c.cacheDispatcher.OnUpdate(update)
		if c.inSync {
			// The consumers only gets updates once we are in-sync otherwise the augmented data will not be correct
			// at start of day. Once in-sync, the onStatusUpdate processing will send a complete dump of the
			// cache to the consumer and only then will updates be sent as we calculate them.
			c.consumerDispatcher.OnUpdate(update)
		}
	}
}

// Get implements the XrefCache interface.
func (c *xrefCache) Get(id resources.ResourceID) CacheEntry {
	return c.caches[id.GroupVersionKind].get(id)
}

// RegisterOnStatusUpdateHandler implements the XrefCache interface.
func (c *xrefCache) RegisterOnStatusUpdateHandler(callback dispatcher.DispatcherOnStatusUpdate) {
	c.consumerDispatcher.RegisterOnStatusUpdateHandler(callback)
}

// RegisterOnUpdateHandler implements the XrefCache interface.
func (c *xrefCache) RegisterOnUpdateHandler(kind schema.GroupVersionKind, updateTypes syncer.UpdateType, callback dispatcher.DispatcherOnUpdate) {
	c.consumerDispatcher.RegisterOnUpdateHandler(kind, updateTypes, callback)
}

// GetCachedResourceIDs implements the XrefCache interface.
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

// RegisterInScopeEndpoints implements the XrefCache interface.
func (c *xrefCache) RegisterInScopeEndpoints(selection apiv3.EndpointsSelection) error {
	resId, sel, err := calculateInScopeEndpointsSelector(selection)
	if err != nil {
		return err
	}
	c.endpointLabelSelector.UpdateSelector(resId, sel)
	return nil
}

// queueUpdate adds this update to the priority queue. The priority is determined by the resource kind.
func (c *xrefCache) queueUpdate(id resources.ResourceID, entry CacheEntry, update syncer.UpdateType) {
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
	} else {
		log.WithField("id", id).Debug("Queue recalculation requested, but alreadt in progress")
	}
}
