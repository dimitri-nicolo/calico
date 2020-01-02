// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package remotecluster

import (
	"context"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"
)

// Time to wait before retry failed connections to datastores.
const retrySeconds = 10 * time.Second

// RemoteClusterInterface provides appropriate hooks for the syncer to:
// - get the Calico API config from the RemoteClusterConfig, returning nil if the RemoteClusterConfig
//   is not valid for this syncer
// - create the appropriate resource types for the remote syncer
// - modify the remote syncer updates (e.g. modifying names to include the cluster name to avoid
//   naming conflicts across the clusters).
type RemoteClusterInterface interface {
	GetCalicoAPIConfig(*apiv3.RemoteClusterConfiguration) *apiconfig.CalicoAPIConfig
	CreateResourceTypes() []watchersyncer.ResourceType
	ConvertUpdates(clusterName string, updates []api.Update) []api.Update
}

// RemoteClusterClientInterface is an optional interface that the supplied remote cluster interface
// may also implement. If supported, this is used to create the appropriate backend client from the
// CalicoAPIConfig.
type RemoteClusterClientInterface interface {
	CreateClient(config apiconfig.CalicoAPIConfig) (api.Client, error)
}

func NewWrappedCallbacks(callbacks api.SyncerCallbacks, rci RemoteClusterInterface) api.SyncerCallbacks {
	// Store remotes as they are created so that they can be stopped.
	// A non-thread safe map is fine, because a mutex is used when it's accessed.
	remotes := make(map[model.Key]*RemoteSyncer)
	return &wrappedCallbacks{callbacks: callbacks, remotes: remotes, rci: rci}
}

// The callbacks used for remote cluster configs watcher
type wrappedCallbacks struct {
	// Interface used to handle specific remote cluster creation and post-processing.
	rci RemoteClusterInterface

	// The syncer callback interface.
	callbacks api.SyncerCallbacks

	// A map of remote clusters (and some associated tracking information)
	remotes map[model.Key]*RemoteSyncer

	// The lock should be used for all accesses to the remotes map. It's also used for some coordination between adding and deleting remote clusters.
	lock sync.Mutex

	// A count of remote clusters to wait for insync messages from before the local cluster's insync in passed through.
	activeUnsyncedRemotes sync.WaitGroup

	// Set to true once the initial list of RCCs is fetched.
	allRCCsAreSynced bool
}

type RemoteSyncer struct {
	// The watchersyncer used to get updates from this remote cluster.
	syncer api.Syncer

	// The client used by the watchersyncer
	client api.Client

	// The cancel function can be called to stop attempting to connect to this remote.
	cancel context.CancelFunc

	// If the remote can be connected to then it will block the insync status coming from the local cluster.
	// Once any error is received from the remote, then it no longer blocks.
	shouldBlockInsync bool

	// The datastore configuration for this cluster.
	datastoreConfig *apiconfig.CalicoAPIConfig
}

func (a *wrappedCallbacks) OnStatusUpdated(status api.SyncStatus) {
	if status != api.InSync {
		a.callbacks.OnStatusUpdated(status)
	} else {
		// The InSync status should only be passed through if any configured remote clusters have had a chance to connect.
		// Spawn a goroutine to wait for the conditions to avoid blocking the calling goroutine.
		log.Info("Local datastore is synced, waiting for remotes.")
		// Don't block on new remote clusters added from this point.
		a.lock.Lock()
		defer a.lock.Unlock()
		a.allRCCsAreSynced = true

		go func(activeUnsyncedRemotes *sync.WaitGroup) {
			log.Info("--> Waiting for remote datastores...")
			activeUnsyncedRemotes.Wait()
			log.Info("--> Remote datastores are synced.")
			a.callbacks.OnStatusUpdated(status)
		}(&a.activeUnsyncedRemotes)
	}
}

