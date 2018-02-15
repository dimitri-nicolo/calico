package felixsyncer

import (
	"context"
	"sync"
	"time"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"
	log "github.com/sirupsen/logrus"
)

// Time to wait before retry failed connections to datastores.
const retrySeconds = 10 * time.Second

func NewWrappedCallbacks(callbacks api.SyncerCallbacks) api.SyncerCallbacks {
	// Store remotes as they are created so that they can be stopped.
	// A non-thread safe map is fine, because a mutex is used when it's accessed.
	remotes := make(map[model.Key]*RemoteSyncer)
	return &wrappedCallbacks{callbacks: callbacks, remotes: remotes}
}

// The callbacks used for remote cluster configs watcher
type wrappedCallbacks struct {
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

	// The cancel function can be called to stop attempting to connect to this remote.
	cancel context.CancelFunc

	// If the remote can be connected to then it will block the insync status coming from the local cluster.
	// Once any error is received from the remote, then it not longer blocks.
	shouldBlockInsync bool
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
			activeUnsyncedRemotes.Wait()
			log.Info("Remote datastores are synced.")
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
	switch update.UpdateType {
	case api.UpdateTypeKVNew:
		log.Infof("Received new for %s.", update.Key)
		a.newRCC(update)
	case api.UpdateTypeKVDeleted:
		log.Infof("Received delete for %s.", update.Key)
		a.deleteRCC(update)
	case api.UpdateTypeKVUpdated:
		// Updates aren't handled, so just log. To handle this, the code would need to handle the following.
		// - If the config is updated to point at a new cluster then the endpoints contained there need to be switched out (atomically?)
		// - If the config is updated to just change the connection info, then a new "client" needs to be created.
		//   But switching out clients for updated connection info isn't generally supported so don't handle it here.
		log.Warnf("Received update for %s. Restart process to pick up changes.", update.Key)
	default:
		log.Warnf("Unknown update type received: %s", update.UpdateType)
	}
}

// Create and start a watchersyncer using the config in the update.
func (a *wrappedCallbacks) newRCC(update api.Update) {
	config := update.Value.(*apiv3.RemoteClusterConfiguration)
	datastoreConfig := convertRCCToCalicoAPIConfig(config)

	// Lock to create the entry in the remotes map.
	a.lock.Lock()
	defer a.lock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	if a.allRCCsAreSynced {
		// The initial list of remote clusters are synced. This update is after the initial sync so it shouldn't block.
		a.remotes[update.Key] = &RemoteSyncer{cancel: cancel, shouldBlockInsync: false}
	} else {
		// Need to wait for this remote to sync. Done() is called on the wg when the in-sync message is received.
		a.remotes[update.Key] = &RemoteSyncer{cancel: cancel, shouldBlockInsync: true}
		a.activeUnsyncedRemotes.Add(1)
	}
	go a.createRemoteSyncer(ctx, update.Key, config.Name, datastoreConfig)
}

