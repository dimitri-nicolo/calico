// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package lookup

import (
	"errors"
	"fmt"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/hashutils"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var UnknownEndpointError = errors.New("Unknown endpoint")

type QueryInterface interface {
	GetEndpointKey(addr [16]byte) (interface{}, error)
	GetTierIndex(epKey interface{}, tierName string) int
	GetNFLOGHashToPolicyID(prefixHash [64]byte) ([]byte, error)
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

	nflogPrefixHash map[[64]byte][]byte
	nflogMutex      sync.RWMutex
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
		nflogPrefixHash:            map[[64]byte][]byte{},
		nflogMutex:                 sync.RWMutex{},
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

	case *proto.ActivePolicyUpdate:
		m.PushPolicyNFLOGPrefixHash(msg)

	case *proto.ActivePolicyRemove:
		m.PopPolicyNFLOGPrefixHash(msg)

	case *proto.ActiveProfileUpdate:
		m.PushProfileNFLOGPrefixHash(msg)

	case *proto.ActiveProfileRemove:
		m.PopProfileNFLOGPrefixHash(msg)

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

// PushPolicyNFLOGPrefixHash takes proto message (ActivePolicyUpdate) and calculates the NFLOG prefix, and if the prefix is too long then
// calculates the hash. Finally it pushes it to a map for stats collector to access.
func (m *LookupManager) PushPolicyNFLOGPrefixHash(msg *proto.ActivePolicyUpdate) {
	// NFLOG prefix which is a combination of action, rule index, policy/profile and tier name
	// separated by `|`s. Example: "D|0|default.deny-icmp|po".
	// We calculate the hash of the prefix's policy/profile name part (which includes tier name and namespace, if applicable)
	// if its length exceeds NFLOG prefix max length which is 64 characters - 9 (2 for first (A|D|N) then a `|` then
	// 3 digits for up to 999 for rule indexes, a `|` after that and 3 more for the `|po` suffix at the end.
	//
	// See iptables/actions.go ToFragment() func, this needs to be in sync,
	// if you are updating the current function, we probably need to change that one as well.
	prefixHash := hashutils.GetLengthLimitedID("", fmt.Sprintf("%s.%s", msg.Id.Tier, msg.Id.Name), rules.NFLOGPrefixMaxLength-9)
	prefixHash += "|po"

	var bph [64]byte
	copy(bph[:], []byte(prefixHash[:]))

	m.nflogMutex.Lock()
	// Store the hash in a map. With hash being the key and string being the value.
	m.nflogPrefixHash[bph] = []byte(fmt.Sprintf("%s.%s|po", msg.Id.Tier, msg.Id.Name))
	m.nflogMutex.Unlock()
}

// PushProfileNFLOGPrefixHash takes proto message (ActiveProfileUpdate) and calculates the NFLOG prefix, and if the prefix is too long then
// calculates the hash. Finally it pushes it to a map for stats collector to access.
func (m *LookupManager) PushProfileNFLOGPrefixHash(msg *proto.ActiveProfileUpdate) {
	// NFLOG prefix which is a combination of action, rule index, policy/profile and tier name
	// separated by `|`s. Example: "D|0|profile-name|pr".
	// We calculate the hash of the prefix's policy/profile name part (which includes tier name and namespace, if applicable)
	// if its length exceeds NFLOG prefix max length which is 64 characters - 9 (2 for first (A|D|N) then a `|` then
	// 3 digits for up to 999 for rule indexes, a `|` after that and 3 more for the `|po` suffix at the end.
	//
	// See iptables/actions.go ToFragment() func, this needs to be in sync,
	// if you are updating the current function, we probably need to change that one as well.
	prefixHash := hashutils.GetLengthLimitedID("", msg.Id.Name, rules.NFLOGPrefixMaxLength-9)
	prefixHash += "|pr"

	var bph [64]byte
	copy(bph[:], []byte(prefixHash[:]))

	m.nflogMutex.Lock()
	// Store the hash in a map. With hash being the key and string being the value.
	m.nflogPrefixHash[bph] = []byte(fmt.Sprintf("%s|pr", msg.Id.Name))
	m.nflogMutex.Unlock()
}

// PopPolicyNFLOGPrefixHash takes proto message (ActivePolicyUpdate) and removes the hash to policy name map entry
// from the nflogPrefixHash map.
func (m *LookupManager) PopPolicyNFLOGPrefixHash(msg *proto.ActivePolicyRemove) {
	var honByte [64]byte
	hashOrName := msg.Id.Name

	// +10 because 2 for first (A|D|N) then a `|` then 3 digits for up to 999 for rule indexes, a `|` after that
	// and 3 more for the `|po` suffix at the end and 1 for the `.` between tier and policy name.
	if len(msg.Id.Name)+len(msg.Id.Tier)+10 > rules.NFLOGPrefixMaxLength {
		hashOrName = hashutils.GetLengthLimitedID("", fmt.Sprintf("%s.%s", msg.Id.Tier, msg.Id.Name), rules.NFLOGPrefixMaxLength-9)
	}

	hashOrName += "|po"

	copy(honByte[:], []byte(hashOrName))

	m.nflogMutex.Lock()
	delete(m.nflogPrefixHash, honByte)
	m.nflogMutex.Unlock()
}

// PopProfileNFLOGPrefixHash takes proto message (ActiveProfileRemove) and removes the hash to profile name map entry
// from the nflogPrefixHash map.
func (m *LookupManager) PopProfileNFLOGPrefixHash(msg *proto.ActiveProfileRemove) {
	var honByte [64]byte

	hashOrName := msg.Id.Name

	// +9 because 2 for first (A|D|N) then a `|` then 3 digits for up to 999 for rule indexes, a `|` after that
	// and 3 more for the `|po` suffix at the end.
	if len(msg.Id.Name)+9 > rules.NFLOGPrefixMaxLength {
		hashOrName = hashutils.GetLengthLimitedID("", msg.Id.Name, rules.NFLOGPrefixMaxLength-6)
	}
	hashOrName += "|pr"

	copy(honByte[:], []byte(hashOrName))

	m.nflogMutex.Lock()
	delete(m.nflogPrefixHash, honByte)
	m.nflogMutex.Unlock()
}

// GetNFLOGHashToPolicyID returns unhashed policy/profile ID (name) associated with the NFLOG prefix string or hash from the nflogPrefixHash map.
func (m *LookupManager) GetNFLOGHashToPolicyID(prefixHash [64]byte) ([]byte, error) {
	m.nflogMutex.RLock()
	policyID, ok := m.nflogPrefixHash[prefixHash]
	m.nflogMutex.RUnlock()
	if !ok {
		return []byte{}, fmt.Errorf("cannot find the specified NFLOG prefix string or hash: %s in the lookup manager", prefixHash)
	}

	return policyID, nil
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