func (a *wrappedCallbacks) OnUpdates(updates []api.Update) {
	// Handle remote cluster configs separately. Other updates are passed through. Optimize for avoiding additional
	// memory allocations if there are no RCCs present in the updates slice.
	indicesToRemove := make(map[int]bool)
	for i, update := range updates {
		switch t := update.Key.(type) {
		default:
		case model.ResourceKey:
			if t.Kind == apiv3.KindRemoteClusterConfiguration {
				a.handleRCCUpdate(update)
				indicesToRemove[i] = true
			}
		}
	}

	if len(indicesToRemove) > 0 {
		// Remove the RCCs from the updates list.
		filteredUpdates := make([]api.Update, 0, len(updates)-len(indicesToRemove))
		for i, update := range updates {
			// Check if the index should be kept
			if _, ok := indicesToRemove[i]; !ok {
				filteredUpdates = append(filteredUpdates, update)
			}
		}
		updates = filteredUpdates
	}

	if len(updates) > 0 {
		a.callbacks.OnUpdates(updates)
	}
}

func (a *wrappedCallbacks) handleRCCUpdate(update api.Update) {
	clog := log.WithField("Key", update.Key)
	key := update.Key.(model.ResourceKey)

	// Lock over a remote cluster configuration update.
	a.lock.Lock()
	defer a.lock.Unlock()

	switch update.UpdateType {
	case api.UpdateTypeKVNew:
		// New remote cluster update. Use the RemoteClusterInterface to obtain the CalicoAPIConfig.
		// If this returns nil, then this remote cluster is excluded from the syncer.
		clog.Debug("Received new RCC update")
		datastoreConfig := a.rci.GetCalicoAPIConfig(update.Value.(*apiv3.RemoteClusterConfiguration))
		if datastoreConfig != nil {
			clog.Info("Handling new valid RCC update")
			a.newRCC(key, datastoreConfig)
		}
	case api.UpdateTypeKVDeleted:
		// Delete the remote cluster if it was previously valid.
		clog.Debug("Received delete RCC update")
		_, wasValid := a.remotes[key]
		if wasValid {
			clog.Info("Handling delete valid RCC update")
			a.deleteRCC(key)
		}
	case api.UpdateTypeKVUpdated:
		// Updates are only partially handled. If the remote cluster config is modified such that
		// the validity (i.e. whether or not the remote cluster will be used in the syncer) changes
		// then treat that as a creation or a deletion. The existence of the entry in the remotes
		// map indicates that it was previously valid.
		//
		// Updates to the connection configuration are not supported, so just log. To support this, the code would need
		// to handle the following.
		// - If the config is updated to point at a new cluster then the endpoints contained there need to be switched out (atomically?)
		// - If the config is updated to just change the connection info, then a new "client" needs to be created.
		//   But switching out clients for updated connection info isn't generally supported so don't handle it here.
		clog.Debug("Received modified RCC update")

		// Use the RemoteClusterInterface to obtain the CalicoAPIConfig. If this returns nil, then this remote cluster
		// is excluded from the syncer (i.e. it's not valid)
		datastoreConfig := a.rci.GetCalicoAPIConfig(update.Value.(*apiv3.RemoteClusterConfiguration))
		isValid := datastoreConfig != nil

		// Get the existing remote data for this cluster. The existence of this data in the remotes map indicates that
		// this cluster was previously valid for this syncer.
		remote, wasValid := a.remotes[update.Key]

		if isValid && !wasValid {
			// It is now valid, but was not previously. Treat as a new cluster.
			clog.Info("Handling modified RCC update as a new valid RCC")
			a.newRCC(key, datastoreConfig)
		} else if !isValid && wasValid {
			// It is now not valid, but was previously. Treat as a deleted cluster.
			clog.Info("Handling modified RCC update as a delete valid RCC")
			a.deleteRCC(key)
		} else if isValid && wasValid && !reflect.DeepEqual(remote.datastoreConfig, datastoreConfig) {
			// It was valid before and is still valid, and the datastore config has changed. Log and send status update
			// warn that the change requires a restart.
			log.Warnf("Received update for %s. Restart process to pick up changes to the connection data.", key)
			a.callbacks.OnUpdates([]api.Update{{
				KVPair: model.KVPair{
					Key: model.RemoteClusterStatusKey{Name: key.Name},
					Value: &model.RemoteClusterStatus{
						Status: model.RemoteClusterConfigChangeRestartRequired,
					},
				},
				UpdateType: api.UpdateTypeKVUpdated,
			}})
		}
	default:
		clog.Warnf("Unknown update type received: %s", update.UpdateType)
	}
}

