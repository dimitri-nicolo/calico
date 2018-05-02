// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package felixsyncer

import (
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
)

// The callbacks used by the remote endpoint watchers
type remoteEndpointCallbacks struct {
	wrappedCallbacks api.SyncerCallbacks
	insync           func()
	clusterName      string
}

func (rec *remoteEndpointCallbacks) OnStatusUpdated(status api.SyncStatus) {
	switch status {
	case api.WaitForDatastore:
	case api.InSync:
		rec.insync()
	case api.ResyncInProgress:
	default:
		log.Warnf("Unknown event type: %s", status)
	}
}

// Resources from remote clusters need special handling.
func (rec *remoteEndpointCallbacks) OnUpdates(updates []api.Update) {
	// Profiles have the rules removed (federated is just about identity) and the name is prefixed with the clustername.
	// Endpoints get the clustername prefixed to the hostname and any profile references are updated.
	for i, update := range updates {
		if update.UpdateType == api.UpdateTypeKVUpdated || update.UpdateType == api.UpdateTypeKVNew {
			switch t := update.Key.(type) {
			default:
				log.Warnf("unexpected type %T\n", t)
			case model.HostEndpointKey:
				t.Hostname = rec.clusterName + "/" + t.Hostname
				updates[i].Key = t
				for profileIndex, profile := range updates[i].Value.(*model.HostEndpoint).ProfileIDs {
					updates[i].Value.(*model.HostEndpoint).ProfileIDs[profileIndex] = rec.clusterName + "/" + profile
				}
			case model.WorkloadEndpointKey:
				t.Hostname = rec.clusterName + "/" + t.Hostname
				updates[i].Key = t
				for profileIndex, profile := range updates[i].Value.(*model.WorkloadEndpoint).ProfileIDs {
					updates[i].Value.(*model.WorkloadEndpoint).ProfileIDs[profileIndex] = rec.clusterName + "/" + profile
				}
			case model.ProfileRulesKey:
				t.Name = rec.clusterName + "/" + t.Name
				updates[i].Value.(*model.ProfileRules).InboundRules = []model.Rule{}
				updates[i].Value.(*model.ProfileRules).OutboundRules = []model.Rule{}
				updates[i].Key = t
			case model.ProfileLabelsKey:
				t.Name = rec.clusterName + "/" + t.Name
				updates[i].Key = t
			case model.ProfileTagsKey:
				t.Name = rec.clusterName + "/" + t.Name
				updates[i].Key = t
			}
		} else if update.UpdateType == api.UpdateTypeKVDeleted {
			switch t := update.Key.(type) {
			default:
				log.Warnf("unexpected type %T\n", t)
			case model.HostEndpointKey:
				t.Hostname = rec.clusterName + "/" + t.Hostname
				updates[i].Key = t
			case model.WorkloadEndpointKey:
				t.Hostname = rec.clusterName + "/" + t.Hostname
				updates[i].Key = t
			case model.ProfileRulesKey:
				t.Name = rec.clusterName + "/" + t.Name
				updates[i].Key = t
			case model.ProfileLabelsKey:
				t.Name = rec.clusterName + "/" + t.Name
				updates[i].Key = t
			case model.ProfileTagsKey:
				t.Name = rec.clusterName + "/" + t.Name
				updates[i].Key = t
			}
		}
	}

	rec.wrappedCallbacks.OnUpdates(updates)
}
