// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calc

import (
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	sel "github.com/projectcalico/libcalico-go/lib/selector"
)

// ActiveEgressCalculator tracks and reference counts the egress selectors that are used by active
// local endpoints.  It generates an egress IP set ID for each unique egress selector.  It calls the
// IP set member index (SelectorAndNamedPortIndex) to get it to calculate the egress gateway pod IPs
// for each selector; and the PolicyResolver to tell it to include the egress IP set ID on the
// WorkloadEndpoint data that is passed to the dataplane implementation.
type ActiveEgressCalculator struct {
	// "EnabledPerNamespaceOrPerPod" or "EnabledPerNamespace".
	supportLevel string

	// Active egress selectors.
	selectors map[string]*esData

	// Active local endpoints.
	endpoints map[model.WorkloadEndpointKey]*epData

	// Known profile egress selectors.
	profiles map[string]string

	// Callbacks.
	OnIPSetActive         func(ipSet *IPSetData)
	OnIPSetInactive       func(ipSet *IPSetData)
	OnEgressIPSetIDUpdate func(key model.WorkloadEndpointKey, egressIPSetID string)
}

// Information that we track for each active local endpoint.
type epData struct {
	// The egress selector, if any, configured directly on this endpoint.
	localSelector string

	// The egress selector that this endpoint is now using - which could come from one of its
	// profiles.
	activeSelector string

	// This endpoint's profile IDs.
	profileIDs []string
}

// Information that we track for each active egress selector.
type esData struct {
	// Definition as IP set (including parsed selector).
	ipSet *IPSetData

	// Number of active local endpoints using this selector.
	refCount int
}

func NewActiveEgressCalculator(supportLevel string) *ActiveEgressCalculator {
	aec := &ActiveEgressCalculator{
		supportLevel: supportLevel,
		selectors:    map[string]*esData{},
		endpoints:    map[model.WorkloadEndpointKey]*epData{},
		profiles:     map[string]string{},
	}
	return aec
}

func (aec *ActiveEgressCalculator) RegisterWith(localEndpointDispatcher, allUpdDispatcher *dispatcher.Dispatcher) {
	// It needs local workload endpoints
	localEndpointDispatcher.Register(model.WorkloadEndpointKey{}, aec.OnUpdate)
	// ...and profiles.
	allUpdDispatcher.Register(model.ResourceKey{}, aec.OnUpdate)
}

