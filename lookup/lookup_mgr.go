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
	"errors"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var UnknownEndpointError = errors.New("Unknown endpoint")

// TODO (Matt): WorkloadEndpoints are only local; so we can't get nice information for the remote ends.
type LookupManager struct {
	// `string`s are IP.String().
	endpoints        map[string]*model.WorkloadEndpointKey
	endpointsReverse map[model.WorkloadEndpointKey]*string
	endpointTiers    map[model.WorkloadEndpointKey][]*proto.TierInfo
	epMutex          sync.Mutex

	hostEndpoints              map[string]*model.HostEndpointKey
	hostEndpointsReverse       map[model.HostEndpointKey]*string
	hostEndpointTiers          map[model.HostEndpointKey][]*proto.TierInfo
	hostEndpointUntrackedTiers map[model.HostEndpointKey][]*proto.TierInfo
	hostEpMutex                sync.Mutex
}

func NewLookupManager() *LookupManager {
	return &LookupManager{
		endpoints:                  map[string]*model.WorkloadEndpointKey{},
		endpointsReverse:           map[model.WorkloadEndpointKey]*string{},
		endpointTiers:              map[model.WorkloadEndpointKey][]*proto.TierInfo{},
		hostEndpoints:              map[string]*model.HostEndpointKey{},
		hostEndpointsReverse:       map[model.HostEndpointKey]*string{},
		hostEndpointTiers:          map[model.HostEndpointKey][]*proto.TierInfo{},
		hostEndpointUntrackedTiers: map[model.HostEndpointKey][]*proto.TierInfo{},
		epMutex:                    sync.Mutex{},
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
		m.epMutex.Lock()
		// Store tiers and policies
		m.endpointTiers[wlEpKey] = msg.Endpoint.Tiers
		// Store IP addresses
		for _, ipv4 := range msg.Endpoint.Ipv4Nets {
			addr, _, err := net.ParseCIDR(ipv4)
			if err != nil {
				log.Warn("Error parsing CIDR ", ipv4)
				continue
			}
			addrStr := addr.String()
			log.Debug("Stored IPv4 endpoint: ", wlEpKey, ": ", addrStr)
			m.endpoints[addrStr] = &wlEpKey
			m.endpointsReverse[wlEpKey] = &addrStr
		}
		for _, ipv6 := range msg.Endpoint.Ipv6Nets {
			addr, _, err := net.ParseCIDR(ipv6)
			if err != nil {
				log.Warn("Error parsing CIDR ", ipv6)
				continue
			}
			// TODO (Matt): IP.String() does funny things to IPv6 mapped IPv4 addresses.
			addrStr := addr.String()
			log.Debug("Stored IPv6 endpoint: ", wlEpKey, ": ", addrStr)
			m.endpoints[addrStr] = &wlEpKey
			m.endpointsReverse[wlEpKey] = &addrStr
		}
		m.epMutex.Unlock()
	case *proto.WorkloadEndpointRemove:
		wlEpKey := model.WorkloadEndpointKey{
			Hostname:       "matt-k8s",
			OrchestratorID: msg.Id.OrchestratorId,
			WorkloadID:     msg.Id.WorkloadId,
			EndpointID:     msg.Id.EndpointId,
		}
		m.epMutex.Lock()
		epIp := m.endpointsReverse[wlEpKey]
		if epIp != nil {
			delete(m.endpoints, *epIp)
			delete(m.endpointsReverse, wlEpKey)
			delete(m.endpointTiers, wlEpKey)
		}
		m.epMutex.Unlock()
	case *proto.HostEndpointUpdate:
		// TODO (Matt): Need to lookup hostname.
		hostEpKey := model.HostEndpointKey{
			Hostname:   "matt-k8s",
			EndpointID: msg.Id.EndpointId,
		}
		m.hostEpMutex.Lock()
		// Store tiers and policies
		m.hostEndpointTiers[hostEpKey] = msg.Endpoint.Tiers
		m.hostEndpointUntrackedTiers[hostEpKey] = msg.Endpoint.UntrackedTiers
		// Store IP addresses
		for _, ipv4 := range msg.Endpoint.ExpectedIpv4Addrs {
			addr := net.ParseIP(ipv4)
			if addr == nil {
				log.Warn("Error parsing IP ", ipv4)
				continue
			}
			addrStr := addr.String()
			log.Debug("Stored expected IPv4 host endpoint: ", hostEpKey, ": ", addrStr)
			m.hostEndpoints[addrStr] = &hostEpKey
			m.hostEndpointsReverse[hostEpKey] = &addrStr
		}
		for _, ipv6 := range msg.Endpoint.ExpectedIpv6Addrs {
			addr := net.ParseIP(ipv6)
			if addr == nil {
				log.Warn("Error parsing IP ", ipv6)
				continue
			}
			// TODO (Matt): IP.String() does funny things to IPv6 mapped IPv4 addresses.
			addrStr := addr.String()
			log.Debug("Stored expected IPv6 host endpoint: ", hostEpKey, ": ", addrStr)
			m.hostEndpoints[addrStr] = &hostEpKey
			m.hostEndpointsReverse[hostEpKey] = &addrStr
		}
		m.hostEpMutex.Unlock()
	case *proto.HostEndpointRemove:
		hostEpKey := model.HostEndpointKey{
			Hostname:   "matt-k8s",
			EndpointID: msg.Id.EndpointId,
		}
		m.epMutex.Lock()
		epIp := m.hostEndpointsReverse[hostEpKey]
		if epIp != nil {
			delete(m.hostEndpoints, *epIp)
			delete(m.hostEndpointsReverse, hostEpKey)
			delete(m.hostEndpointTiers, hostEpKey)
			delete(m.hostEndpointUntrackedTiers, hostEpKey)
		}
		m.epMutex.Unlock()
	}
}

