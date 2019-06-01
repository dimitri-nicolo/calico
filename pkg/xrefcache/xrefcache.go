// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	"container/heap"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/dispatcher"
	"github.com/tigera/compliance/pkg/keyselector"
	"github.com/tigera/compliance/pkg/labelselector"
	"github.com/tigera/compliance/pkg/syncer"
)

// NewXrefCache creates a new cross-referenced XrefCache.
func NewXrefCache(healthy func()) XrefCache {
	// Create a dispatcher for use internally within the cross reference cache. The resources passed around in this
	// dispatcher will be augmented from the basic resource provided by the syncer.
	cacheDispatcher := dispatcher.NewDispatcher("cache")

	// Create a dispatcher for the syncer. This simply fans out the original update to the appropriate cache responsible
	// for storing that resource kind.
	syncerDispatcher := dispatcher.NewDispatcher("syncer")

	// Create a dispatcher sending events to the consumer of the cross reference cache.
	consumerDispatcher := dispatcher.NewDispatcher("consumer")

	// Register the cache dispatcher as a consumer of status events with the syncer dispatcher (basically this passes
	// status events straight through from the syncer to the xref caches.
	syncerDispatcher.RegisterOnStatusUpdateHandler(cacheDispatcher.OnStatusUpdate)

	// Label handler for endpoint matches.
	endpointLabelSelection := labelselector.NewLabelSelection()

	// Label handler for positive network set matches. We don't currently need to track negated matches.
	netsetLabelSelection := labelselector.NewLabelSelection()

	// Policy to rule selection manager
	networkPolicyRuleSelectorManager := NewNetworkPolicyRuleSelectorManager(syncerDispatcher.OnUpdate)

	// Initialize the config.
	config := &Config{}
	envconfig.MustProcess("TIGERA_COMPLIANCE", config)

	// Create the various engines that underpin the separate resource caches. This list is ordered by recalculation
	// queue priority (highest index, highest priority).
	allEngines := []resourceCacheEngine{
		newEndpointsEngine(config),
		newK8sServiceEndpointsEngine(),
		newK8sNamespacesEngine(),
		newK8sServiceAccountsEngine(),
		newNetworkPoliciesEngine(),
		newNetworkPolicyRuleSelectorsEngine(),
		newCalicoGlobalNetworkSetEngine(),
	}

	c := &xrefCache{
		healthy:                          healthy,
		cacheDispatcher:                  cacheDispatcher,
		syncerDispatcher:                 syncerDispatcher,
		consumerDispatcher:               consumerDispatcher,
		endpointLabelSelector:            endpointLabelSelection,
		networkSetLabelSelector:          netsetLabelSelection,
		networkPolicyRuleSelectorManager: networkPolicyRuleSelectorManager,
		ipOrEndpointManager:              keyselector.New(),
		caches:                           map[metav1.TypeMeta]*resourceCache{},
		priorities:                       map[metav1.TypeMeta]int8{},
	}
	// Initialise the priority queue used for handling asynchronous resource recalculations.
	heap.Init(&c.modified)

	// Create caches and determine priorities of resource types.
	for i, engine := range allEngines {
		cache := newResourceCache(engine)
		cache.register(c)
		for _, kind := range cache.kinds() {
			if _, ok := c.caches[kind]; ok {
				log.Panicf("Resource kind %s registered with multiple caches.", kind)
			}
			c.caches[kind] = cache
			c.priorities[kind] = int8(i)
		}
	}

	return c
}

// xc implements the GlobalCache interface.
type xrefCache struct {
	healthy                          func()
	livenessReporter                 string
	syncerDispatcher                 dispatcher.Dispatcher
	cacheDispatcher                  dispatcher.Dispatcher
	consumerDispatcher               dispatcher.Dispatcher
	endpointLabelSelector            labelselector.Interface
	networkSetLabelSelector          labelselector.Interface
	networkPolicyRuleSelectorManager NetworkPolicyRuleSelectorManager
	ipOrEndpointManager              keyselector.Interface
	caches                           map[metav1.TypeMeta]*resourceCache
	priorities                       map[metav1.TypeMeta]int8
	modified                         PriorityQueue
	inSync                           bool
}

