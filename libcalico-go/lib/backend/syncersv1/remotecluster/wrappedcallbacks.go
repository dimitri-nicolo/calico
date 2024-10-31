// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.

package remotecluster

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/calico/libcalico-go/lib/backend"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/watchersyncer"
	validatorv3 "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
)

// Time to wait before retry failed connections to datastores.
const retrySeconds = 10 * time.Second

// Metrics for remote cluster config status.
var (
	statusToGaugeValue = map[model.RemoteClusterStatusType]float64{
		model.RemoteClusterConnectionFailed:            0,
		model.RemoteClusterConnecting:                  1,
		model.RemoteClusterInSync:                      2,
		model.RemoteClusterResyncInProgress:            3,
		model.RemoteClusterConfigChangeRestartRequired: 4,
		model.RemoteClusterConfigIncomplete:            5,
	}
)

var (
	statusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "remote_cluster_connection_status",
		Help: "0-NotConnecting ,1-Connecting, 2-InSync, 3-ReSyncInProgress, 4-ConfigChangeRestartRequired, 5-ConfigInComplete.",
	}, []string{"remote_cluster_name"})

	// prometheusRegisterOnce ensures New gauge vector is registered once.
	prometheusRegisterOnce sync.Once
)

// RemoteClusterInterface provides appropriate hooks for the syncer to:
//   - get the Calico API config from the RemoteClusterConfig, returning nil if the RemoteClusterConfig
//     is not valid for this syncer
//   - create the appropriate resource types for the remote syncer
//   - modify the remote syncer updates (e.g. modifying names to include the cluster name to avoid
//     naming conflicts across the clusters).
type RemoteClusterInterface interface {
	GetCalicoAPIConfig(*apiv3.RemoteClusterConfiguration) *apiconfig.CalicoAPIConfig
	CreateResourceTypes(overlayRoutingMode apiv3.OverlayRoutingMode) []watchersyncer.ResourceType
	ConvertUpdates(clusterName string, updates []api.Update) []api.Update
}

// RemoteClusterClientInterface is an optional interface that the supplied remote cluster interface
// may also implement. If supported, this is used to create the appropriate backend client from the
// CalicoAPIConfig.
type RemoteClusterClientInterface interface {
	CreateClient(config apiconfig.CalicoAPIConfig) (api.Client, error)
}

func NewWrappedCallbacks(callbacks api.SyncerCallbacks, k8sClientset *kubernetes.Clientset, rci RemoteClusterInterface) api.SyncerCallbacks {
	// Store remotes as they are created so that they can be stopped.
	// A non-thread safe map is fine, because a mutex is used when it's accessed.

	prometheusRegisterOnce.Do(func() {
		prometheus.MustRegister(statusGauge)
	})
	remotes := make(map[model.ResourceKey]*RemoteSyncer)
	wcb := wrappedCallbacks{callbacks: callbacks, remotes: remotes, rci: rci, statusGauge: statusGauge}
	sw := NewSecretWatcher(&wcb, k8sClientset)
	wcb.secretWatcher = sw
	return &wcb
}

// The callbacks used for remote cluster configs watcher
type wrappedCallbacks struct {
	// Interface used to handle specific remote cluster creation and post-processing.
	rci RemoteClusterInterface

	// The syncer callback interface.
	callbacks api.SyncerCallbacks

	// A map of remote clusters (and some associated tracking information)
	remotes map[model.ResourceKey]*RemoteSyncer

	// The lock should be used for all accesses to the remotes map. It's also used for some coordination between adding and deleting remote clusters.
	lock sync.Mutex

	// A count of remote clusters to wait for insync messages from before the local cluster's insync in passed through.
	activeUnsyncedRemotes sync.WaitGroup

	// Set to true once the initial list of RCCs is fetched.
	allRCCsAreSynced bool

	secretWatcher *secretWatcher

	// GaugeVec to capture the RCC status.
	statusGauge *prometheus.GaugeVec
}