func (m *LookupManager) CompleteDeferredWork() error {
	return nil
}

// GetEndpointKey returns either a *model.WorkloadEndpointKey or *model.HostEndpointKey
// or nil if addr is a Workload Endpoint or a HostEndpoint or if we don't have any
// idea about it.
func (m *LookupManager) GetEndpointKey(addr net.IP) (interface{}, error) {
	addrStr := addr.String()
	m.epMutex.Lock()
	// There's no need to copy the result because we never modify fields,
	// only delete or replace.
	epKey := m.endpoints[addrStr]
	m.epMutex.Unlock()
	if epKey != nil {
		return epKey, nil
	}
	m.hostEpMutex.Lock()
	hostEpKey := m.hostEndpoints[addrStr]
	m.hostEpMutex.Unlock()
	if hostEpKey != nil {
		return hostEpKey, nil
	}
	return nil, UnknownEndpointError
}

// GetPolicyIndex returns the number of tiers that have been traversed before reaching a given Policy.
// For a profile, this means it returns the total number of tiers that apply.
// epKey is either a *model.WorkloadEndpointKey or *model.HostEndpointKey
func (m *LookupManager) GetPolicyIndex(epKey interface{}, policyKey *model.PolicyKey) (tiersBefore int) {
	switch epKey.(type) {
	case *model.WorkloadEndpointKey:
		ek := epKey.(*model.WorkloadEndpointKey)
		m.epMutex.Lock()
		tiers := m.endpointTiers[*ek]
		log.Debug("Checking tiers ", tiers, " against policy ", policyKey)
		for _, tier := range tiers {
			log.Debug("Checking endpoint tier ", tier)
			if tier.Name == policyKey.Tier {
				break
			} else {
				tiersBefore++
			}
		}
		log.Debug("TiersBefore: ", tiersBefore)
		m.epMutex.Unlock()
	case *model.HostEndpointKey:
		ek := epKey.(*model.HostEndpointKey)
		m.hostEpMutex.Lock()
		// TODO(doublek): Not sure how to consider Untracked Tiers into this.
		tiers := m.hostEndpointTiers[*ek]
		log.Debug("Checking tiers ", tiers, " against policy ", policyKey)
		for _, tier := range tiers {
			log.Debug("Checking endpoint tier ", tier)
			if tier.Name == policyKey.Tier {
				break
			} else {
				tiersBefore++
			}
		}
		log.Debug("TiersBefore: ", tiersBefore)
		m.hostEpMutex.Unlock()
	}
	return tiersBefore
}
