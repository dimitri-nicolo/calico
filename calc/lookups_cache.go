// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package calc

import "github.com/projectcalico/libcalico-go/lib/backend/model"

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
func (lc *LookupsCache) GetEndpoint(addr [16]byte) (EndpointData, bool) {
	return lc.epCache.GetEndpoint(addr)
}

// GetRuleIDFromNFLOGPrefix returns the RuleID associated with the supplied NFLOG prefix.
func (lc *LookupsCache) GetRuleIDFromNFLOGPrefix(prefix [64]byte) *RuleID {
	return lc.polCache.GetRuleIDFromNFLOGPrefix(prefix)
}

// SetMockData fills in some of the data structures for use in the test code. This should not
// be called from any mainline code.
func (lc *LookupsCache) SetMockData(
	em map[[16]byte]*model.WorkloadEndpointKey,
	nm map[[64]byte]*RuleID,
) {
	lc.polCache.nflogPrefixHash = nm
	for ip, wep := range em {
		ep := EndpointData{
			Key:          wep,
			Endpoint:     wep,
			OrderedTiers: []string{"default"},
		}
		lc.epCache.ipToEndpoints[ip] = []EndpointData{ep}
	}
}
