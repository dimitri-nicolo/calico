// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package remotecluster

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
)

// The callbacks used by the remote endpoint watchers
type remoteEndpointCallbacks struct {
	wrappedCallbacks api.SyncerCallbacks
	rci              RemoteClusterInterface
	insync           func()
	clusterName      string
	syncErr          func(error)
	resync           func()
}

func (rec *remoteEndpointCallbacks) OnStatusUpdated(status api.SyncStatus) {
	switch status {
	case api.WaitForDatastore:
	case api.InSync:
		rec.insync()
	case api.ResyncInProgress:
		rec.resync()
	default:
		log.Warnf("Unknown event type: %s", status)
	}
}

// Resources from remote clusters need special handling.
func (rec *remoteEndpointCallbacks) OnUpdates(updates []api.Update) {
	// Use the RemoteClusterInterface to convert the updates.
	updates = rec.rci.ConvertUpdates(rec.clusterName, updates)
	if len(updates) > 0 {
		rec.wrappedCallbacks.OnUpdates(updates)
	}
}

// Resources from remote clusters need to handle connection failures
// so they do not block "InSync" status
func (rec *remoteEndpointCallbacks) SyncFailed(err error) {
	rec.syncErr(err)
}
