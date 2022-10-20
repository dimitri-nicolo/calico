// Copyright (c) 2016-2020 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package calc

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/dispatcher"
	"github.com/projectcalico/calico/felix/multidict"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

var (
	gaugeNumActiveEndpoints = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_active_local_endpoints",
		Help: "Number of active endpoints on this host.",
	})
	gaugeNumActivePolicies = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_active_local_policies",
		Help: "Number of active policies on this host.",
	})
)

func init() {
	prometheus.MustRegister(gaugeNumActiveEndpoints)
	prometheus.MustRegister(gaugeNumActivePolicies)
}

// PolicyResolver marries up the active policies with local endpoints and
// calculates the complete, ordered set of policies that apply to each endpoint.
// As policies and endpoints are added/removed/updated, it emits events
// via the PolicyResolverCallbacks with the updated set of matching policies.
//
// The PolicyResolver doesn't figure out which policies are currently active, it
// expects to be told via its OnPolicyMatch(Stopped) methods which policies match
// which endpoints.  The ActiveRulesCalculator does that calculation.
type PolicyResolver struct {
	policyIDToEndpointIDs multidict.IfaceToIface
	endpointIDToPolicyIDs multidict.IfaceToIface
	sortedTierData        []*tierInfo
	endpoints             map[model.Key]interface{}
	endpointEgressData    map[model.WorkloadEndpointKey]epEgressData
	endpointGatewayUsage  map[model.WorkloadEndpointKey]int
	dirtyEndpoints        set.Set[any] /* FIXME model.WorkloadEndpointKey or model.HostEndpointKey */
	sortRequired          bool
	policySorter          *PolicySorter
	Callbacks             []PolicyResolverCallbacks
	InSync                bool
}

type PolicyResolverCallbacks interface {
	OnEndpointTierUpdate(endpointKey model.Key, endpoint interface{}, egressData EndpointEgressData, filteredTiers []tierInfo)
}

func NewPolicyResolver() *PolicyResolver {
	return &PolicyResolver{
		policyIDToEndpointIDs: multidict.NewIfaceToIface(),
		endpointIDToPolicyIDs: multidict.NewIfaceToIface(),
		endpoints:             make(map[model.Key]interface{}),
		endpointEgressData:    make(map[model.WorkloadEndpointKey]epEgressData),
		endpointGatewayUsage:  make(map[model.WorkloadEndpointKey]int),
		dirtyEndpoints:        set.NewBoxed[any](),
		policySorter:          NewPolicySorter(),
		Callbacks:             []PolicyResolverCallbacks{},
	}
}

func (pr *PolicyResolver) RegisterWith(allUpdDispatcher, localEndpointDispatcher, tierDispatcher *dispatcher.Dispatcher) {
	tierDispatcher.Register(model.PolicyKey{}, pr.OnUpdate)
	tierDispatcher.Register(model.TierKey{}, pr.OnUpdate)
	localEndpointDispatcher.Register(model.WorkloadEndpointKey{}, pr.OnUpdate)
	localEndpointDispatcher.Register(model.HostEndpointKey{}, pr.OnUpdate)
	localEndpointDispatcher.RegisterStatusHandler(pr.OnDatamodelStatus)
}

func (pr *PolicyResolver) RegisterCallback(cb PolicyResolverCallbacks) {
	pr.Callbacks = append(pr.Callbacks, cb)
}

func (pr *PolicyResolver) OnUpdate(update api.Update) (filterOut bool) {
	policiesDirty := false
	switch key := update.Key.(type) {
	case model.WorkloadEndpointKey, model.HostEndpointKey:
		if update.Value != nil {
			pr.endpoints[key] = update.Value
		} else {
			delete(pr.endpoints, key)
			if wlKey, ok := key.(model.WorkloadEndpointKey); ok {
				delete(pr.endpointEgressData, wlKey)
			}
		}
		pr.dirtyEndpoints.Add(key)
		gaugeNumActiveEndpoints.Set(float64(len(pr.endpoints)))
	case model.PolicyKey:
		log.Debugf("Policy update: %v", key)
		policiesDirty = pr.policySorter.OnUpdate(update)
		if policiesDirty {
			pr.markEndpointsMatchingPolicyDirty(key)
		}
	case model.TierKey:
		log.Debugf("Tier update: %v", key)
		policiesDirty = pr.policySorter.OnUpdate(update)
		pr.markAllEndpointsDirty()
	}
	pr.sortRequired = pr.sortRequired || policiesDirty
	pr.maybeFlush()
	gaugeNumActivePolicies.Set(float64(pr.policyIDToEndpointIDs.Len()))
	return
}

func (pr *PolicyResolver) OnDatamodelStatus(status api.SyncStatus) {
	if status == api.InSync {
		pr.InSync = true
		pr.maybeFlush()
	}
}

func (pr *PolicyResolver) refreshSortOrder() {
	pr.sortedTierData = pr.policySorter.Sorted()
	pr.sortRequired = false
	log.Debugf("New sort order: %v", pr.sortedTierData)
}

func (pr *PolicyResolver) markAllEndpointsDirty() {
	log.Debugf("Marking all endpoints dirty")
	pr.endpointIDToPolicyIDs.IterKeys(func(epID interface{}) {
		pr.dirtyEndpoints.Add(epID)
	})
}

