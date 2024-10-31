// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package syncer

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/deep-packet-inspection/pkg/dispatcher"
	bapi "github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

// syncerCallbacks implements the bapi.SyncerCallbacks interface.
// syncerCallbacks watches DeepPacketInspection and WorkLoadEndpoint resource and triggers a reconciliation
// whenever it spots changes to these resources.
type syncerCallbacks struct {
	ch       chan cacheRequest
	healthCh chan bool
}

const (
	keyPrefixDPI = "DeepPacketInspection"
	// bufferQueueSize is set to twice the size of typha's max batch size
	bufferQueueSize = 200
)

// requestType to indicate if the update is related to status of syncerCallbacks or regular resource update.
type requestType int

const (
	updateSyncStatus requestType = iota
	updateResource
)

// cacheRequest is used to communicate the updates from syncerCallbacks to the local cache.
type cacheRequest struct {
	inSync      bool
	requestType requestType
	updateType  bapi.UpdateType
	kvPair      model.KVPair
}

func NewSyncerCallbacks(healthCh chan bool) *syncerCallbacks {
	return &syncerCallbacks{
		ch:       make(chan cacheRequest, bufferQueueSize),
		healthCh: healthCh,
	}
}

// Sync is the main reconciliation loop, it loops until done. During start up when syncerCallbacks is not in-Sync,
// it will cache all the resource updates, once syncerCallbacks is in-Sync it passes the cached resources to
// dispatcher that will consume the data and then clears the cache.
// If the syncerCallbacks is in-Sync, the resource update received is directly passed to the dispatcher that will consume the data.
func (r *syncerCallbacks) Sync(ctx context.Context, dispatch dispatcher.Dispatcher) {
	defer close(r.ch)

	var cache []cacheRequest
	var inSync bool
	for {
		select {
		case req := <-r.ch:
			switch req.requestType {
			case updateSyncStatus:
				inSync = req.inSync
				if inSync && len(cache) != 0 {
					// If in-Sync request is received, send a copy of all the cached entries to dispatcher for processing
					// and clean the cache.
					log.Debugf("Processing the Sync request for cached entries %#v", cache)
					dispatch.Dispatch(ctx, convertSyncerToHandlerCache(cache))
					cache = []cacheRequest{}
				}
			case updateResource:
				if inSync {
					// If already in-Sync with syncerCallbacks server, send the received resource to dispatcher.
					dispatch.Dispatch(ctx, convertSyncerToHandlerCache([]cacheRequest{req}))
				} else {
					// If not in-Sync, cache the received resource till in-Sync request is received.
					cache = append(cache, req)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// convertSyncerToHandlerCache copies all the items in syncerCallbacks cache to dispatcher and returns the copy.
func convertSyncerToHandlerCache(sycnerCache []cacheRequest) []dispatcher.CacheRequest {
	var ctrlCache []dispatcher.CacheRequest
	for _, i := range sycnerCache {
		ctrlCache = append(ctrlCache, dispatcher.CacheRequest{
			UpdateType: i.updateType,
			KVPair:     i.kvPair,
		})
	}
	return ctrlCache
}

// OnStatusUpdated handles changes to the Sync status of the datastore.
func (r *syncerCallbacks) OnStatusUpdated(status bapi.SyncStatus) {
	req := cacheRequest{requestType: updateSyncStatus}
	if status == bapi.InSync {
		req.inSync = true
	} else {
		req.inSync = false
	}
	r.ch <- req
}

// OnUpdates handles the resource updates.
func (r *syncerCallbacks) OnUpdates(updates []bapi.Update) {
	for _, u := range updates {
		// Handle WorkloadEndpoint resource
		if _, ok := u.Key.(model.WorkloadEndpointKey); ok {
			switch u.UpdateType {
			case bapi.UpdateTypeKVDeleted, bapi.UpdateTypeKVNew, bapi.UpdateTypeKVUpdated:
				r.healthCh <- true
				r.ch <- cacheRequest{
					requestType: updateResource,
					updateType:  u.UpdateType,
					kvPair:      u.KVPair,
				}
			default:
				log.Warningf("Unexpected update type on resource: %s", u.Key)
				r.healthCh <- true
			}
			continue
		}

		// Handle DeepPacketInspection resource
		if strings.HasPrefix(u.Key.String(), keyPrefixDPI) {
			switch u.UpdateType {
			case bapi.UpdateTypeKVDeleted, bapi.UpdateTypeKVNew, bapi.UpdateTypeKVUpdated:
				r.healthCh <- true
				r.ch <- cacheRequest{
					requestType: updateResource,
					updateType:  u.UpdateType,
					kvPair:      u.KVPair,
				}
			default:
				log.Warningf("Unexpected update type on resource: %s", u.Key)
				r.healthCh <- false
			}
			continue
		}

		// Ignore update on other resources
		log.Warningf("Unexpected data with key %s on update", u.Key)
		r.healthCh <- false
	}
}
