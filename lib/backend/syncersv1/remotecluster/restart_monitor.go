// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package remotecluster

import (
	"fmt"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

// RestartCallback is a function definition for passing to the RestartMonitor which
// will be called when a restart is required due to RemoteConfiguration changes
type RestartCallback func(reason string)

// RestartMonitor is a shim to watch for RemoteClusterConfigChangeRestartRequired status
// and upon seeing the status calling the restartCallback.
type RestartMonitor struct {
	wrappedCallbacks api.SyncerCallbacks
	restartCallback  RestartCallback
}

// NewRemoteClusterRestartMonitor creates a new RestartMonitor
func NewRemoteClusterRestartMonitor(callbacks api.SyncerCallbacks, rc RestartCallback) *RestartMonitor {
	return &RestartMonitor{
		wrappedCallbacks: callbacks,
		restartCallback:  rc,
	}
}

// OnStatusUpdated passes through OnStatusUpdated calls
func (rm *RestartMonitor) OnStatusUpdated(status api.SyncStatus) {
	rm.wrappedCallbacks.OnStatusUpdated(status)
}

// OnUpdates for RestartMonitor is only interested in RemoteClusterStatusKeys and
// when the status is ChangeRestartRequired will call the restartCallback
func (rm *RestartMonitor) OnUpdates(updates []api.Update) {
	for i := range updates {
		if k, ok := updates[i].KVPair.Key.(model.RemoteClusterStatusKey); ok {
			if v, ok := updates[i].KVPair.Value.(*model.RemoteClusterStatus); ok {
				if v.Status == model.RemoteClusterConfigChangeRestartRequired {
					rm.restartCallback(fmt.Sprintf("RemoteClusterConfiguration (%s) changed, restart required", k.Name))
				}
			}
		}
	}
	rm.wrappedCallbacks.OnUpdates(updates)
}
