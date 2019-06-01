// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	"strconv"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/errors"

	"github.com/tigera/compliance/pkg/dispatcher"
	"github.com/tigera/compliance/pkg/keyselector"
	"github.com/tigera/compliance/pkg/labelselector"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

// engineCache is the interface provided to the engine in registration processing to call back into the cache. This
// simply hides the internals of the cache from the resource specific engine implementations.
type engineCache interface {
	GetFromOurCache(res apiv3.ResourceID) CacheEntry
	GetFromXrefCache(res apiv3.ResourceID) CacheEntry

	EndpointLabelSelector() labelselector.Interface
	NetworkSetLabelSelector() labelselector.Interface
	NetworkPolicyRuleSelectorManager() NetworkPolicyRuleSelectorManager
	IPOrEndpointManager() keyselector.Interface

	// Register for updates for other resource types. This registers with the xref cache dispatcher, so the updates
	// will be CacheEntry types and the available updateTypes are defined by the events in flags.go.
	RegisterOnUpdateHandler(kind metav1.TypeMeta, updateTypes syncer.UpdateType, callback dispatcher.DispatcherOnUpdate)

	// Queue the specified resource for async recalculation and update. If the CacheEntry is specified, it should match
	// the ResourceID. If no CacheEntry is specified, it will be looked up.
	QueueUpdate(apiv3.ResourceID, CacheEntry, syncer.UpdateType)
}

// resourceCacheEngine is the interface a specific engine must implement for a resourceCache.
type resourceCacheEngine interface {
	// kinds returns the kinds of resources managed by a particular cache engine. Only a single engine should register
	// for each resource type.
	kinds() []metav1.TypeMeta

	// register initializes the engine with details of the owning cache, this allows the engine to call into the cache
	// to get resources, and to register with the label-selectors for match stopped/started events.
	register(c engineCache)

	// newCacheEntry creates and initializes a new CacheEntry specific to the cache.
	newCacheEntry() CacheEntry

	// convertToVersioned converts a resource provided on the syncer interface to a VersionedResource suitable for
	// use within the cache. A VersionedResource contains Calico V3 and Calico V1 representations of the original
	// resource. Not all resources necessarily contain Calico components, but the cache should still implement this
	// interface for consistency.
	convertToVersioned(res resources.Resource) (VersionedResource, error)

	// resourceAdded is called *after* a CacheEntry has been added into the cache. The CacheEntry will contain the
	// VersionedResource created from the syncer event.
	resourceAdded(id apiv3.ResourceID, entry CacheEntry)

	// resourceUpdated is called *after* a CacheEntry has been updated in the cache. The CacheEntry will contain the
	// VersionedResource created from the syncer event. The method should return the set of updates for any changes that
	// occurred as an immediate result of this method invocation. Generally, this method should only update
	// configuration that is directly determined from the VersionedResource and not from related resources. All cross
	// reference processing should be handled synchronously from this call (e.g. by updating the label or key handlers).
	resourceUpdated(id apiv3.ResourceID, entry CacheEntry, prev VersionedResource)

	// resourceDeleted is called from a syncer delete event *before* the CacheEntry has been deleted. The entry
	// will be deleted immediately after returning from this method invocation.
	resourceDeleted(id apiv3.ResourceID, entry CacheEntry)

	// recalculate is called to calculate or recalculate the augmented resource data for a CacheEntry. This is called
	// asychronously after a related entry was updated that indicated an update was required. This method may itself
	// queue further recalculations. Augmented data that is determine solely from it's own VersionedResource should
	// be handled in resourceAdded/Updated.
	recalculate(id apiv3.ResourceID, entry CacheEntry) syncer.UpdateType
}

// newResourceCache creates a new resourceCache backed by a specific implementation of a resourceCacheEngine.
func newResourceCache(engine resourceCacheEngine) *resourceCache {
	return &resourceCache{
		resources: make(map[apiv3.ResourceID]CacheEntry, 0),
		engine:    engine,
	}
}

// resourceCache is a cache for a related set of resource types. It is a sub-cache of the main cache.
type resourceCache struct {
	xc        *xrefCache
	resources map[apiv3.ResourceID]CacheEntry
	engine    resourceCacheEngine
}

// Callback for the syncer updates. Fan out to the onNewOrUpdated and onDeleted methods.
func (c *resourceCache) onUpdate(update syncer.Update) {
	switch update.Type {
	case syncer.UpdateTypeDeleted:
		c.onDeleted(update.ResourceID)
	case syncer.UpdateTypeSet:
		c.onNewOrUpdated(update.ResourceID, update.Resource)
	default:
		log.Errorf("Unexpected update type from syncer: %d (%s)", update.Type, strconv.FormatInt(int64(update.Type), 2))
		return
	}
}

