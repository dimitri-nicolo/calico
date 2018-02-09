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

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var UnknownEndpointError = errors.New("Unknown endpoint")

type QueryInterface interface {
	GetEndpointKey(addr [16]byte) (interface{}, error)
	GetTierIndex(epKey interface{}, tierName string) int
}

// TODO (Matt): WorkloadEndpoints are only local; so we can't get nice information for the remote ends.
type LookupManager struct {
	// `string`s are IP.String().
	endpoints        map[[16]byte]*model.WorkloadEndpointKey
	endpointsReverse map[model.WorkloadEndpointKey]*[16]byte
	endpointTiers    map[model.WorkloadEndpointKey][]*proto.TierInfo
	epMutex          sync.RWMutex

	hostEndpoints              map[[16]byte]*model.HostEndpointKey
	hostEndpointsReverse       map[model.HostEndpointKey]*[16]byte
	hostEndpointTiers          map[model.HostEndpointKey][]*proto.TierInfo
	hostEndpointUntrackedTiers map[model.HostEndpointKey][]*proto.TierInfo
	hostEpMutex                sync.RWMutex
}

func NewLookupManager() *LookupManager {
	return &LookupManager{
		endpoints:                  map[[16]byte]*model.WorkloadEndpointKey{},
		endpointsReverse:           map[model.WorkloadEndpointKey]*[16]byte{},
		endpointTiers:              map[model.WorkloadEndpointKey][]*proto.TierInfo{},
		hostEndpoints:              map[[16]byte]*model.HostEndpointKey{},
		hostEndpointsReverse:       map[model.HostEndpointKey]*[16]byte{},
		hostEndpointTiers:          map[model.HostEndpointKey][]*proto.TierInfo{},
		hostEndpointUntrackedTiers: map[model.HostEndpointKey][]*proto.TierInfo{},
		epMutex:                    sync.RWMutex{},
		hostEpMutex:                sync.RWMutex{},
	}
}

func (m *LookupManager) OnUpdate(protoBufMsg interface{}) {
	switch msg := protoBufMsg.(type) {
	case *proto.WorkloadEndpointUpdate:
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		wlEpKey := model.WorkloadEndpointKey{
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
				log.Warnf("Error parsing CIDR %v", ipv4)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.endpoints[addrB] = &wlEpKey
			m.endpointsReverse[wlEpKey] = &addrB
		}
		for _, ipv6 := range msg.Endpoint.Ipv6Nets {
			addr, _, err := net.ParseCIDR(ipv6)
			if err != nil {
				log.Warnf("Error parsing CIDR %v", ipv6)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.endpoints[addrB] = &wlEpKey
			m.endpointsReverse[wlEpKey] = &addrB
		}
		m.epMutex.Unlock()
	case *proto.WorkloadEndpointRemove:
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		wlEpKey := model.WorkloadEndpointKey{
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
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		hostEpKey := model.HostEndpointKey{
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
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.hostEndpoints[addrB] = &hostEpKey
			m.hostEndpointsReverse[hostEpKey] = &addrB
		}
		for _, ipv6 := range msg.Endpoint.ExpectedIpv6Addrs {
			addr := net.ParseIP(ipv6)
			if addr == nil {
				log.Warn("Error parsing IP ", ipv6)
				continue
			}
			var addrB [16]byte
			copy(addrB[:], addr.To16()[:16])
			m.hostEndpoints[addrB] = &hostEpKey
			m.hostEndpointsReverse[hostEpKey] = &addrB
		}
		m.hostEpMutex.Unlock()
	case *proto.HostEndpointRemove:
		// We omit setting Hostname since at this point it is implied that the
		// endpoint belongs to this host.
		hostEpKey := model.HostEndpointKey{
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
func (m *LookupManager) GetEndpointKey(addr [16]byte) (interface{}, error) {
	m.epMutex.RLock()
	// There's no need to copy the result because we never modify fields,
	// only delete or replace.
	epKey := m.endpoints[addr]
	m.epMutex.RUnlock()
	if epKey != nil {
		return epKey, nil
	}
	m.hostEpMutex.RLock()
	hostEpKey := m.hostEndpoints[addr]
	m.hostEpMutex.RUnlock()
	if hostEpKey != nil {
		return hostEpKey, nil
	}
	return nil, UnknownEndpointError
}

// GetTierIndex returns the number of tiers that have been traversed before reaching a given Tier.
// For a profile, this means it returns the total number of tiers that apply.
// epKey is either a *model.WorkloadEndpointKey or *model.HostEndpointKey
//TODO: RLB: Do we really need to keep track of EP vs. Tier indexes?  Seems an overkill - we only need
// to know the overall tier order to determine the order of the NFLOGs in a set of traces.
func (m *LookupManager) GetTierIndex(epKey interface{}, tierName string) (tiersBefore int) {
	switch epKey.(type) {
	case *model.WorkloadEndpointKey:
		ek := epKey.(*model.WorkloadEndpointKey)
		m.epMutex.RLock()
		tiers := m.endpointTiers[*ek]
		for _, tier := range tiers {
			if tier.Name == tierName {
				break
			} else {
				tiersBefore++
			}
		}
		m.epMutex.RUnlock()
	case *model.HostEndpointKey:
		ek := epKey.(*model.HostEndpointKey)
		m.hostEpMutex.RLock()
		tiers := append(m.hostEndpointUntrackedTiers[*ek], m.hostEndpointTiers[*ek]...)
		for _, tier := range tiers {
			if tier.Name == tierName {
				break
			} else {
				tiersBefore++
			}
		}
		m.hostEpMutex.RUnlock()
	}
	return tiersBefore
}
