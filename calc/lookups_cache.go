// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package calc

import (
	"runtime"

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
}

func NewLookupsCache() *LookupsCache {
	if runtime.GOOS == "windows" {
		// For windows OS, return nil lookupCache and rest of all lookupCache
		// should handle the nil pointer
		return nil
	}

	lc := &LookupsCache{
		polCache: NewPolicyLookupsCache(),
		epCache:  NewEndpointLookupsCache(),
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

// GetRuleIDFromNFLOGPrefix returns the RuleID associated with the supplied NFLOG prefix.
func (lc *LookupsCache) GetRuleIDFromNFLOGPrefix(prefix [64]byte) *RuleID {
	return lc.polCache.GetRuleIDFromNFLOGPrefix(prefix)
}

// SetMockData fills in some of the data structures for use in the test code. This should not
// be called from any mainline code.
func (lc *LookupsCache) SetMockData(
	em map[[16]byte]*EndpointData,
	nm map[[64]byte]*RuleID,
) {
	lc.polCache.nflogPrefixHash = nm
	for ip, ed := range em {
		lc.epCache.ipToEndpoints[ip] = []*EndpointData{ed}
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