// Create and start a watchersyncer using the config in the update.
func (a *wrappedCallbacks) newRCC(key model.ResourceKey, datastoreConfig *apiconfig.CalicoAPIConfig) {
	// Send a status update for the remote cluster indicating that it is starting connection processing. We do
	// this synchronously from the RCC update thread to ensure this is the first event for each RCC.
	a.callbacks.OnUpdates([]api.Update{{
		KVPair: model.KVPair{
			Key: model.RemoteClusterStatusKey{Name: key.Name},
			Value: &model.RemoteClusterStatus{
				Status: model.RemoteClusterConnecting,
			},
		},
		UpdateType: api.UpdateTypeKVNew,
	}})

	ctx, cancel := context.WithCancel(context.Background())
	if a.allRCCsAreSynced {
		// The initial list of remote clusters are synced. This update is after the initial sync so it shouldn't block.
		a.remotes[key] = &RemoteSyncer{cancel: cancel, shouldBlockInsync: false, datastoreConfig: datastoreConfig}
	} else {
		// Need to wait for this remote to sync. Done() is called on the wg when the in-sync message is received.
		a.remotes[key] = &RemoteSyncer{cancel: cancel, shouldBlockInsync: true, datastoreConfig: datastoreConfig}
		a.activeUnsyncedRemotes.Add(1)
	}
	go a.createRemoteSyncer(ctx, key, key.Name, datastoreConfig)
}

func (a *wrappedCallbacks) createRemoteSyncer(ctx context.Context, key model.Key, name string, datastoreConfig *apiconfig.CalicoAPIConfig) {
	// Create a backend client.
	// This can fail (e.g. if the remote cluster can't be reached) and should be retried in the background.
	// If there are any failures then Typha won't be blocked from starting, it will be allowed to start, potentially
	// losing remote endpoints from the dataplane.
	// The context will be marked as done if the resource is deleted.
	var backendClient api.Client
	var err error
	for backendClient == nil {
		// Create the client using the RemoteClusterClientInterface if the supplied helper supports
		// that interface, otherwise, just use the standard Calico client.
		if rcci, ok := a.rci.(RemoteClusterClientInterface); ok {
			backendClient, err = rcci.CreateClient(*datastoreConfig)
		} else {
			backendClient, err = backend.NewClient(*datastoreConfig)
		}

		if err != nil {
			// Hit an error. Handle by not blocking on this remote, and by sending a status update.
			log.Warnf("Could not connect to remote cluster. Will retry in %v: %s %v", retrySeconds, key, err)
			if done := a.handleConnectionFailed(ctx, key, name, err); done {
				log.Infof("Abandoning creation of syncer for %s", key)
				return
			}

			// Sleep and try later. We have already unblocked this remote in the call to handleConnectionFailure
			// above.
			select {
			case <-ctx.Done():
				log.Infof("Abandoning creation of syncer for %s", key)
				return
			case <-time.After(retrySeconds):
			}
		}
	}

	// The client connected. Create a watchersyncer for the remote and start it.
	// This is done as an atomic operation and only if the request isn't cancelled by a delete.
	a.lock.Lock()
	defer a.lock.Unlock()
	select {
	// Check the context again inside the lock. This will ensure that a syncer doesn't get created if it doesn't need to be.
	case <-ctx.Done():
		log.Infof("Abandoning creation of syncer for %s", key)
		if err := backendClient.Close(); err != nil {
			log.Warnf("Hit error closing client. Ignoring. %v", err)
		}
		a.finishRemote(key)
	default:
		log.Infof("Creating syncer for %s", key)

		// Resources that are fetched from remote clusters.  We call into the RemoteClusterInterface to obtain
		// this as it is dependent on the specific syncer.
		remoteResources := a.rci.CreateResourceTypes()

		remoteEndpointCallbacks := remoteEndpointCallbacks{
			wrappedCallbacks: a.callbacks,
			rci:              a.rci,
			clusterName:      name,
			insync:           func() { a.handleRemoteInSync(ctx, key, name) },
			syncErr:          func(err error) { a.handleConnectionFailed(ctx, key, name, err) },
			resync: func() {
				// Send a status update for this remote cluster, to indicate that we are synchronizing data.
				a.callbacks.OnUpdates([]api.Update{{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: name},
						Value: &model.RemoteClusterStatus{
							Status: model.RemoteClusterResyncInProgress,
						},
					},
					UpdateType: api.UpdateTypeKVUpdated,
				}})
			},
		}

		remoteWatcher := watchersyncer.New(backendClient, remoteResources, &remoteEndpointCallbacks)
		a.remotes[key].syncer = remoteWatcher
		a.remotes[key].client = backendClient
		remoteWatcher.Start()
	}
}