type RemoteSyncer struct {
	// The watchersyncer used to get updates from this remote cluster.
	syncer api.Syncer

	// Flag set true while RemoteSyncer is being stopped to ensure two stops cannot
	// run in parallel for this syncer.
	beingStopped bool
	needsRemoval bool

	// The client used by the watchersyncer
	client api.Client

	// The cancel function can be called to stop attempting to connect to this remote.
	cancel context.CancelFunc

	// If the remote can be connected to then it will block the insync status coming from the local cluster.
	// Once any error is received from the remote, then it no longer blocks.
	shouldBlockInsync bool

	remoteClusterConfig *apiv3.RemoteClusterConfiguration

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
		rcConfig := update.Value.(*apiv3.RemoteClusterConfiguration)
		if rcConfig != nil {
			clog.Info("Handling new RCC update")

			// Send a status update for the remote cluster indicating that it is starting connection processing. We do
			// this synchronously from the RCC update thread to ensure this is the first event for each RCC.
			log.Debugf("Callback update for %s: %s", key, updateTypeToString(api.UpdateTypeKVNew))
			a.callbacks.OnUpdates([]api.Update{{
				KVPair: model.KVPair{
					Key: model.RemoteClusterStatusKey{Name: key.Name},
					Value: &model.RemoteClusterStatus{
						Status: model.RemoteClusterConnecting,
					},
				},
				UpdateType: api.UpdateTypeKVNew,
			}})
			a.reportRemoteClusterStatus(key.Name, model.RemoteClusterConnecting)
			a.updateRCC(key, rcConfig, "new RCC", clog)
		}
	case api.UpdateTypeKVDeleted:
		// Delete the remote cluster if it was previously valid.
		clog.Debug("Received delete RCC update")
		_, existed := a.remotes[key]
		if existed {
			clog.Info("Handling delete RCC update")
			a.stopRCC(key, true)
		}
	case api.UpdateTypeKVUpdated:
		clog.Debug("Received modified RCC update")

		rcConfig := update.Value.(*apiv3.RemoteClusterConfiguration)
		a.updateRCC(key, rcConfig, "modified RCC", clog)
	default:
		clog.Warnf("Unknown update type received: %s", updateTypeToString(update.UpdateType))
	}
}

func (a *wrappedCallbacks) updateRCC(key model.ResourceKey, newRCC *apiv3.RemoteClusterConfiguration, updateSrc string, log *log.Entry) {
	// Updates are only partially handled. If the remote cluster config is modified such that
	// the validity (i.e. whether or not the remote cluster will be used in the syncer) changes
	// then treat that as a creation or a deletion. The existence of the entry in the remotes
	// map and having a non-nil datastoreConfig indicates that it was previously valid.
	//
	// Updates to the connection configuration are not supported, so just log and trigger restart. To support this, the code would need
	// to handle the following.
	// - If the config is updated to point at a new cluster then the endpoints contained there need to be switched out (atomically?)
	// - If the config is updated to just change the connection info, then a new "client" needs to be created.
	//   But switching out clients for updated connection info isn't generally supported so don't handle it here.
	_, existed := a.remotes[key]
	if !existed {
		log.Infof("Adding RCC to remotes %v", key)
		a.remotes[key] = &RemoteSyncer{shouldBlockInsync: false}
	}
	remote := a.remotes[key]

	// If this returns nil, then this remote cluster is excluded from the syncer (i.e. it's not valid)
	datastoreConfig, err := a.getDatastoreConfig(newRCC)
	if err != nil {
		log.Warnf("Received %s. Unable to get datastore config: %s", updateSrc, err)
		log.Debugf("Callback update for %s: %s", key, updateTypeToString(api.UpdateTypeKVUpdated))
		a.callbacks.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: key.Name},
				Value: &model.RemoteClusterStatus{
					Status: model.RemoteClusterConfigIncomplete,
					Error:  err.Error(),
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}})
		a.reportRemoteClusterStatus(key.Name, model.RemoteClusterConfigIncomplete)
	} else if err == nil && datastoreConfig == nil {
		a.reportRemoteClusterStatus(key.Name, model.RemoteClusterConfigIncomplete)
		log.Warnf("Received %s. Cluster access secret was not found or the inline datastore config was invalid", updateSrc)
		log.Debugf("Callback update for %s: %s", key, updateTypeToString(api.UpdateTypeKVUpdated))
		a.callbacks.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: key.Name},
				Value: &model.RemoteClusterStatus{
					Status: model.RemoteClusterConfigIncomplete,
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}})
	}

	isValid := datastoreConfig != nil

	// Get the existing remote data for this cluster. If it is not present then add as a new RCC
	if !existed {
		// Treat as a new cluster.
		remote.remoteClusterConfig = newRCC
		log.Info("Handling modified RCC update as a new RCC")
		a.startRemoteSyncer(key, datastoreConfig)
		return
	}
	wasValid := remote.datastoreConfig != nil
	routingModeChanged := remote.remoteClusterConfig.Spec.SyncOptions.OverlayRoutingMode != newRCC.Spec.SyncOptions.OverlayRoutingMode
	remote.remoteClusterConfig = newRCC

	if isValid && !wasValid {
		// It is now valid, but was not previously. Treat as a new cluster.
		log.Infof("Handling %s as a valid RCC", updateSrc)
		a.startRemoteSyncer(key, datastoreConfig)
	} else if !isValid && wasValid {
		// It is now not valid, but was previously. Treat as a deleted cluster.
		log.Infof("Handling %s as invalidating remote cluster", updateSrc)
		a.stopRCC(key, false)
	} else if isValid && wasValid && (!reflect.DeepEqual(remote.datastoreConfig, datastoreConfig) || routingModeChanged) {
		// It was valid before and is still valid, and the datastore config has changed. Log and send status update
		// warn that the change requires a restart.
		log.Warnf("Received %s. Restart process to pick up changes to the connection data.", updateSrc)
		log.Debugf("Callback update for %s: %s", key, updateTypeToString(api.UpdateTypeKVUpdated))
		a.callbacks.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: key.Name},
				Value: &model.RemoteClusterStatus{
					Status: model.RemoteClusterConfigChangeRestartRequired,
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}})
		a.reportRemoteClusterStatus(key.Name, model.RemoteClusterConfigChangeRestartRequired)
	}
}