// OnStatusUpdate implements the syncer interface.
func (x *xrefCache) OnStatusUpdate(status syncer.StatusUpdate) {
	log.Infof("Processing status update: %#o", status.Type)

	// Indicate we are healthy.
	x.healthy()

	// Notify the syncer dispatcher first.
	x.syncerDispatcher.OnStatusUpdate(status)

	// Notify the cache dispatcher and the consumer dispatcher of the status update.
	x.cacheDispatcher.OnStatusUpdate(status)
	x.consumerDispatcher.OnStatusUpdate(status)

	// If we are now in-sync then process anything awaiting calculation on the queue and send updates.
	if status.Type == syncer.StatusTypeInSync {
		x.processQueue()
		x.inSync = true
	}

	// Indicate we are healthy.
	x.healthy()
}

// OnUpdate implements the syncer interface.
func (x *xrefCache) OnUpdates(updates []syncer.Update) {
	log.Debugf("Processing OnUpdates with %d updates in transaction", len(updates))

	// Indicate we are healthy.
	x.healthy()

	// Short-circuit no updates.
	if len(updates) == 0 {
		return
	}

	// To handle the rather unlikely situation where we have a delete followed by a recreate in a single transaction
	// (which would only happen with watcher updates because we gather updates when processing can't keep up) we'll
	// split up multiple updates into groups of sets and deletes in the order supplied.
	firstIdx := 0
	updateType := updates[0].Type
	for lastIdx := 1; lastIdx < len(updates); lastIdx++ {
		if updates[lastIdx].Type != updateType {
			x.processUpdatesOfType(updateType, updates[firstIdx:lastIdx])
			firstIdx = lastIdx
			updateType = updates[firstIdx].Type
		}
	}
	x.processUpdatesOfType(updateType, updates[firstIdx:])

	// Indicate we are healthy.
	x.healthy()
}

// processUpdates handles a batch of updates of a common type.
func (x *xrefCache) processUpdatesOfType(updateType syncer.UpdateType, updates []syncer.Update) {
	log.Debugf("Processing updates of type: %s", updateType)

	if updateType == syncer.UpdateTypeDeleted {
		for i := range updates {
			id := updates[i].ResourceID
			cache := x.caches[id.TypeMeta]
			entry := cache.get(id)
			if entry == nil {
				// Deletion for a resource that is not in our cache. Unexpected, but just log and skip.
				log.Warningf("Deletion event for resource that is not in cache: %s", id)
				continue
			}

			// Flag the resource as deleted - this will minimize any further processing on this resource.
			entry.setDeleted()

			if x.inSync {
				// If we are in-sync send deletes to the consumer immediately. We do not want to  process the deletes
				// in the cross-ref processing first because it may introduce churn in resources that are going
				// to be deleted by this same syncer update and we want the deletes to appear atomic and with their
				// current settings.
				x.consumerDispatcher.OnUpdate(syncer.Update{
					Type:       EventResourceDeleted | entry.getInScopeFlag(),
					ResourceID: id,
					Resource:   entry,
				})
			}
		}
	}

	// Now send the updates on the syncer dispatcher to process each in our sub resource cross referenced caches. This
	// may queue up other resource updates for consumer notification which we don't need to process until we are
	// in-sync.
	for i := range updates {
		x.syncerDispatcher.OnUpdate(updates[i])
	}

	// If we are in-sync, process the work queue.
	if x.inSync {
		x.processQueue()
	}
}

