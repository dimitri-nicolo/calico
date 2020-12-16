// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.

package calc

import (
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/proxy"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
)

// LookupsCache provides an API to do the following:
// - lookup endpoint information given an IP
// - lookup policy/profile information given the NFLOG prefix
//
// To do this, the LookupsCache uses two caches to hook into the
// calculation graph at various stages
// - EndpointLookupsCache
// - PolicyLookupsCache
type LookupsCache struct {
	polCache *PolicyLookupsCache
	epCache  *EndpointLookupsCache
	nsCache  *NetworksetLookupsCache
	svcCache *ServiceLookupsCache
}

func NewLookupsCache() *LookupsCache {
	lc := &LookupsCache{
		polCache: NewPolicyLookupsCache(),
		epCache:  NewEndpointLookupsCache(),
		nsCache:  NewNetworksetLookupsCache(),
		svcCache: NewServiceLookupsCache(),
	}
	return lc
}

// IsEndpoint returns true if the supplied address is a endpoint, otherwise returns false.
// Use the EndpointData.IsLocal() method to check if an EndpointData object (returned by the
// LookupsCache.GetEndpoint() method) is a local endpoint or not.
func (lc *LookupsCache) IsEndpoint(addr [16]byte) bool {
	return lc.epCache.IsEndpoint(addr)
}

// GetEndpoint returns the ordered list of tiers for a particular endpoint.
func (lc *LookupsCache) GetEndpoint(addr [16]byte) (*EndpointData, bool) {
	return lc.epCache.GetEndpoint(addr)
}

// GetNode returns the node configured with the supplied address. This matches against one of the following:
// - The node IP address
// - The node IPIP tunnel address
// - The node VXLAN tunnel address
// - The node wireguard tunnel address
func (lc *LookupsCache) GetNode(addr [16]byte) (string, bool) {
	return lc.epCache.GetNode(addr)
}

// GetNetworkset returns the networkset information for an address.
// It returns the first networkset it finds that contains the given address.
func (lc *LookupsCache) GetNetworkset(addr [16]byte) (*EndpointData, bool) {
	return lc.nsCache.GetNetworkset(addr)
}

// GetRuleIDFromNFLOGPrefix returns the RuleID associated with the supplied NFLOG prefix.
func (lc *LookupsCache) GetRuleIDFromNFLOGPrefix(prefix [64]byte) *RuleID {
	return lc.polCache.GetRuleIDFromNFLOGPrefix(prefix)
}

// GetServiceFromPreNATDest looks up a service by cluster/external IP.
func (lc *LookupsCache) GetServiceFromPreDNATDest(ipPreDNAT [16]byte, portPreDNAT int, proto int) (proxy.ServicePortName, bool) {
	return lc.svcCache.GetServiceFromPreDNATDest(ipPreDNAT, portPreDNAT, proto)
}

// GetNodePortService looks up a service by port and protocol (assuming a node IP).
func (lc *LookupsCache) GetNodePortService(port int, proto int) (proxy.ServicePortName, bool) {
	return lc.svcCache.GetNodePortService(port, proto)
}

// SetMockData fills in some of the data structures for use in the test code. This should not
// be called from any mainline code.
func (lc *LookupsCache) SetMockData(
	em map[[16]byte]*EndpointData,
	nm map[[64]byte]*RuleID,
	ns map[model.NetworkSetKey]*model.NetworkSet,
	svcs map[model.ResourceKey]*kapiv1.Service,
) {
	for ip, ed := range em {
		if ed == nil {
			delete(lc.epCache.ipToEndpoints, ip)
		} else {
			lc.epCache.ipToEndpoints[ip] = []*EndpointData{ed}
		}
	}
	for id, rid := range nm {
		if rid == nil {
			delete(lc.polCache.nflogPrefixHash, id)
		} else {
			lc.polCache.nflogPrefixHash[id] = rid
		}
	}
	for k, v := range ns {
		lc.nsCache.OnUpdate(api.Update{KVPair: model.KVPair{Key: k, Value: v}})
	}
	for k, v := range svcs {
		lc.svcCache.OnResourceUpdate(api.Update{KVPair: model.KVPair{Key: k, Value: v}})
	}
}

// GetEndpointKeys returns all endpoint keys that the cache is tracking.
// Convenience method only used for testing purposes.
func (lc *LookupsCache) GetEndpointKeys() []model.Key {
	lc.epCache.epMutex.RLock()
	defer lc.epCache.epMutex.RUnlock()
	eps := []model.Key{}
	for key, _ := range lc.epCache.endpointToIps {
		eps = append(eps, key)
	}
	return eps
}

// GetEndpointData returns all endpoint data that the cache is tracking.
// Convenience method only used for testing purposes.
func (lc *LookupsCache) GetAllEndpointData() []*EndpointData {
	lc.epCache.epMutex.RLock()
	defer lc.epCache.epMutex.RUnlock()
	uniq := set.New()
	allEds := []*EndpointData{}
	for _, eds := range lc.epCache.ipToEndpoints {
		for _, ed := range eds {
			if uniq.Contains(ed.Key) {
				continue
			}
			uniq.Add(ed.Key)
			allEds = append(allEds, ed)
		}
	}
	return allEds
}
