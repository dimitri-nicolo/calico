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
	networksetToCidrs map[model.Key]set.Set
	ipTree            *IpTrie
}

func NewNetworksetLookupsCache() *NetworksetLookupsCache {
	nc := &NetworksetLookupsCache{
		networksetToCidrs: map[model.Key]set.Set{},
		nsMutex:           sync.RWMutex{},
		ipTree:            NewIpTrie(),
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
// ip address in the ipTree.
// This method isn't safe to be used concurrently and the caller should acquire the
// NetworksetLookupsCache.nsMutex before calling this method.
func (nc *NetworksetLookupsCache) updateCIDRToNetworksetMapping(cidr ip.CIDR, ed *EndpointData) {
	nc.ipTree.InsertNetworkset(cidr, ed)
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

	// Remove existing CIDR and corresponding networkset mapping.
	nc.ipTree.DeleteNetworkset(cidr, key)

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

// GetNetworkset finds Longest Prefix Match CIDR from given IP ADDR and return last observed
// Networkset for that CIDR
func (nc *NetworksetLookupsCache) GetNetworkset(addr [16]byte) (*EndpointData, bool) {
	nc.nsMutex.RLock()
	defer nc.nsMutex.RUnlock()
	// Find the first cidr that contains the ip address to use for the lookup.
	ipAddr := ip.FromNetIP(net.IP(addr[:]))
	return nc.ipTree.GetLongestPrefixCidr(ipAddr)
}

func (nc *NetworksetLookupsCache) DumpNetworksets() string {
	nc.nsMutex.RLock()
	defer nc.nsMutex.RUnlock()
	lines := []string{}
	lines = nc.ipTree.DumpCIDRNetworksets()
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