func (aec *ActiveEgressCalculator) OnUpdate(update api.Update) (_ bool) {
	switch key := update.Key.(type) {
	case model.WorkloadEndpointKey:
		if update.Value != nil {
			log.Debugf("Updating AEC with endpoint %v", key)
			endpoint := update.Value.(*model.WorkloadEndpoint)
			if aec.supportLevel == "EnabledPerNamespaceOrPerPod" {
				// Endpoint-level selectors are supported.
				aec.updateEndpoint(key, endpoint.ProfileIDs, endpoint.EgressSelector)
			} else {
				// Endpoint-level selectors are not supported.
				aec.updateEndpoint(key, endpoint.ProfileIDs, "")
			}

		} else {
			log.Debugf("Deleting endpoint %v from AEC", key)
			aec.deleteEndpoint(key)
		}
	case model.ResourceKey:
		switch key.Kind {
		case v3.KindProfile:
			if update.Value != nil {
				log.Debugf("Updating AEC with profile %v", key)
				profile := update.Value.(*v3.Profile)
				aec.updateProfile(key.Name, profile.Spec.EgressGateway)
			} else {
				log.Debugf("Deleting profile %v from AEC", key)
				aec.updateProfile(key.Name, nil)
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

func (aec *ActiveEgressCalculator) updateProfile(profileID string, egress *v3.EgressSpec) {
	// Find the existing selector for this profile.
	oldSelector := aec.profiles[profileID]

	// Calculate the new selector
	newSelector := ""
	if egress != nil {
		// Convert egress Selector and NamespaceSelector fields to a single selector
		// expression in the same way we do for namespaced policy EntityRule selectors.
		newSelector = updateprocessors.GetEgressGatewaySelector(
			egress,
			strings.TrimPrefix(profileID, conversion.NamespaceProfileNamePrefix),
		)
	}

	// If the selector hasn't changed, no need to scan the endpoints.
	if newSelector == oldSelector {
		return
	}

	// Update profile selector map.
	if newSelector != "" {
		aec.profiles[profileID] = newSelector
	} else {
		delete(aec.profiles, profileID)
	}

	// Scan endpoints to find those that use this profile and don't specify their own egress
	// selector.  We follow SelectorAndNamedPortIndex here in using more CPU and less occupancy
	// - i.e. not maintaining a reverse map of profiles to endpoints - because profile changes
	// should be rare and we are only scanning through local endpoints, which scales only with
	// single node capacity, not with overall cluster size.
	for key, epData := range aec.endpoints {
		if epData.localSelector != "" {
			// Endpoint specifies its own egress selector, so profiles aren't relevant.
			continue
		}
		if epData.activeSelector == "" && newSelector == "" {
			// Endpoint has no egress selector, and this profile isn't providing one, so
			// can't possibly change the endpoint's situation.
			continue
		}

		// Spin through endpoint's profiles to find the first one, if any, that provides an
		// egress selector.
		oldEpSelector := epData.activeSelector
		epData.activeSelector = ""
		for _, profileID := range epData.profileIDs {
			if aec.profiles[profileID] != "" {
				epData.activeSelector = aec.profiles[profileID]
				break
			}
		}

		// Push selector change to IP set member index and policy resolver.
		aec.updateEndpointSelector(key, oldEpSelector, epData.activeSelector)
	}
}

func (aec *ActiveEgressCalculator) updateEndpointSelector(key model.WorkloadEndpointKey, old, new string) {
	if new == old {
		return
	}

	// Decref the old one and incref the new one.
	aec.decRefSelector(old)
	aec.incRefSelector(new)
	egressIPSetID := ""
	if new != "" {
		egressIPSetID = aec.selectors[new].ipSet.UniqueID()
	}
	aec.OnEgressIPSetIDUpdate(key, egressIPSetID)
}

func (aec *ActiveEgressCalculator) updateEndpoint(key model.WorkloadEndpointKey, profileIDs []string, endpointSelector string) {
	// Find or create the data for this endpoint.
	ep, exists := aec.endpoints[key]
	if !exists {
		ep = &epData{}
		aec.endpoints[key] = ep
	}

	// Note the existing active selector, which may be about to be overwritten.
	oldSelector := ep.activeSelector

	// Inherit an egress selector from the profiles, if the endpoint itself doesn't have one.
	ep.localSelector = endpointSelector
	ep.activeSelector = endpointSelector
	if ep.activeSelector == "" {
		for _, id := range profileIDs {
			if aec.profiles[id] != "" {
				ep.activeSelector = aec.profiles[id]
				break
			}
		}
	}
	ep.profileIDs = profileIDs

	// Push selector change to IP set member index and policy resolver.
	aec.updateEndpointSelector(key, oldSelector, ep.activeSelector)
}

func (aec *ActiveEgressCalculator) deleteEndpoint(key model.WorkloadEndpointKey) {
	// Find and delete the data for this endpoint.
	ep, exists := aec.endpoints[key]
	if !exists {
		return
	}
	delete(aec.endpoints, key)

	// Decref this endpoint's selector.
	aec.decRefSelector(ep.activeSelector)

	if ep.activeSelector != "" {
		// Ensure downstream components clear any egress IP set ID data for this endpoint
		// key.
		aec.OnEgressIPSetIDUpdate(key, "")
	}
}

func (aec *ActiveEgressCalculator) incRefSelector(selector string) {
	if selector == "" {
		return
	}
	selData, exists := aec.selectors[selector]
	if !exists {
		log.Debugf("Selector: %v", selector)
		sel, err := sel.Parse(selector)
		if err != nil {
			// Should have been validated further back in the pipeline.
			log.WithField("selector", selector).Panic(
				"Failed to parse egress selector that should have been validated already")
		}
		selData = &esData{ipSet: &IPSetData{
			Selector:         sel,
			IsEgressSelector: true,
		}}
		aec.selectors[selector] = selData
		aec.OnIPSetActive(selData.ipSet)
	}
	selData.refCount += 1
}

func (aec *ActiveEgressCalculator) decRefSelector(selector string) {
	if selector == "" {
		return
	}
	esData, exists := aec.selectors[selector]
	if !exists || esData.refCount <= 0 {
		log.Panicf("Decref for unknown egress selector '%v'", selector)
	}
	esData.refCount -= 1
	if esData.refCount == 0 {
		aec.OnIPSetInactive(esData.ipSet)
		delete(aec.selectors, selector)
	}
}