// handleRemoteInSync processes an in-sync event from a remote cluster syncer. This sends an in-sync status message
// for that cluster and then flags the cluster that we should no longer block waiting for it.
func (a *wrappedCallbacks) handleRemoteInSync(ctx context.Context, key model.Key, name string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	select {
	case <-ctx.Done():
		log.Infof("Remote cluster deleted, no need to send in-sync event: %s", key)
	default:
		log.Infof("Sending in-sync update for %s", key)
		// Send a status update to indicate that we are in-sync for a particular remote cluster.
		a.callbacks.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: name},
				Value: &model.RemoteClusterStatus{
					Status: model.RemoteClusterInSync,
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}})
	}
	a.finishRemote(key)
}

// handleConnectionFailed processes a connection failure by flagging that we should not block on this remote, and
// sending an error provided the remote has not been deleted. Returns true if the remote cluster has been
// deleted.
func (a *wrappedCallbacks) handleConnectionFailed(ctx context.Context, key model.Key, name string, err error) bool {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.finishRemote(key)
	select {
	case <-ctx.Done():
		log.Infof("Remote cluster deleted, no need to send connection failed event: %s", key)
		return true
	default:
		log.WithError(err).Infof("Sending connection failed update for %s", key)
		// Send a status update to indicate that the connection has failed to a particular remote cluster.
		a.callbacks.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: name},
				Value: &model.RemoteClusterStatus{
					Status: model.RemoteClusterConnectionFailed,
					Error:  err.Error(),
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}})
		return false
	}
}

// deleteRCC processes a remote cluster deletion. The internal lock is held by the caller.
func (a *wrappedCallbacks) deleteRCC(key model.ResourceKey) {
	// This will only be called when the remote cluster was previously valid and included in the syncer.
	remote := a.remotes[key]

	// Grab the syncer and client.
	syncer := remote.syncer
	client := remote.client

	// Never hold our lock when stopping the syncer (since that'll deadlock with the syncer callbacks).
	a.lock.Unlock()

	if syncer != nil {
		// Stop the watcher and generate a delete event for each item in the watch cache
		log.Infof("Stop syncer for %s", key)
		syncer.Stop()
	}

	if client != nil {
		// Close the client.
		log.Infof("Close client for %s", key)
		if err := client.Close(); err != nil {
			log.Warnf("Hit error closing client. Ignoring. %v", err)
		}
	}

	// Grab the lock again to finish off the processing.
	a.lock.Lock()

	// We released the lock, so we'd better check that the remote is still valid before continuing with the finish
	// processing.
	remote, ok := a.remotes[key]
	if ok {
		log.Infof("Cancel remote context for %s", key)
		remote.cancel()

		// Finish the remote (before deleting it)
		a.finishRemote(key)

		// Delete the remote from the list of remotes.
		delete(a.remotes, key)

		// Send a delete for the remote cluster status. We do this after all other deletion processing to ensure
		// no other events for this remote cluster occur after this deletion event.
		a.callbacks.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: key.Name},
			},
			UpdateType: api.UpdateTypeKVDeleted,
		}})
	}
}

// finishRemote signals that the remote should no longer block insync messages
func (a *wrappedCallbacks) finishRemote(key model.Key) {
	if a.remotes[key] != nil && a.remotes[key].shouldBlockInsync {
		log.Infof("--> Finish processing for remote cluster: %s", key)
		a.remotes[key].shouldBlockInsync = false
		a.activeUnsyncedRemotes.Done()
	}
}
