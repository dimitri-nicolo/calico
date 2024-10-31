// Copyright 2022 Tigera Inc. All rights reserved.
package panoramawatcher

import (
	"context"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"

	panclient "github.com/projectcalico/calico/firewall-integration/pkg/controllers/panorama/backend/client"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/jitter"
)

// The resyncCache provides syncer support for a single key type in the backend.
// These results are sent to the main WatcherSyncer on a buffered "results" channel.
// To ensure the order of events is received correctly by the main WatcherSyncer,
// we send all notification types in this channel. Note that because of this the results
// channel is untyped - however the watcherSyncer only expects one of the following
// types:
// -  An error
// -  An api.Update
// -  A api.SyncStatus (only for the very first InSync notification)
type resyncCache struct {
	logger       *logrus.Entry
	client       panclient.PanoramaFirewallPolicyClient
	resources    map[string]cacheEntry
	oldResources map[string]cacheEntry
	results      chan<- interface{}
	hasSynced    bool
	resourceType ResourceType
	ticker       *jitter.Ticker
}

var (
	ListRetryInterval = 1000 * time.Millisecond
)

// cacheEntry is an entry in our cache.
type cacheEntry struct {
	key   model.Key
	value interface{}
}

// Create a new watcherCache.
// The implementation is duplicate of the libcalico-go lib/backend/watchersyncer/watchercache.go,
// however its use publicly available in the controller.
// TODO(dimitrin): Investigate the publishing the libcalico-go watcher cache for use in this
// controller, and removing this implementation.
func newWatcherCache(
	client panclient.PanoramaFirewallPolicyClient, resourceType ResourceType, results chan<- interface{}, ticker *jitter.Ticker,
) *resyncCache {
	return &resyncCache{
		logger:       logrus.WithField("ClientId", resourceType.ClientID),
		client:       client,
		resourceType: resourceType,
		results:      results,
		resources:    make(map[string]cacheEntry),
		ticker:       ticker,
	}
}

// run creates the watcher and loops indefinitely reading from the watcher.
func (wc *resyncCache) run(ctx context.Context) {
	wc.logger.Debug("Start resync processing")

	wc.resync(ctx)
}

// resync loops performing sync processing. Returns if the ctx's Done() channel receives a value.
func (wc *resyncCache) resync(ctx context.Context) {
	wc.logger.Debug("Starting resync processing")

	for {
		wc.logger.Debug("in resync loop")
		select {
		case <-ctx.Done():
			wc.logger.Debug("Context is done. Returning")
			return
		case <-wc.ticker.C:
			// Start the resync on a timed synchronization loop.
			wc.logger.Debug("Ticker channel received value, starting resync")
		}

		wc.logger.Info("Starting main resync")

		// Start the sync by listing the current resources.
		l, err := wc.client.List()
		if err != nil {
			// Failed to perform the list. Pause briefly (so we don't tight loop) and retry.
			wc.logger.WithError(err).Info("Failed to perform list of current data during resync")

			// Need to send back an error here for handling. Only callbacks with connection failure handling should actually kick off anything.
			wc.results <- errorSyncBackendError{
				Err: err,
			}

			select {
			case <-time.After(ListRetryInterval):
				continue
			case <-ctx.Done():
				wc.logger.Debug("Context is done. Returning")
				return
			}
		}

		// Move the current resources over to the oldResources
		wc.oldResources = wc.resources
		wc.resources = make(map[string]cacheEntry)

		// Send updates for each of the resources we listed - this will revalidate entries in the
		// oldResources map.
		for _, kvp := range l.KVPairs {
			wc.handleAddedOrModifiedUpdate(kvp)
		}

		// We've listed the current settings. Complete the sync by sending deletes for the old resources
		// that were not acknowledged by the List. The oldResources will be empty after this call.
		wc.finishResync()
	}
}

// finishResync handles processing to finish synchronization.
// If this watcher has never been synced then notify the main watcherSyncer that we've synced.
// We may also need to send deleted messages for old resources that were not validated in the
// resync (i.e. they must have since been deleted).
func (wc *resyncCache) finishResync() {
	// If we haven't already sent an InSync event then send a synced notification. The watcherSyncer
	// will send a Synced event when it has received synced events from each cache. Once in-sync the
	// cache remains in-sync.
	if !wc.hasSynced {
		wc.logger.Info("Sending synced update")
		wc.results <- api.InSync
		wc.hasSynced = true
	}

	// Now that we have finished the sync, any of the remaining resources that were not accounted for
	// must have been deleted and we need to send deleted events for them.
	numOldResources := len(wc.oldResources)
	if numOldResources > 0 {
		wc.logger.WithField("Num", numOldResources).Debug("Sending resync deletes")
		updates := make([]api.Update, 0, len(wc.oldResources))
		for _, r := range wc.oldResources {
			updates = append(updates, api.Update{
				UpdateType: api.UpdateTypeKVDeleted,
				KVPair: model.KVPair{
					Key: r.key,
				},
			})
		}
		wc.results <- updates
	}
	wc.oldResources = nil
}

// handleAddedOrModifiedUpdate handles a single Added or Modified update request.
// Whether we send an Added or Modified depends on whether we have already sent
// an added notification for this resource.
func (wc *resyncCache) handleAddedOrModifiedUpdate(kvp *model.KVPair) {
	thisKey := kvp.Key
	thisKeyString := thisKey.String()
	wc.markAsValid(thisKeyString)

	// If the resource is already in our map, then this is a modified event. Check the revision to see
	// if we actually need to send an update.
	if resource, ok := wc.resources[thisKeyString]; ok {
		if reflect.DeepEqual(kvp.Value, resource.value) {
			// No update to revision, so no event to send.
			wc.logger.WithField("Key", thisKeyString).
				Debug("Swallowing event update from datastore because entry is same as cached entry")
			return
		}
		wc.logger.WithField("Key", thisKeyString).Debug("Datastore entry modified, sending syncer update")
		wc.results <- []api.Update{{
			UpdateType: api.UpdateTypeKVUpdated,
			KVPair:     *kvp,
		}}
		wc.resources[thisKeyString] = cacheEntry{
			key:   thisKey,
			value: kvp.Value,
		}
		return
	}

	// The resource has not been seen before, so send a new event, and store the current revision.
	wc.logger.WithField("Key", thisKeyString).Debug("Cache entry added, sending syncer update")
	wc.results <- []api.Update{{
		UpdateType: api.UpdateTypeKVNew,
		KVPair:     *kvp,
	}}
	wc.resources[thisKeyString] = cacheEntry{
		key:   thisKey,
		value: kvp.Value,
	}
}

// markAsValid marks a resource that we have just seen as valid, by moving it from the set of
// "oldResources" that were stored during the resync back into the main "resources" set.  Any
// entries remaining in the oldResources map once the current snapshot events have been processed,
// indicates entries that were deleted during the resync - see corresponding code in
// finishResync().
func (wc *resyncCache) markAsValid(resourceKey string) {
	if wc.oldResources != nil {
		if oldResource, ok := wc.oldResources[resourceKey]; ok {
			wc.logger.WithField("Key", resourceKey).Debug("Marking key as re-processed")
			wc.resources[resourceKey] = oldResource
			delete(wc.oldResources, resourceKey)
		}
	}
}
