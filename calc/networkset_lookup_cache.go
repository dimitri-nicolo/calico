// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package calc

import (
	"net"
	"reflect"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
)

// Networkset data is stored in the EndpointData object for easier type processing for flow logs.
type NetworksetLookupsCache struct {
	nsMutex           sync.RWMutex
	cidrToNetworksets map[ip.CIDR][]*EndpointData
	networksetToCidrs map[model.Key]set.Set
}

func NewNetworksetLookupsCache() *NetworksetLookupsCache {
	nc := &NetworksetLookupsCache{
		cidrToNetworksets: map[ip.CIDR][]*EndpointData{},
		networksetToCidrs: map[model.Key]set.Set{},
		nsMutex:           sync.RWMutex{},
	}
	return nc
}

func (nc *NetworksetLookupsCache) RegisterWith(allUpdateDispatcher *dispatcher.Dispatcher) {
	allUpdateDispatcher.Register(model.NetworkSetKey{}, nc.OnUpdate)
}

// OnUpdate is the callback method registered with the AllUpdatesDispatcher for
// the model.NetworkSet type. This method updates the mapping between networksets
// and the corresponding CIDRs that they contain.
func (nc *NetworksetLookupsCache) OnUpdate(nsUpdate api.Update) (_ bool) {
	switch k := nsUpdate.Key.(type) {
	case model.NetworkSetKey:
		if nsUpdate.Value == nil {
			nc.removeNetworkset(k)
		} else {
			networkset := nsUpdate.Value.(*model.NetworkSet)
			ed := &EndpointData{
				Key:        k,
				Networkset: nsUpdate.Value,
			}
			nc.addOrUpdateNetworkset(k, ed, ip.CIDRsFromCalicoNets(networkset.Nets))
		}
	default:
		log.Infof("ignoring unexpected update: %v %#v",
			reflect.TypeOf(nsUpdate.Key), nsUpdate)
		return
	}
	log.Infof("Updating networkset cache with networkset data %v", nsUpdate.Key)
	return
}

// addOrUpdateNetworkset tracks networkset to CIDR mapping as well as the reverse
// mapping from CIDR to networkset.
func (nc *NetworksetLookupsCache) addOrUpdateNetworkset(key model.Key, ed *EndpointData, nets []ip.CIDR) {
	// If the networkset exists, it was updated, then we might have to add or
	// remove CIDRs.
	// First up, get all current CIDRs.
	// Note: We don't acquire a lock when reading from the mapping
	// because the only writes are done via the calc_graph thread.
	var cidrsToRemove set.Set
	currentCIDRs, ok := nc.networksetToCidrs[key]
	// Create a copy so any working changes do not affect the actual set.
	if !ok {
		cidrsToRemove = set.New()
	} else {
		cidrsToRemove = currentCIDRs.Copy()
	}

	// Collect all the CIDRs that correspond to this networkset and mark
	// any CIDR that shouldn't be discarded.
	newCIDRs := set.New()
	for _, cidr := range nets {
		// If this is an already existing CIDR, then remove it from the removal set
		// which will be used later to delete CIDRs.
		if cidrsToRemove.Contains(cidr) {
			cidrsToRemove.Discard(cidr)
		}
		newCIDRs.Add(cidr)
	}

	nc.nsMutex.Lock()
	defer nc.nsMutex.Unlock()
	newCIDRs.Iter(func(item interface{}) error {
		newCIDR := item.(ip.CIDR)
		nc.updateCIDRToNetworksetMapping(newCIDR, ed)
		return nil
	})
	cidrsToRemove.Iter(func(item interface{}) error {
		cidr := item.(ip.CIDR)
		nc.removeNetworksetCidrMapping(key, cidr)
		return set.RemoveItem
	})
	if newCIDRs.Len() != 0 {
		nc.networksetToCidrs[key] = newCIDRs
	}

	// At this point, we can check if we need to update endpoint data
	// for existing IP addresses.
	ccidrs, ok := nc.networksetToCidrs[key]
	if !ok {
		return
	}
	ccidrs.Iter(func(item interface{}) error {
		curCIDR := item.(ip.CIDR)
		if newCIDRs.Contains(curCIDR) {
			// Already updated above.
			return nil
		}
		nc.updateCIDRToNetworksetMapping(curCIDR, ed)
		return nil
	})
}

// updateCIDRToNetworksetMapping creates or appends the EndpointData to a corresponding
// ip address in the cidrToNetworksets map.
// This method isn't safe to be used concurrently and the caller should acquire the
// NetworksetLookupsCache.nsMutex before calling this method.
func (nc *NetworksetLookupsCache) updateCIDRToNetworksetMapping(cidr ip.CIDR, ed *EndpointData) {
	// Check if this CIDR already has a corresponding networkset.
	// If it has one, then append the networkset to it. This is
	// expected to happen if a CIDR is reused in a very
	// short interval. Otherwise, create a new CIDR to networkset
	// mapping entry.
	existingNetsets, ok := nc.cidrToNetworksets[cidr]
	if !ok {
		nc.cidrToNetworksets[cidr] = []*EndpointData{ed}
	} else {
		isExistingNetset := false
		for i := range existingNetsets {
			// Check if this is an existing networkset. If it is,
			// then just store the updated networkset.
			if ed.Key == existingNetsets[i].Key {
				existingNetsets[i] = ed
				isExistingNetset = true
				break
			}
		}
		if !isExistingNetset {
			existingNetsets = append(existingNetsets, ed)
		}
		nc.cidrToNetworksets[cidr] = existingNetsets
	}
}

