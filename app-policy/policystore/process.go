// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package policystore

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

// ProcessUpdate -  Update the PolicyStore with the information passed over the Sync API.
func (store *PolicyStore) ProcessUpdate(subscriptionType string, update *proto.ToDataplane, storeStaged bool) {
	// TODO: maybe coalesce-ing updater fits here
	switch payload := update.Payload.(type) {
	case *proto.ToDataplane_InSync:
		store.processInSync(payload.InSync)
	case *proto.ToDataplane_IpsetUpdate:
		store.processIPSetUpdate(payload.IpsetUpdate)
	case *proto.ToDataplane_IpsetDeltaUpdate:
		store.processIPSetDeltaUpdate(payload.IpsetDeltaUpdate)
	case *proto.ToDataplane_IpsetRemove:
		store.processIPSetRemove(payload.IpsetRemove)
	case *proto.ToDataplane_ActiveProfileUpdate:
		store.processActiveProfileUpdate(payload.ActiveProfileUpdate)
	case *proto.ToDataplane_ActiveProfileRemove:
		store.processActiveProfileRemove(payload.ActiveProfileRemove)
	case *proto.ToDataplane_ActivePolicyUpdate:
		if !storeStaged && model.PolicyIsStaged(payload.ActivePolicyUpdate.Id.Name) {
			log.WithFields(log.Fields{
				"id": payload.ActivePolicyUpdate.Id,
			}).Debug("Skipping StagedPolicy ActivePolicyUpdate")

			return
		}

		store.processActivePolicyUpdate(payload.ActivePolicyUpdate)
	case *proto.ToDataplane_ActivePolicyRemove:
		if !storeStaged && model.PolicyIsStaged(payload.ActivePolicyRemove.Id.Name) {
			log.WithFields(log.Fields{
				"id": payload.ActivePolicyRemove.Id,
			}).Debug("Skipping StagedPolicy ActivePolicyRemove")

			return
		}

		store.processActivePolicyRemove(payload.ActivePolicyRemove)
	case *proto.ToDataplane_WorkloadEndpointUpdate:
		store.processWorkloadEndpointUpdate(subscriptionType, payload.WorkloadEndpointUpdate)
	case *proto.ToDataplane_WorkloadEndpointRemove:
		store.processWorkloadEndpointRemove(subscriptionType, payload.WorkloadEndpointRemove)
	case *proto.ToDataplane_ServiceAccountUpdate:
		store.processServiceAccountUpdate(payload.ServiceAccountUpdate)
	case *proto.ToDataplane_ServiceAccountRemove:
		store.processServiceAccountRemove(payload.ServiceAccountRemove)
	case *proto.ToDataplane_NamespaceUpdate:
		store.processNamespaceUpdate(payload.NamespaceUpdate)
	case *proto.ToDataplane_NamespaceRemove:
		store.processNamespaceRemove(payload.NamespaceRemove)
	case *proto.ToDataplane_ConfigUpdate:
		store.processConfigUpdate(payload.ConfigUpdate)
	default:
		log.Warn(fmt.Sprintf("unknown payload %v", update.String()))
	}
}

func (store *PolicyStore) processInSync(inSync *proto.InSync) {
	log.Debug("Processing InSync")
}

func (store *PolicyStore) processConfigUpdate(update *proto.ConfigUpdate) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"config": update.Config,
		}).Debug("Processing ConfigUpdate")
	}

	// Update the DropActionOverride setting if it is available.
	if val, ok := update.Config["DropActionOverride"]; ok {
		log.Debug("DropActionOverride is present in config")
		var psVal DropActionOverride
		switch strings.ToLower(val) {
		case "drop":
			psVal = DROP
		case "accept":
			psVal = ACCEPT
		case "loganddrop":
			psVal = LOG_AND_DROP
		case "logandaccept":
			psVal = LOG_AND_ACCEPT
		default:
			log.Errorf("Unknown DropActionOverride value: %s", val)
			psVal = DROP
		}
		store.DropActionOverride = psVal
	}

	// Extract the flow logs settings, defaulting to false if not present.
	store.DataplaneStatsEnabledForAllowed = getBoolFromConfig(update.Config, "DataplaneStatsEnabledForAllowed", false)
	store.DataplaneStatsEnabledForDenied = getBoolFromConfig(update.Config, "DataplaneStatsEnabledForDenied", false)
}

func (store *PolicyStore) processIPSetUpdate(update *proto.IPSetUpdate) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"id":      update.Id,
			"type":    update.Type.String(),
			"members": update.Members,
		}).Debug("Processing IPSetUpdate")
	}

	// IPSetUpdate replaces the existing set.
	s := NewIPSet(update.Type)
	for _, addr := range update.Members {
		s.AddString(addr)
	}
	store.IPSetByID[update.Id] = s
}