func (a *wrappedCallbacks) createRemoteSyncer(ctx context.Context, key model.Key, name string, datastoreConfig *apiconfig.CalicoAPIConfig) {
	// Create a backend client.
	// This can fail (e.g. if the remote cluster can't be reached) and should be retried in the background.
	// If there are any failures then Typha won't be blocked from starting, it will be allowed to start, potentially
	// losing remote endpoints form the dataplane.
	// The context will be marked as done if the resource is deleted.
	var backendClient api.Client
	for backendClient == nil {
		var err error
		backendClient, err = backend.NewClient(*datastoreConfig)
		if err != nil {
			// Hit an error, don't block on this remote. Sleep and try later.
			log.Warnf("Could not connect to remote cluster. Will retry in %v: %s %v", retrySeconds, key, err)
			a.finishRemote(key, false)
			select {
			case <-ctx.Done():
				log.Infof("Abandoning creation of syncer for %s", key)
				a.finishRemote(key, false)
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
		a.finishRemote(key, true)
	default:
		log.Infof("Creating syncer for %s", key)

		// Resources that are fetched from remote clusters
		remoteResources := []watchersyncer.ResourceType{
			{
				ListInterface:   model.ResourceListOptions{Kind: apiv3.KindWorkloadEndpoint},
				UpdateProcessor: updateprocessors.NewWorkloadEndpointUpdateProcessor(),
			},
			{
				ListInterface:   model.ResourceListOptions{Kind: apiv3.KindHostEndpoint},
				UpdateProcessor: updateprocessors.NewHostEndpointUpdateProcessor(),
			},
			{
				ListInterface:   model.ResourceListOptions{Kind: apiv3.KindProfile},
				UpdateProcessor: updateprocessors.NewProfileUpdateProcessor(),
			},
		}

		remoteEndpointCallbacks := remoteEndpointCallbacks{
			wrappedCallbacks: a.callbacks,
			clusterName:      name,
			insync:           func() { a.finishRemote(key, true) },
		}

		remoteWatcher := watchersyncer.New(backendClient, remoteResources, &remoteEndpointCallbacks)
		a.remotes[key].syncer = remoteWatcher
		remoteWatcher.Start()
	}
}

func (a *wrappedCallbacks) deleteRCC(update api.Update) {
	a.lock.Lock()
	defer a.lock.Unlock()
	// The key in remote will be guaranteed to exist, since a delete can't be sent before an update.
	// Call cancel() so any update processing happening in the background can stop.
	a.remotes[update.Key].cancel()
	if a.remotes[update.Key].syncer != nil {
		// Stop the watcher and generate a delete event for each item in the watch cache
		a.remotes[update.Key].syncer.Stop()

		// Finish the remote (before deleting it)
		a.finishRemote(update.Key, true)

		// Delete the remote from the list of remotes.
		delete(a.remotes, update.Key)
	}
}

func (a *wrappedCallbacks) finishRemote(key model.Key, alreadyLocked bool) {
	if !alreadyLocked {
		a.lock.Lock()
		defer a.lock.Unlock()
	}
	// Mark that the remote should no longer block insync messages
	if a.remotes[key] != nil && a.remotes[key].shouldBlockInsync {
		a.remotes[key].shouldBlockInsync = false
		a.activeUnsyncedRemotes.Done()
	}
}

func convertRCCToCalicoAPIConfig(config *apiv3.RemoteClusterConfiguration) *apiconfig.CalicoAPIConfig {
	datastoreConfig := apiconfig.NewCalicoAPIConfig()
	datastoreConfig.Spec.DatastoreType = apiconfig.DatastoreType(config.Spec.DatastoreType)
	datastoreConfig.Spec.EtcdEndpoints = config.Spec.EtcdEndpoints
	datastoreConfig.Spec.EtcdUsername = config.Spec.EtcdUsername
	datastoreConfig.Spec.EtcdPassword = config.Spec.EtcdPassword
	datastoreConfig.Spec.EtcdKeyFile = config.Spec.EtcdKeyFile
	datastoreConfig.Spec.EtcdCertFile = config.Spec.EtcdCertFile
	datastoreConfig.Spec.EtcdCACertFile = config.Spec.EtcdCACertFile
	datastoreConfig.Spec.Kubeconfig = config.Spec.Kubeconfig
	datastoreConfig.Spec.K8sAPIEndpoint = config.Spec.K8sAPIEndpoint
	datastoreConfig.Spec.K8sKeyFile = config.Spec.K8sKeyFile
	datastoreConfig.Spec.K8sCertFile = config.Spec.K8sCertFile
	datastoreConfig.Spec.K8sCAFile = config.Spec.K8sCAFile
	datastoreConfig.Spec.K8sAPIToken = config.Spec.K8sAPIToken
	datastoreConfig.Spec.K8sInsecureSkipTLSVerify = config.Spec.K8sInsecureSkipTLSVerify
	return datastoreConfig
}
