// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package syncer

import (
	"context"
	"fmt"
	"strings"

	"github.com/tigera/deep-packet-inspection/pkg/handler"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/dpisyncer"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"

	"github.com/projectcalico/typha/pkg/buildinfo"
	"github.com/projectcalico/typha/pkg/syncclientutils"
	"github.com/projectcalico/typha/pkg/syncproto"
)

const (
	keyPrefixDPI = "DeepPacketInspection"
	// bufferQueueSize is set to twice the size of typha's max batch size
	bufferQueueSize = 200
)

// Run starts a long-running reconciler to sync deep packet inspection related resources.
// If typha is configured uses typha syncclient else uses local syncer.
func Run(ctx context.Context, hndler handler.Handler, nodeName string, healthCh chan bool, cfg *apiconfig.CalicoAPIConfig, calicoClient client.Interface) {
	var syncerClient bapi.Syncer
	r := newReconciler(nodeName, healthCh, cfg, calicoClient)

	// Either create a typha syncclient or a local syncerClient depending on configuration. This calls back into the
	// reconciler to trigger updates when necessary.

	// Read Typha settings from the environment.
	// When Typha is in use, there will already be variables prefixed with DPI_, so it's
	// convenient if we honor those variables.
	typhaConfig := syncclientutils.ReadTyphaConfig([]string{"DPI_"})
	if syncclientutils.MustStartSyncerClientIfTyphaConfigured(
		&typhaConfig, syncproto.SyncerTypeDPI,
		buildinfo.GitVersion, r.nodeName, fmt.Sprintf("dpi %s", buildinfo.GitVersion),
		r,
	) {
		log.Debug("Using typha syncclient")
	} else {
		log.Debug("Using local syncerClient")
		syncerClient = dpisyncer.New(r.client.(backendClientAccessor).Backend(), r)
		syncerClient.Start()
		defer syncerClient.Stop()
	}

	// Run the reconciler.
	r.run(ctx, hndler)
}

// requestType to indicate if the update is related to status of syncer or regular resource update.
type requestType int

const (
	updateSyncStatus requestType = iota
	updateResource
)

// reconciler watches DeepPacketInspection and WorkLoadEndpoint resource and triggers a reconciliation
// whenever it spots changes to these resources.
type reconciler struct {
	nodeName string
	cfg      *apiconfig.CalicoAPIConfig
	client   client.Interface
	ch       chan cacheRequest
	healthCh chan bool
}

// cacheRequest is used to communicate the updates from syncer to the local cache.
type cacheRequest struct {
	inSync      bool
	requestType requestType
	updateType  bapi.UpdateType
	kvPair      model.KVPair
}

func newReconciler(nodeName string, healthCh chan bool, cfg *apiconfig.CalicoAPIConfig, calicoClient client.Interface) *reconciler {
	return &reconciler{
		nodeName: nodeName,
		cfg:      cfg,
		client:   calicoClient,
		ch:       make(chan cacheRequest, bufferQueueSize),
		healthCh: healthCh,
	}
}

// run is the main reconciliation loop, it loops until done. During start up when syncer is not in-sync,
// it will cache all the resource updates, once syncer is in-sync it passes the cached resources to
// handler that will consume the data and then clears the cache.
// If the syncer is in-sync, the resource update received is directly passed to the handler that will consume the data.
func (r *reconciler) run(ctx context.Context, hndler handler.Handler) {
	defer close(r.ch)
	// Close the handler when syncer callback loop terminates
	defer hndler.Close()

	var cache []cacheRequest
	var inSync bool
	for {
		select {
		case req := <-r.ch:
			switch req.requestType {
			case updateSyncStatus:
				inSync = req.inSync
				if inSync && len(cache) != 0 {
					// If in-sync request is received, send a copy of all the cached entries to handler for processing
					// and clean the cache.
					log.Debugf("Processing the sync request for cached entries %#v", cache)
					hndler.OnUpdate(ctx, convertSyncerToHandlerCache(cache))
					cache = []cacheRequest{}
				}
			case updateResource:
				if inSync {
					// If already in-sync with syncer server, send the received resource to handler.
					hndler.OnUpdate(ctx, convertSyncerToHandlerCache([]cacheRequest{req}))
				} else {
					// If not in-sync, cache the received resource till in-sync request is received.
					cache = append(cache, req)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// convertSyncerToHandlerCache copies all the items in syncer cache to handler and returns the copy.
func convertSyncerToHandlerCache(sycnerCache []cacheRequest) []handler.CacheRequest {
	var ctrlCache []handler.CacheRequest
	for _, i := range sycnerCache {
		ctrlCache = append(ctrlCache, handler.CacheRequest{
			UpdateType: i.updateType,
			KVPair:     i.kvPair,
		})
	}
	return ctrlCache
}

// OnStatusUpdated handles changes to the sync status of the datastore.
func (r *reconciler) OnStatusUpdated(status bapi.SyncStatus) {
	req := cacheRequest{requestType: updateSyncStatus}
	if status == bapi.InSync {
		req.inSync = true
	} else {
		req.inSync = false
	}
	r.ch <- req
}

// OnUpdates handles the resource updates.
func (r *reconciler) OnUpdates(updates []bapi.Update) {
	for _, u := range updates {
		// Handle WorkloadEndpoint resource
		if k, ok := u.Key.(model.WorkloadEndpointKey); ok {
			if k.Hostname != r.nodeName {
				log.Debugf("Skipping WEP %s that does not belong to the current host", k.WorkloadID)
				continue
			}
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
		if k, ok := u.Key.(model.Key); ok && strings.HasPrefix(k.String(), keyPrefixDPI) {
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

// backendClientAccessor is an interface to access the backend client from the main v2 client.
type backendClientAccessor interface {
	Backend() bapi.Client
}