func (store *PolicyStore) processIPSetDeltaUpdate(update *proto.IPSetDeltaUpdate) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"id":      update.Id,
			"added":   update.AddedMembers,
			"removed": update.RemovedMembers,
		}).Debug("Processing IPSetDeltaUpdate")
	}
	s, ok := store.IPSetByID[update.Id]
	if !ok {
		log.Errorf("Unknown IPSet id: %v, skipping update", update.Id)
		return // we shouldn't be getting a delta update before we've seen the IPSet
	}
	for _, addr := range update.AddedMembers {
		s.AddString(addr)
	}
	for _, addr := range update.RemovedMembers {
		s.RemoveString(addr)
	}
}

func (store *PolicyStore) processIPSetRemove(update *proto.IPSetRemove) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"id": update.Id,
		}).Debug("Processing IPSetRemove")
	}
	delete(store.IPSetByID, update.Id)
}

func (store *PolicyStore) processActiveProfileUpdate(update *proto.ActiveProfileUpdate) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"id": update.Id,
		}).Debug("Processing ActiveProfileUpdate")
	}
	if update.Id == nil {
		log.Error("got ActiveProfileUpdate with nil ProfileID")
		return
	}
	store.ProfileByID[*update.Id] = update.Profile
}

func (store *PolicyStore) processActiveProfileRemove(update *proto.ActiveProfileRemove) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"id": update.Id,
		}).Debug("Processing ActiveProfileRemove")
	}
	if update.Id == nil {
		log.Error("got ActiveProfileRemove with nil ProfileID")
		return
	}
	delete(store.ProfileByID, *update.Id)
}

func (store *PolicyStore) processActivePolicyUpdate(update *proto.ActivePolicyUpdate) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"id": update.Id,
		}).Debug("Processing ActivePolicyUpdate")
	}
	if update.Id == nil {
		log.Error("got ActivePolicyUpdate with nil PolicyID")
		return
	}
	store.PolicyByID[*update.Id] = update.Policy
}

func (store *PolicyStore) processActivePolicyRemove(update *proto.ActivePolicyRemove) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"id": update.Id,
		}).Debug("Processing ActivePolicyRemove")
	}
	if update.Id == nil {
		log.Error("got ActivePolicyRemove with nil PolicyID")
		return
	}
	delete(store.PolicyByID, *update.Id)
}

func (store *PolicyStore) processWorkloadEndpointUpdate(subscriptionType string, update *proto.WorkloadEndpointUpdate) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"orchestratorID": update.GetId().GetOrchestratorId(),
			"workloadID":     update.GetId().GetWorkloadId(),
			"endpointID":     update.GetId().GetEndpointId(),
		}).Debug("Processing WorkloadEndpointUpdate")
	}
	switch subscriptionType {
	case "per-pod-policies", "":
		store.Endpoint = update.Endpoint
	case "per-host-policies":
		store.Endpoints[*update.Id] = update.Endpoint
		log.Debugf("%d endpoints received so far", len(store.Endpoints))
		store.wepUpdates.onWorkloadEndpointUpdate(update, store.IPToIndexes)
	}
}

func (store *PolicyStore) processWorkloadEndpointRemove(subscriptionType string, update *proto.WorkloadEndpointRemove) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"orchestratorID": update.GetId().GetOrchestratorId(),
			"workloadID":     update.GetId().GetWorkloadId(),
			"endpointID":     update.GetId().GetEndpointId(),
		}).Debug("Processing WorkloadEndpointRemove")
	}

	switch subscriptionType {
	case "per-pod-policies", "":
		store.Endpoint = nil
	case "per-host-policies":
		delete(store.Endpoints, *update.Id)
		store.wepUpdates.onWorkloadEndpointRemove(update, store.IPToIndexes)
	}
}

func (store *PolicyStore) processServiceAccountUpdate(update *proto.ServiceAccountUpdate) {
	log.WithField("id", update.Id).Debug("Processing ServiceAccountUpdate")
	if update.Id == nil {
		log.Error("got ServiceAccountUpdate with nil ServiceAccountID")
		return
	}
	store.ServiceAccountByID[*update.Id] = update
}

func (store *PolicyStore) processServiceAccountRemove(update *proto.ServiceAccountRemove) {
	log.WithField("id", update.Id).Debug("Processing ServiceAccountRemove")
	if update.Id == nil {
		log.Error("got ServiceAccountRemove with nil ServiceAccountID")
		return
	}
	delete(store.ServiceAccountByID, *update.Id)
}

func (store *PolicyStore) processNamespaceUpdate(update *proto.NamespaceUpdate) {
	log.WithField("id", update.Id).Debug("Processing NamespaceUpdate")
	if update.Id == nil {
		log.Error("got NamespaceUpdate with nil NamespaceID")
		return
	}
	store.NamespaceByID[*update.Id] = update
}

func (store *PolicyStore) processNamespaceRemove(update *proto.NamespaceRemove) {
	log.WithField("id", update.Id).Debug("Processing NamespaceRemove")
	if update.Id == nil {
		log.Error("got NamespaceRemove with nil NamespaceID")
		return
	}
	delete(store.NamespaceByID, *update.Id)
}