func (pr *PolicyResolver) markEndpointsMatchingPolicyDirty(polKey model.PolicyKey) {
	log.Debugf("Marking all endpoints matching %v dirty", polKey)
	pr.policyIDToEndpointIDs.Iter(polKey, func(epID interface{}) {
		pr.dirtyEndpoints.Add(epID)
	})
}

func (pr *PolicyResolver) OnPolicyMatch(policyKey model.PolicyKey, endpointKey interface{}) {
	log.Debugf("Storing policy match %v -> %v", policyKey, endpointKey)
	pr.policyIDToEndpointIDs.Put(policyKey, endpointKey)
	pr.endpointIDToPolicyIDs.Put(endpointKey, policyKey)
	pr.dirtyEndpoints.Add(endpointKey)
	pr.maybeFlush()
}

func (pr *PolicyResolver) OnPolicyMatchStopped(policyKey model.PolicyKey, endpointKey interface{}) {
	log.Debugf("Deleting policy match %v -> %v", policyKey, endpointKey)
	pr.policyIDToEndpointIDs.Discard(policyKey, endpointKey)
	pr.endpointIDToPolicyIDs.Discard(endpointKey, policyKey)
	pr.dirtyEndpoints.Add(endpointKey)
	pr.maybeFlush()
}

func (pr *PolicyResolver) OnEgressSelectorMatch(es string, endpointKey interface{}) {
	if key, ok := endpointKey.(model.WorkloadEndpointKey); ok {
		log.Debugf("Egress selector match %v -> %v", es, key)
		pr.endpointGatewayUsage[key]++
		pr.dirtyEndpoints.Add(endpointKey)
		pr.maybeFlush()
	}
}

func (pr *PolicyResolver) OnEgressSelectorMatchStopped(es string, endpointKey interface{}) {
	if key, ok := endpointKey.(model.WorkloadEndpointKey); ok {
		log.Debugf("Delete egress selector match %v -> %v", es, key)
		pr.endpointGatewayUsage[key]--
		if pr.endpointGatewayUsage[key] == 0 {
			delete(pr.endpointGatewayUsage, key)
		}
		pr.dirtyEndpoints.Add(endpointKey)
		pr.maybeFlush()
	}
}

func (pr *PolicyResolver) maybeFlush() {
	if !pr.InSync {
		log.Debugf("Not in sync, skipping flush")
		return
	}
	if pr.sortRequired {
		pr.refreshSortOrder()
	}
	pr.dirtyEndpoints.Iter(pr.sendEndpointUpdate)
	pr.dirtyEndpoints = set.NewBoxed[any]()
}

func (pr *PolicyResolver) sendEndpointUpdate(endpointID interface{}) error {
	log.Debugf("Sending tier update for endpoint %v", endpointID)
	endpoint, ok := pr.endpoints[endpointID.(model.Key)]
	if !ok {
		log.Debugf("Endpoint is unknown, sending nil update")
		for _, cb := range pr.Callbacks {
			cb.OnEndpointTierUpdate(endpointID.(model.Key),
				nil, EndpointEgressData{}, []tierInfo{})
		}
		return nil
	}

	applicableTiers := []tierInfo{}
	for _, tier := range pr.sortedTierData {
		if !tier.Valid {
			log.Debugf("Tier %v invalid, skipping", tier.Name)
		}
		tierMatches := false
		filteredTier := tierInfo{
			Name:  tier.Name,
			Order: tier.Order,
			Valid: true,
		}
		for _, polKV := range tier.OrderedPolicies {
			log.Debugf("Checking if policy %v matches %v", polKV.Key, endpointID)
			if pr.endpointIDToPolicyIDs.Contains(endpointID, polKV.Key) {
				log.Debugf("Policy %v matches %v", polKV.Key, endpointID)
				tierMatches = true
				filteredTier.OrderedPolicies = append(filteredTier.OrderedPolicies,
					polKV)
			}
		}
		if tierMatches {
			log.Debugf("Tier %v matches %v", tier.Name, endpointID)
			applicableTiers = append(applicableTiers, filteredTier)
		}
	}

	egressData := EndpointEgressData{}
	if key, ok := endpointID.(model.WorkloadEndpointKey); ok {
		egressData.EgressIPSetID = pr.endpointEgressData[key].ipSetID
		egressData.MaxNextHops = pr.endpointEgressData[key].maxNextHops
		egressData.IsEgressGateway = pr.endpointGatewayUsage[key] > 0
		egressData.HealthPort = findHealthPort(endpoint.(*model.WorkloadEndpoint))
	}

	log.Debugf("Endpoint tier update: %v -> %v", endpointID, applicableTiers)
	for _, cb := range pr.Callbacks {
		cb.OnEndpointTierUpdate(endpointID.(model.Key),
			endpoint, egressData, applicableTiers)
	}
	return nil
}

func findHealthPort(endpoint *model.WorkloadEndpoint) uint16 {
	for _, p := range endpoint.Ports {
		if p.Name == "health" {
			return p.Port
		}
	}
	return 0
}

func (pr *PolicyResolver) OnEndpointEgressDataUpdate(key model.WorkloadEndpointKey, egressData epEgressData) {
	if egressData.ipSetID != "" {
		pr.endpointEgressData[key] = egressData
	} else {
		delete(pr.endpointEgressData, key)
	}
	pr.dirtyEndpoints.Add(key)
	pr.maybeFlush()
}