// removeNetworkset removes the networkset from the NetworksetLookupscache.networksetToCidrs map
// and also removes all corresponding CIDR to networkset mappings as well.
// This method should acquire (and release) the NetworksetLookupsCache.nsMutex before (and after)
// manipulating the maps.
func (nc *NetworksetLookupsCache) removeNetworkset(key model.Key) {
	nc.nsMutex.Lock()
	defer nc.nsMutex.Unlock()
	currentCIDRs, ok := nc.networksetToCidrs[key]
	if !ok {
		// We don't know about this networkset. Nothing to do.
		return
	}
	currentCIDRs.Iter(func(item interface{}) error {
		cidr := item.(ip.CIDR)
		nc.removeNetworksetCidrMapping(key, cidr)
		return nil
	})
	delete(nc.networksetToCidrs, key)
}

func (nc *NetworksetLookupsCache) removeNetworksetCidrMapping(key model.Key, cidr ip.CIDR) {
	// Remove existing CIDR to networkset mapping.
	existingNetsets, ok := nc.cidrToNetworksets[cidr]
	if !ok || len(existingNetsets) == 1 {
		// There are no entries or only a single endpoint corresponding
		// to thiss IP address so it is safe to remove this mapping.
		delete(nc.cidrToNetworksets, cidr)
	} else {
		// if there is more than one networkset, then keep the reverse cidr
		// to networkset mapping but only remove the networkset corresponding
		// to this remove call.
		newNetsets := make([]*EndpointData, 0, len(existingNetsets)-1)
		for _, netset := range existingNetsets {
			if netset.Key == key {
				continue
			}
			newNetsets = append(newNetsets, netset)
		}
		nc.cidrToNetworksets[cidr] = newNetsets
	}

	// Remove the networkset to CIDR mapping.
	existingCidrs, ok := nc.networksetToCidrs[key]
	if !ok {
		return
	}
	if existingCidrs.Len() == 1 {
		// There is only a single cidr corresponding to this
		// key so it is safe to remove this mapping.
		delete(nc.networksetToCidrs, key)
	} else {
		existingCidrs.Discard(cidr)
		nc.networksetToCidrs[key] = existingCidrs
	}
}

//getLongestPrefixMatch Function Iterates over all CIDRS, and return a Longest prefix match
// CIDR for the given IP ADDR
func (nc *NetworksetLookupsCache) getLongestPrefixMatch(addr [16]byte) (ip.CIDR, bool) {
	var ok bool = false
	var max int
	var cidrLpm ip.CIDR
	ipAddr := ip.FromNetIP(net.IP(addr[:]))
	ipBin := ipAddr.AsBinary()

	for cidr, _ := range nc.cidrToNetworksets {
		cidrBin := cidr.AsBinary()
		if strings.HasPrefix(ipBin, cidrBin) {
			if len(cidrBin) > max {
				max = len(cidrBin)
				cidrLpm = cidr
				ok = true
			}
		}
	}

	return cidrLpm, ok
}

// GetNetworkset finds Longest Prefix Match CIDR from given IP ADDR and return last observed
// Networkset for that CIDR
func (nc *NetworksetLookupsCache) GetNetworkset(addr [16]byte) (*EndpointData, bool) {
	nc.nsMutex.RLock()
	defer nc.nsMutex.RUnlock()
	// Find the first cidr that contains the ip address to use for the lookup.
	var netsets []*EndpointData
	var ok bool
	var cidrLpm ip.CIDR

	cidrLpm, ok = nc.getLongestPrefixMatch(addr)
	if ok {
		netsets, ok = nc.cidrToNetworksets[cidrLpm]
		if len(netsets) >= 1 && ok {
			// We return the last observed networkset.
			return netsets[len(netsets)-1], ok
		}
	}
	return nil, ok
}

func (nc *NetworksetLookupsCache) DumpNetworksets() string {
	nc.nsMutex.RLock()
	defer nc.nsMutex.RUnlock()
	lines := []string{}
	for cidr, eds := range nc.cidrToNetworksets {
		edNames := []string{}
		for _, ed := range eds {
			edNames = append(edNames, ed.Key.(model.NetworkSetKey).Name)
		}
		lines = append(lines, cidr.String()+": "+strings.Join(edNames, ","))
	}
	lines = append(lines, "-------")
	for key, cidrs := range nc.networksetToCidrs {
		cidrStr := []string{}
		cidrs.Iter(func(item interface{}) error {
			cidr := item.(ip.CIDR)
			cidrStr = append(cidrStr, cidr.String())
			return nil
		})
		lines = append(lines, key.(model.NetworkSetKey).Name, ": ", strings.Join(cidrStr, ","))
	}
	return strings.Join(lines, "\n")
}
