// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calc

import (
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/dispatcher"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
)

// EgressSelectorPool tracks and reference counts the egress selectors that are used by profiles and
// endpoints anywhere in the cluster.  We need this to identify active local endpoints that match
// egress selectors, and hence should be privileged in the ways that are needed for endpoints acting
// as egress gateways.
type EgressSelectorPool struct {
	// "EnabledPerNamespaceOrPerPod" or "EnabledPerNamespace".
	supportLevel string

	// Known egress selectors and their ref counts.
	endpointSelectors map[model.WorkloadEndpointKey]string
	profileSelectors  map[model.ResourceKey]string
	selectorRefCount  map[string]int

	// Callbacks.
	OnEgressSelectorAdded   func(selector string)
	OnEgressSelectorRemoved func(selector string)
}

func NewEgressSelectorPool(supportLevel string) *EgressSelectorPool {
	esp := &EgressSelectorPool{
		supportLevel:      supportLevel,
		endpointSelectors: map[model.WorkloadEndpointKey]string{},
		profileSelectors:  map[model.ResourceKey]string{},
		selectorRefCount:  map[string]int{},
	}
	return esp
}

func (esp *EgressSelectorPool) RegisterWith(allUpdDispatcher *dispatcher.Dispatcher) {
	// Subject to support level, it needs all workload endpoints and v3 profiles.
	if esp.supportLevel == "EnabledPerNamespaceOrPerPod" {
		allUpdDispatcher.Register(model.WorkloadEndpointKey{}, esp.OnUpdate)
	}
	allUpdDispatcher.Register(model.ResourceKey{}, esp.OnUpdate)
}

func (esp *EgressSelectorPool) OnUpdate(update api.Update) (_ bool) {
	switch key := update.Key.(type) {
	case model.WorkloadEndpointKey:
		if update.Value != nil {
			log.Debugf("Updating ESP with endpoint %v", key)
			endpoint := update.Value.(*model.WorkloadEndpoint)
			esp.updateEndpoint(key, endpoint.EgressSelector)
		} else {
			log.Debugf("Deleting endpoint %v from ESP", key)
			esp.updateEndpoint(key, "")
		}
	case model.ResourceKey:
		switch key.Kind {
		case v3.KindProfile:
			if update.Value != nil {
				log.Debugf("Updating ESP with profile %v", key)
				profile := update.Value.(*v3.Profile)
				esp.updateProfile(key, profile.Spec.EgressGateway)
			} else {
				log.Debugf("Deleting profile %v from ESP", key)
				esp.updateProfile(key, nil)
			}
		default:
			// Ignore other kinds of v3 resource.
		}
	default:
		log.Infof("Ignoring unexpected update: %v %#v",
			reflect.TypeOf(update.Key), update)
	}

	return
}

func (esp *EgressSelectorPool) updateEndpoint(key model.WorkloadEndpointKey, newSelector string) {
	oldSelector := esp.endpointSelectors[key]
	if newSelector == oldSelector {
		// No change.
		return
	}
	if newSelector != "" {
		esp.endpointSelectors[key] = newSelector
	} else {
		delete(esp.endpointSelectors, key)
	}
	esp.decRefSelector(oldSelector)
	esp.incRefSelector(newSelector)
}

func (esp *EgressSelectorPool) updateProfile(key model.ResourceKey, egress *v3.EgressSpec) {
	// Find the existing selector for this profile.
	oldSelector := esp.profileSelectors[key]

	// Calculate the new selector
	newSelector := ""
	if egress != nil {
		// Convert egress Selector and NamespaceSelector fields to a single selector
		// expression in the same way we do for namespaced policy EntityRule selectors.
		newSelector = updateprocessors.GetEgressGatewaySelector(
			egress,
			strings.TrimPrefix(key.Name, conversion.NamespaceProfileNamePrefix),
		)
	}

	if newSelector == oldSelector {
		// No change.
		return
	}
	if newSelector != "" {
		esp.profileSelectors[key] = newSelector
	} else {
		delete(esp.profileSelectors, key)
	}
	esp.decRefSelector(oldSelector)
	esp.incRefSelector(newSelector)
}

func (esp *EgressSelectorPool) incRefSelector(selector string) {
	if selector == "" {
		return
	}
	esp.selectorRefCount[selector]++
	if esp.selectorRefCount[selector] == 1 {
		esp.OnEgressSelectorAdded(selector)
	}
}

func (esp *EgressSelectorPool) decRefSelector(selector string) {
	if selector == "" {
		return
	}
	esp.selectorRefCount[selector]--
	if esp.selectorRefCount[selector] == 0 {
		esp.OnEgressSelectorRemoved(selector)
		delete(esp.selectorRefCount, selector)
	}
}
