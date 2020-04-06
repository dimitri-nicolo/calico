// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calc

import (
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/dispatcher"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	sel "github.com/projectcalico/libcalico-go/lib/selector"
	"github.com/projectcalico/libcalico-go/lib/set"
)

// ActiveEgressCalculator tracks and reference counts the egress selectors that are used by active
// local endpoints.  It generates an egress IP set ID for each unique egress selector.  It calls the
// IP set member index (SelectorAndNamedPortIndex) to get it to calculate the egress gateway pod IPs
// for each selector; and the PolicyResolver to tell it to include the egress IP set ID on the
// WorkloadEndpoint data that is passed to the dataplane implementation.
type ActiveEgressCalculator struct {
	// Active egress selectors.
	selectors map[string]*esData

	// Active local endpoints.
	endpoints map[model.WorkloadEndpointKey]*epData

	// Known or needed profiles.
	profiles map[string]*prData

	// Callbacks.
	OnIPSetActive         func(ipSet *IPSetData)
	OnIPSetInactive       func(ipSet *IPSetData)
	OnEgressIPSetIDUpdate func(key model.WorkloadEndpointKey, egressIPSetID string)
}

// Information that we track for each profile.
type prData struct {
	// The egress selector for this profile.
	egressSelector string

	// References: 1 for the defining Profile resource and 1 for each active local endpoints
	// that uses that profile.
	profileExists bool
	endpointCount int
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

func NewActiveEgressCalculator() *ActiveEgressCalculator {
	aec := &ActiveEgressCalculator{
		selectors: map[string]*esData{},
		endpoints: map[model.WorkloadEndpointKey]*epData{},
		profiles:  map[string]*prData{},
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
			aec.updateEndpoint(key, endpoint.ProfileIDs, endpoint.EgressSelector, update.UpdateType)
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
				aec.updateProfile(key.Name, true, profile.Spec.EgressGateway)
			} else {
				log.Debugf("'Deleting' profile %v from AEC", key)
				aec.updateProfile(key.Name, false, nil)
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

func (aec *ActiveEgressCalculator) updateProfile(profileID string, profileExists bool, egress *v3.EgressSpec) {
	// Find or create the data for this profile.
	profile, exists := aec.profiles[profileID]
	if !exists {
		profile = &prData{}
		aec.profiles[profileID] = profile
	}
	if profile.endpointCount == 0 && !profileExists {
		delete(aec.profiles, profileID)
		return
	}
	profile.profileExists = profileExists

	// Note the existing selector, which may be about to be overwritten.
	oldSelector := profile.egressSelector

	// Calculate the new selector
	if egress == nil {
		// No egress control.
		profile.egressSelector = ""
	} else {
		// Convert egress Selector and NamespaceSelector fields to a single selector
		// expression in the same way we do for namespaced policy EntityRule selectors.
		profile.egressSelector = updateprocessors.GetEgressGatewaySelector(
			egress,
			strings.TrimPrefix(profileID, conversion.NamespaceProfileNamePrefix),
		)
	}

	// If the selector hasn't changed, no need to scan the endpoints.
	if profile.egressSelector == oldSelector {
		return
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
		if epData.activeSelector != "" && epData.activeSelector != oldSelector {
			// Endpoint has an egress selector that didn't come from this profile.  No
			// need to change it.  Note: undefined behaviour alert if an endpoint has
			// multiple profiles that specify egress selectors; but we don't expect or
			// support that case.
			continue
		}
		if epData.activeSelector == "" && profile.egressSelector == "" {
			// Endpoint has no egress selector, and this profile isn't providing one, so
			// can't possibly change the endpoint's situation.
			continue
		}

		// The remaining possibilities require checking if the endpoint uses the profile
		// that is changing.
		endpointUsesProfile := func() bool {
			for _, id := range epData.profileIDs {
				if id == profileID {
					return true
				}
			}
			return false
		}
		if !endpointUsesProfile() {
			// Endpoint doesn't use this profile.
			continue
		}

		// Endpoint uses this profile and may be affected by the profile's egress selector
		// changing.
		oldEpSelector := epData.activeSelector
		if epData.activeSelector == "" && profile.egressSelector != "" {
			// Endpoint did not have an egress selector, and this profile now provides
			// one, so make it the endpoint's active selector.
			epData.activeSelector = profile.egressSelector
		} else {
			// Spin through endpoint's profiles to find the first one, if any, that
			// provides an egress selector.
			epData.activeSelector = ""
			for _, profileID := range epData.profileIDs {
				profile := aec.profiles[profileID]
				if profile.egressSelector != "" {
					epData.activeSelector = profile.egressSelector
					break
				}
			}
		}

		// Push selector change to IP set member index and policy resolver.
		aec.updateEndpointSelector(key, oldEpSelector, epData.activeSelector, false)
	}
}

func (aec *ActiveEgressCalculator) updateEndpointSelector(key model.WorkloadEndpointKey, old, new string, newEndpoint bool) {
	if new == old && !newEndpoint {
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

func (aec *ActiveEgressCalculator) updateEndpoint(key model.WorkloadEndpointKey, profileIDs []string, endpointSelector string, updateType api.UpdateType) {
	// Find or create the data for this endpoint.
	ep, exists := aec.endpoints[key]
	if !exists {
		ep = &epData{}
		aec.endpoints[key] = ep
	}

	// Note the existing (net) selector, which may be about to be overwritten.
	oldSelector := ep.activeSelector

	// Update and create profile data for this endpoint.  Also inherit an egress selector from
	// the profiles, if the endpoint itself doesn't have one.  It's undefined behaviour if more
	// than one profile specifies an egress selector; we normally expect only one profile to do
	// this, the one corresponding to the namespace.
	oldProfileIDs := set.FromArray(ep.profileIDs)
	ep.localSelector = endpointSelector
	ep.activeSelector = endpointSelector
	for _, id := range profileIDs {
		if oldProfileIDs.Contains(id) {
			// Endpoint was already using this profile.
			oldProfileIDs.Discard(id)
		} else {
			// Find or create the data for this profile.
			if _, exists := aec.profiles[id]; !exists {
				aec.profiles[id] = &prData{}
			}
			aec.profiles[id].endpointCount += 1
		}
		if ep.activeSelector == "" {
			ep.activeSelector = aec.profiles[id].egressSelector
		}
	}
	ep.profileIDs = profileIDs

	// Reduce reference count for profiles that the endpoint is no longer using.
	oldProfileIDs.Iter(aec.decEndpointCount)

	// Push selector change to IP set member index and policy resolver.
	aec.updateEndpointSelector(key, oldSelector, ep.activeSelector, updateType == api.UpdateTypeKVNew)
}

func (aec *ActiveEgressCalculator) decEndpointCount(item interface{}) error {
	id := item.(string)
	aec.profiles[id].endpointCount -= 1
	if aec.profiles[id].endpointCount == 0 && !aec.profiles[id].profileExists {
		delete(aec.profiles, id)
	}
	return nil
}

func (aec *ActiveEgressCalculator) deleteEndpoint(key model.WorkloadEndpointKey) {
	// Find and delete the data for this endpoint.
	ep, exists := aec.endpoints[key]
	if !exists {
		return
	}
	delete(aec.endpoints, key)

	// Reduce reference count for profiles that this endpoint was using.
	set.FromArray(ep.profileIDs).Iter(aec.decEndpointCount)

	// Decref this endpoint's selector.
	aec.decRefSelector(ep.activeSelector)
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