// processQueue pulls entries off the work queue, performing recalculations if required and sending dispatcher updates
// for each entry.
//
// Notes:
// -  Cross referencing between different resources (e.g. profile -> endpoint) is updated synchronously as part of
//    the syncer dispatcher update above. Thus we can send multiple updates to the caches and perform the calculated
//    state updates afterwards.
// -  There is no circular dependency of *calculated* state, so it is safe and more efficient to use a priority
//    queue for processing the updated resources. Once a resource of higher priority has been calculated it is
//    safe to send updates for that resource without triggering further updates for that resource.
func (x *xrefCache) processQueue() {
	// Process queued updates for recalculation and consumer notification.
	for x.modified.Len() > 0 {
		entry := heap.Pop(&x.modified).(*QueueItem).Entry
		id := entry.getResourceID()

		// A deleted resource may have been queued prior to deletion. Skip.
		if entry.isDeleted() {
			log.Infof("Resource queued for recalculation, but is now deleted from cache: %s", id)
			continue
		}

		// Reset the update types now, just incase by some oddity the recalculation attempts to re-add itself to the
		// queue (which is fine, but odd).
		log.Debugf("Processing resource in queue: %s", id)
		updates := entry.getUpdateTypes()
		entry.resetUpdateTypes()
		cache := x.caches[id.TypeMeta]

		if updates&^EventsNotRequiringRecalculation != 0 {
			// The set of updates that have been queued do require some recalculation, therefore recalculate the entry,
			// combine the response with the existing set of update types.
			updates |= cache.engine.recalculate(id, entry)
		}

		// Notify interested caches of changes to this entry. Always include the in-scope flag if the entry is in-scope.
		update := syncer.Update{
			Type:       updates | entry.getInScopeFlag(),
			ResourceID: id,
			Resource:   entry,
		}
		x.cacheDispatcher.OnUpdate(update)

		// And notify the consumer of the update.
		x.consumerDispatcher.OnUpdate(update)
	}
}

// Get implements the XrefCache interface.
func (x *xrefCache) Get(id apiv3.ResourceID) CacheEntry {
	return x.caches[id.TypeMeta].get(id)
}

// RegisterOnStatusUpdateHandler implements the XrefCache interface.
func (x *xrefCache) RegisterOnStatusUpdateHandler(callback dispatcher.DispatcherOnStatusUpdate) {
	x.consumerDispatcher.RegisterOnStatusUpdateHandler(callback)
}

// RegisterOnUpdateHandler implements the XrefCache interface.
func (x *xrefCache) RegisterOnUpdateHandler(kind metav1.TypeMeta, updateTypes syncer.UpdateType, callback dispatcher.DispatcherOnUpdate) {
	x.consumerDispatcher.RegisterOnUpdateHandler(kind, updateTypes, callback)
}

// GetCachedResourceIDs implements the XrefCache interface.
func (x *xrefCache) GetCachedResourceIDs(kind metav1.TypeMeta) []apiv3.ResourceID {
	var ids []apiv3.ResourceID
	cache := x.caches[kind]
	for k := range cache.resources {
		if k.TypeMeta == kind {
			ids = append(ids, k)
		}
	}
	return ids
}

// RegisterInScopeEndpoints implements the XrefCache interface.
func (x *xrefCache) RegisterInScopeEndpoints(selection *apiv3.EndpointsSelection) error {
	resId, sel, err := calculateInScopeEndpointsSelector(selection)
	if err != nil {
		return err
	}
	x.endpointLabelSelector.UpdateSelector(resId, sel)
	return nil
}

// queueUpdate adds this update to the priority queue. The priority is determined by the resource kind.
func (x *xrefCache) queueUpdate(id apiv3.ResourceID, entry CacheEntry, update syncer.UpdateType) {
	if update == 0 {
		log.Errorf("Update type should always be specified for resource recalculation: %s", id)
		return
	}

	if entry == nil {
		entry = x.Get(id)
		if entry == nil {
			// If the resource was deleted then cross references should have been updated. This is therefore an
			// unexpected condition.
			log.Errorf("Queue recalculation request for resource that is no longer in cache: %s", id)
			return
		}
	}

	if entry.isDeleted() {
		// For multi-update transactions, it's possible to get an update for a deleted resource.
		log.Debugf("Queue recalculation request for resource that has been deleted: %s", id)
		return
	}

	queue := entry.getUpdateTypes() == 0
	entry.addUpdateTypes(update)
	if queue {
		// There are no other recalculations pending for this resource
		log.WithField("id", id).Debug("Queue recalculation of resource")
		item := &QueueItem{
			Entry:    entry,
			Priority: x.priorities[id.TypeMeta],
		}
		heap.Push(&x.modified, item)
	} else {
		log.WithField("id", id).Debug("Queue recalculation requested, but already in progress")
	}
}