// onNewOrUpdated checks cache for existing entry and treats as new or updated based on that rather than on what
// the syncer indicated - just to more gracefully handle discrepencies.
func (c *resourceCache) onNewOrUpdated(id apiv3.ResourceID, res resources.Resource) {
	// Convert the resource to a versioned resource (this contains the various versioned representations of the
	// resource required by our cache processing).
	v, err := c.engine.convertToVersioned(res)
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
			log.WithField("id", id).Debug("Cache has indicated an explicit delete of resource")
		} else {
			log.WithError(err).WithField("id", id).Error("Unable to convert resource, treating as delete")
		}
		c.onDeleted(id)
		return
	}
	if v == nil {
		// The conversion may have deliberately filtered out this resource, in which case treat as a delete. This
		// is not an error condition and so should log appropriately.
		log.WithField("id", id).Info("Resource filtered out, treating as delete")
		c.onDeleted(id)
		return
	}

	if entry, ok := c.resources[id]; ok {
		log.Debugf("Update existing resource in cache: %s", id)
		// Updating the resource, start by getting the current versioned resource data and then update with the new
		// versioned resource.
		prev := entry.getVersionedResource()
		entry.setVersionedResource(v)

		// Set the update type flag for this update, this prevents the update being sent by any callback processing
		// until the resourceUpdated() call returns.
		c.xc.queueUpdate(id, entry, EventResourceModified)

		// Call through to the engine to perform any additional processing for this resource update.
		c.engine.resourceUpdated(id, entry, prev)
	} else {
		log.Debugf("Add new resource to cache: %s", id)
		// Create a new cache entry and set the versioned resource.
		entry = c.engine.newCacheEntry()
		c.resources[id] = entry
		entry.setResourceID(id)
		entry.setVersionedResource(v)

		// Requeue this resource for recalculation.
		c.xc.queueUpdate(id, entry, EventResourceAdded)

		// Call through to the engine to perform any additional processing for this resource creation, in particular
		// setting up any xrefs. Calculation of data is performed asynchronously.
		c.engine.resourceAdded(id, entry)
	}
}

func (c *resourceCache) onDeleted(id apiv3.ResourceID) {
	log.Debugf("Deleting resource from cache: %s", id)
	if entry, ok := c.resources[id]; ok {
		// Call through to the engine to perform any additional processing for this resource creation.
		c.engine.resourceDeleted(id, entry)

		// Delete the entry from the cache.
		delete(c.resources, id)
	}
}

func (c *resourceCache) get(id apiv3.ResourceID) CacheEntry {
	return c.resources[id]
}

// Called by the main cache to register itself with this sub-cache.
func (c *resourceCache) register(xc *xrefCache) {
	// Store the main cache.
	c.xc = xc

	// Create the cache interface for the engine.
	ci := &resourceEngineCache{
		ours: c,
		xc:   xc,
	}

	// Register the cache with the syncer dispatcher to get updates for actual resource updates for the resources
	// managed by this cache.
	for _, kind := range c.engine.kinds() {
		xc.syncerDispatcher.RegisterOnUpdateHandler(
			kind,
			syncer.UpdateTypeSet|syncer.UpdateTypeDeleted,
			c.onUpdate,
		)
	}

	// Register with the engine class so that it can register for xref updates (i.e. updates from indirectly generated
	// data).
	c.engine.register(ci)
}

func (c *resourceCache) kinds() []metav1.TypeMeta {
	return c.engine.kinds()
}

// resourceEngineCache implements the engineCache. This limits access to cache innards from the engine implementation,
// and more importantly provides a level of indirection between the engine and the cache dispatcher.
type resourceEngineCache struct {
	ours *resourceCache
	xc   *xrefCache
}

func (c *resourceEngineCache) GetFromOurCache(res apiv3.ResourceID) CacheEntry {
	log.WithField("id", res).Debug("Get resource from our own cache")
	return c.ours.resources[res]
}

func (c *resourceEngineCache) GetFromXrefCache(res apiv3.ResourceID) CacheEntry {
	log.WithField("id", res).Debug("Get resource from x-ref cache")
	return c.xc.Get(res)
}

func (c *resourceEngineCache) EndpointLabelSelector() labelselector.Interface {
	return c.xc.endpointLabelSelector
}

func (c *resourceEngineCache) NetworkSetLabelSelector() labelselector.Interface {
	return c.xc.networkSetLabelSelector
}

func (c *resourceEngineCache) NetworkPolicyRuleSelectorManager() NetworkPolicyRuleSelectorManager {
	return c.xc.networkPolicyRuleSelectorManager
}

func (c *resourceEngineCache) IPOrEndpointManager() keyselector.Interface {
	return c.xc.ipOrEndpointManager
}

func (c *resourceEngineCache) QueueUpdate(id apiv3.ResourceID, entry CacheEntry, update syncer.UpdateType) {
	c.xc.queueUpdate(id, entry, update)
}

func (c *resourceEngineCache) RegisterOnUpdateHandler(kind metav1.TypeMeta, updateTypes syncer.UpdateType, callback dispatcher.DispatcherOnUpdate) {
	c.xc.cacheDispatcher.RegisterOnUpdateHandler(kind, updateTypes, callback)
}