func (a *wrappedCallbacks) getDatastoreConfig(rcConfig *apiv3.RemoteClusterConfiguration) (*apiconfig.CalicoAPIConfig, error) {
	// If no secret is specified then everything is in the RCC so convert to a datstoreConfig
	if rcConfig.Spec.ClusterAccessSecret == nil {
		return a.rci.GetCalicoAPIConfig(rcConfig), nil
	}
	if a.secretWatcher == nil {
		return nil, fmt.Errorf("secret watcher not available, unable to get secrets for cluster access")
	}
	data, err := a.secretWatcher.GetSecretData(rcConfig.Spec.ClusterAccessSecret.Namespace, rcConfig.Spec.ClusterAccessSecret.Name)
	if err == nil && data == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rcc := apiv3.RemoteClusterConfiguration{
		ObjectMeta: rcConfig.ObjectMeta,
		Spec: apiv3.RemoteClusterConfigurationSpec{
			DatastoreType: string(data["datastoreType"]),
			EtcdConfig: apiv3.EtcdConfig{
				EtcdEndpoints: string(data["etcdEndpoints"]),
				EtcdUsername:  string(data["etcdUsername"]),
				EtcdPassword:  string(data["etcdPassword"]),
				EtcdKey:       string(data["etcdKey"]),
				EtcdCert:      string(data["etcdCert"]),
				EtcdCACert:    string(data["etcdCACert"]),
			},
			KubeConfig: apiv3.KubeConfig{
				KubeconfigInline: string(data["kubeconfig"]),
			},
		},
	}

	err = validatorv3.Validate(rcc)
	if err != nil {
		return nil, err
	}

	return a.rci.GetCalicoAPIConfig(&rcc), nil
}

