// Copyright (c) 2016 Tigera, Inc. All rights reserved.
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

package lookup

import (
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

// TODO (Matt): WorkloadEndpoints are only local; so we can't get nice information for the remote ends.
type LookupManager struct {
	// `string`s are IP.String().
	endpoints        map[string]*model.WorkloadEndpointKey
	endpointsReverse map[model.WorkloadEndpointKey]*string
	endpointTiers    map[model.WorkloadEndpointKey]*[]proto.TierInfo
	mutex            sync.Mutex
}

func NewLookupManager() *LookupManager {
	return &LookupManager{
		endpoints:        map[string]*model.WorkloadEndpointKey{},
		endpointsReverse: map[model.WorkloadEndpointKey]*string,
		endpointTiers:    map[model.WorkloadEndpointKey]*[]proto.TierInfo,
		mutex:            sync.Mutex{},
	}
}

func (m *LookupManager) OnUpdate(protoBufMsg interface{}) {
	switch msg := protoBufMsg.(type) {
	case *proto.WorkloadEndpointUpdate:
		// TODO (Matt): Need to lookup hostname.
		wlEpKey := model.WorkloadEndpointKey{
			Hostname:       "matt-k8s",
			OrchestratorID: msg.Id.OrchestratorId,
			WorkloadID:     msg.Id.WorkloadId,
			EndpointID:     msg.Id.EndpointId,
		}
		m.mutex.Lock()
		// Store tiers and policies
		m.endpointTiers[wlEpKey] = msg.Endpoint.Tiers
		// Store IP addresses
		for _, ipv4 := range msg.Endpoint.Ipv4Nets {
			addr, _, err := net.ParseCIDR(ipv4)
			if err != nil {
				log.Warn("Error parsing CIDR ", ipv4)
				continue
			}
			log.Debug("Stored IPv4 endpoint: ", wlEpKey, ": ", addr.String())
			m.endpoints[addr.String()] = &wlEpKey
			m.EndpointsReverse[wlEpKey] = &addr.String()
		}
		for _, ipv6 := range msg.Endpoint.Ipv6Nets {
			addr, _, err := net.ParseCIDR(ipv6)
			if err != nil {
				log.Warn("Error parsing CIDR ", ipv6)
				continue
			}
			// TODO (Matt): IP.String() does funny things to IPv6 mapped IPv4 addresses.
			log.Debug("Stored IPv6 endpoint: ", wlEpKey, ": ", addr.String())
			m.endpoints[addr.String()] = &wlEpKey
			m.EndpointsReverse[wlEpKey] = &addr.String()
		}
		m.mutex.Unlock()
	case *proto.WorkloadEndpointRemove:
		wlEpKey := model.WorkloadEndpointKey{
			Hostname:       "matt-k8s",
			OrchestratorID: msg.Id.OrchestratorId,
			WorkloadID:     msg.Id.WorkloadId,
			EndpointID:     msg.Id.EndpointId,
		}
		m.mutex.Lock()
		epIp := m.endpointsReverse[wlEpKey]
		if epIp != nil {
			delete(m.endpoints, epIp)
			delete(m.endpointsReverse, wlEpKey)
			delete(m.endpointTiers, wlEpKey)
		}
		m.mutex.Unlock()
	case *proto.HostEndpointUpdate:
		// TODO(Matt) Host endpoint updates
		log.WithField("msg", msg).Warn("Message not implemented")
	case *proto.HostEndpointRemove:
		// TODO(Matt) Host endpoint updates
		log.WithField("msg", msg).Warn("Message not implemented")
	}
}

func (m *LookupManager) CompleteDeferredWork() error {
	return nil
}

func (m *LookupManager) GetEndpointKey(addr net.IP) *model.WorkloadEndpointKey {
	m.mutex.Lock()
	// There's no need to copy the result because we never modify fields,
	// only delete or replace.
	epKey := m.endpoints[addr.String()]
	m.mutex.Unlock()
	return epKey
}

func (m *LookupManager) GetPolicyIndex(epKey *model.WorkloadEndpointKey, policyKey *model.PolicyKey) int {
	// TODO (Matt): Need to handle profiles as well as tiered policy.
	ti := m.endpointTiers[*epKey]
	//if profile return fold(sum, len(Policies) in TierInfos in ep.Tiers)
	//else walk tiers until policyKey.Tier, summing len(Policies) then + index(policy in tier)
	return 3
}