// Create and start a watchersyncer using the config in the update.
func (a *wrappedCallbacks) startRemoteSyncer(key model.ResourceKey, datastoreConfig *apiconfig.CalicoAPIConfig) {
	// If datastoreConfig is nil then we can't start a syncer yet.
	if datastoreConfig == nil {
		// TODO: Should there be callback update sent for this case?
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.remotes[key].cancel = cancel
	a.remotes[key].datastoreConfig = datastoreConfig
	if a.allRCCsAreSynced {
		// The initial list of remote clusters are synced. This update is after the initial sync so it shouldn't block.
		a.remotes[key].shouldBlockInsync = false
	} else {
		// Need to wait for this remote to sync. Done() is called on the wg when the in-sync message is received.
		a.remotes[key].shouldBlockInsync = true
		a.activeUnsyncedRemotes.Add(1)
	}
	go a.createRemoteSyncer(ctx, key, datastoreConfig)
}

func (a *wrappedCallbacks) createRemoteSyncer(ctx context.Context, key model.ResourceKey, datastoreConfig *apiconfig.CalicoAPIConfig) {
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
			if done := a.handleConnectionFailed(ctx, key, err); done {
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
		overlayRoutingMode := a.remotes[key].remoteClusterConfig.Spec.SyncOptions.OverlayRoutingMode
		remoteResources := a.rci.CreateResourceTypes(overlayRoutingMode)

		remoteEndpointCallbacks := remoteEndpointCallbacks{
			wrappedCallbacks: a.callbacks,
			rci:              a.rci,
			clusterName:      key.Name,
			insync:           func() { a.handleRemoteInSync(ctx, key) },
			syncErr:          func(err error) { a.handleConnectionFailed(ctx, key, err) },
			resync: func() {
				// Send a status update for this remote cluster, to indicate that we are synchronizing data.
				a.callbacks.OnUpdates([]api.Update{{
					KVPair: model.KVPair{
						Key: model.RemoteClusterStatusKey{Name: key.Name},
						Value: &model.RemoteClusterStatus{
							Status: model.RemoteClusterResyncInProgress,
						},
					},
					UpdateType: api.UpdateTypeKVUpdated,
				}})
				a.reportRemoteClusterStatus(key.Name, model.RemoteClusterResyncInProgress)
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
func (a *wrappedCallbacks) handleRemoteInSync(ctx context.Context, key model.ResourceKey) {
	a.lock.Lock()
	defer a.lock.Unlock()
	select {
	case <-ctx.Done():
		log.Infof("Remote cluster deleted, no need to send in-sync event: %s", key)
	default:
		log.Infof("Sending in-sync update for %s", key)
		// Send a status update to indicate that we are in-sync for a particular remote cluster.
		log.Debugf("Callback update for %s: %s", key, updateTypeToString(api.UpdateTypeKVUpdated))
		a.callbacks.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: key.Name},
				Value: &model.RemoteClusterStatus{
					Status: model.RemoteClusterInSync,
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}})
		a.reportRemoteClusterStatus(key.Name, model.RemoteClusterInSync)
	}
	a.finishRemote(key)
}

// handleConnectionFailed processes a connection failure by flagging that we should not block on this remote, and
// sending an error provided the remote has not been deleted. Returns true if the remote cluster has been
// deleted.
func (a *wrappedCallbacks) handleConnectionFailed(ctx context.Context, key model.ResourceKey, err error) bool {
	a.lock.Lock()
	defer func() {
		a.finishRemote(key)
		a.lock.Unlock()
	}()

	select {
	case <-ctx.Done():
		log.Infof("Remote cluster deleted, no need to send connection failed event: %s", key)
		return true
	default:
		log.WithError(err).Infof("Sending connection failed update for %s", key)
		// Send a status update to indicate that the connection has failed to a particular remote cluster.
		log.Debugf("Callback update for %s: %s", key, updateTypeToString(api.UpdateTypeKVUpdated))
		a.callbacks.OnUpdates([]api.Update{{
			KVPair: model.KVPair{
				Key: model.RemoteClusterStatusKey{Name: key.Name},
				Value: &model.RemoteClusterStatus{
					Status: model.RemoteClusterConnectionFailed,
					Error:  err.Error(),
				},
			},
			UpdateType: api.UpdateTypeKVUpdated,
		}})
		a.reportRemoteClusterStatus(key.Name, model.RemoteClusterConnectionFailed)
		return false
	}
}

// stopRCC processes a remote cluster deletion. The internal lock is held by the caller.
func (a *wrappedCallbacks) stopRCC(key model.ResourceKey, remove bool) {
	// This will only be called when the remote cluster was previously valid and included in the syncer.
	remote := a.remotes[key]

	// Do this before we check if it is already beingStopped, so that if the removal
	// stopRCC returns because it is already being stopped this remote will be removed anyway.
	if remove {
		remote.needsRemoval = remove
	}
	if remote.beingStopped {
		// This remote is already being stopped, nothing to do.
		return
	}
	remote.beingStopped = true
	defer func() { remote.beingStopped = false }()

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
		if remote.cancel != nil {
			log.Infof("Cancel remote context for %s", key)
			remote.cancel()

			// Finish the remote (before deleting it)
			a.finishRemote(key)

			remote.syncer = nil
			remote.client = nil
			remote.datastoreConfig = nil
		} else {
			log.Infof("Cancel remote context for %s was not needed", key)
		}
		if remote.needsRemoval {
			// Delete the remote from the list of remotes.
			delete(a.remotes, key)

			// Send a delete for the remote cluster status. We do this after all other deletion processing to ensure
			// no other events for this remote cluster occur after this deletion event.
			log.Debugf("Callback update for %s: %s", key, updateTypeToString(api.UpdateTypeKVDeleted))
			a.callbacks.OnUpdates([]api.Update{{
				KVPair: model.KVPair{
					Key: model.RemoteClusterStatusKey{Name: key.Name},
				},
				UpdateType: api.UpdateTypeKVDeleted,
			}})
			// Delete the Gauge once the remote cluster is removed to avoid memory leak.
			if a.statusGauge != nil {
				log.Infof("Deleting the status gauge for cluster: %s", key)
				a.statusGauge.DeleteLabelValues(strings.ToLower(key.Name))
			}
		} else {
			log.Debugf("Callback update for %s: %s", key, updateTypeToString(api.UpdateTypeKVUpdated))
			a.callbacks.OnUpdates([]api.Update{{
				KVPair: model.KVPair{
					Key: model.RemoteClusterStatusKey{Name: key.Name},
					Value: &model.RemoteClusterStatus{
						Status: model.RemoteClusterConfigIncomplete,
						Error:  "Config is incomplete, stopping watch remote",
					},
				},
				UpdateType: api.UpdateTypeKVUpdated,
			}})
			a.reportRemoteClusterStatus(key.Name, model.RemoteClusterConfigIncomplete)
		}
	}
	a.cleanStaleSecrets()
}

// finishRemote signals that the remote should no longer block insync messages
func (a *wrappedCallbacks) finishRemote(key model.ResourceKey) {
	if a.remotes[key] != nil && a.remotes[key].shouldBlockInsync {
		log.Infof("--> Finish processing for remote cluster: %s", key)
		a.remotes[key].shouldBlockInsync = false
		a.activeUnsyncedRemotes.Done()
	}
}

func (a *wrappedCallbacks) OnSecretUpdated(namespace, name string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	for k, v := range a.remotes {
		if v.remoteClusterConfig == nil {
			continue
		}

		if v.remoteClusterConfig.Spec.ClusterAccessSecret == nil {
			continue
		}

		secretRef := v.remoteClusterConfig.Spec.ClusterAccessSecret
		if secretRef.Namespace == namespace && secretRef.Name == name {
			fields := log.Fields{"Namespace": namespace, "Secret": name}
			l := log.WithFields(fields)
			a.updateRCC(k, v.remoteClusterConfig, "Secret update", l)
		}
	}

	a.cleanStaleSecrets()
}

func (a *wrappedCallbacks) cleanStaleSecrets() {
	if a.secretWatcher == nil {
		return
	}
	// Is it too often to clean up stale on every Secret update?
	a.secretWatcher.MarkStale()
	for _, v := range a.remotes {
		if v.remoteClusterConfig == nil {
			continue
		}

		if v.remoteClusterConfig.Spec.ClusterAccessSecret == nil {
			continue
		}

		secretRef := v.remoteClusterConfig.Spec.ClusterAccessSecret
		_, _ = a.secretWatcher.GetSecretData(secretRef.Namespace, secretRef.Name)
	}
	a.secretWatcher.SweepStale()
}

func updateTypeToString(ut api.UpdateType) string {
	switch ut {
	case api.UpdateTypeKVUnknown:
		return "UpdateTypeKVUnknown"
	case api.UpdateTypeKVNew:
		return "UpdateTypeKVNew"
	case api.UpdateTypeKVUpdated:
		return "UpdateTypeKVUpdated"
	case api.UpdateTypeKVDeleted:
		return "UpdateTypeKVDeleted"
	}
	return "UknownUpdateType"
}

// reportRemoteClusterStatus reports all the remote cluster status.
func (a *wrappedCallbacks) reportRemoteClusterStatus(key string, status model.RemoteClusterStatusType) {
	if a.statusGauge != nil {
		log.Debugf("Reporting remote cluster Status for cluster: %s - Status: %v ", key, statusToGaugeValue[status])
		a.statusGauge.WithLabelValues(strings.ToLower(key)).Set(statusToGaugeValue[status])
	}
}
