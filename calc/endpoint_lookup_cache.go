// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package calc

import (
	"net"
	"reflect"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type EndpointData struct {
	Key          model.Key
	Endpoint     interface{}
	OrderedTiers []string

	ImplicitDropIngressRuleID map[string]*RuleID
	ImplicitDropEgressRuleID  map[string]*RuleID
}

// IsLocal returns if this EndpointData corresponds to a local endpoint or not.
// This works because, we don't process tier information for remote endpoints
// and only do so for remote endpoints.
func (e *EndpointData) IsLocal() bool {
	return e.OrderedTiers != nil
}

// IsHostEndpoint returns if this EndpointData corresponds to a hostendpoint.
func (e *EndpointData) IsHostEndpoint() (isHep bool) {
	switch e.Key.(type) {
	case model.HostEndpointKey:
		isHep = true
	}
	return
}

// EndpointLookupsCache provides an API to lookup endpoint information given
// an IP address.
//
// To do this, the EndpointLookupsCache hooks into the calculation graph
// by handling callbacks for updated local endpoint tier information.
//
// It also functions as a node that is part of the calculation graph
// to handle remote endpoint information. To do this, it registers
// with the remote endpoint dispatcher and updates the endpoint
// cache appropriately.
type EndpointLookupsCache struct {
	epMutex       sync.RWMutex
	ipToEndpoints map[[16]byte][]*EndpointData
	endpointToIps map[model.Key]set.Set
}

func NewEndpointLookupsCache() *EndpointLookupsCache {
	ec := &EndpointLookupsCache{
		ipToEndpoints: map[[16]byte][]*EndpointData{},
		endpointToIps: map[model.Key]set.Set{},
		epMutex:       sync.RWMutex{},
	}
	return ec
}

func (ec *EndpointLookupsCache) RegisterWith(remoteEndpointDispatcher *dispatcher.Dispatcher) {
	remoteEndpointDispatcher.Register(model.WorkloadEndpointKey{}, ec.OnUpdate)
	remoteEndpointDispatcher.Register(model.HostEndpointKey{}, ec.OnUpdate)
}

// OnEndpointTierUpdate is called by the PolicyResolver when it figures out tiers that apply
// to an endpoint. This method tracks local endpoint (model.WorkloadEndpoint and model.HostEndpoint)
// and corresponding IP address relationship. The difference between this handler and the OnUpdate
// handler (below) is this method records tier information for local endpoints while this information
// is ignored for remote endpoints.
func (ec *EndpointLookupsCache) OnEndpointTierUpdate(key model.Key, ep interface{}, filteredTiers []tierInfo) {
	createEndpointData := func() *EndpointData {
		ed := &EndpointData{
			Key:                       key,
			Endpoint:                  ep,
			OrderedTiers:              make([]string, len(filteredTiers)),
			ImplicitDropIngressRuleID: map[string]*RuleID{},
			ImplicitDropEgressRuleID:  map[string]*RuleID{},
		}
		for i := range filteredTiers {
			ti := filteredTiers[i]
			ed.OrderedTiers[i] = ti.Name
			if len(ti.OrderedPolicies) == 0 {
				continue
			}
			var ingressAssigned, egressAssigned bool
			for j := len(ti.OrderedPolicies) - 1; j >= 0; j-- {
				pol := ti.OrderedPolicies[j]
				namespace, tier, name, err := deconstructPolicyName(pol.Key.Name)
				if err != nil {
					log.WithError(err).Error("Unable to parse policy name")
					continue
				}
				if pol.GovernsIngress() && !ingressAssigned {
					rid := NewRuleID(tier, name, namespace, RuleIDIndexImplicitDrop, rules.RuleDirIngress, rules.RuleActionDeny)
					ed.ImplicitDropIngressRuleID[ti.Name] = rid
					ingressAssigned = true
				}
				if pol.GovernsEgress() && !egressAssigned {
					rid := NewRuleID(tier, name, namespace, RuleIDIndexImplicitDrop, rules.RuleDirEgress, rules.RuleActionDeny)
					ed.ImplicitDropEgressRuleID[ti.Name] = rid
					egressAssigned = true
				}
			}
		}
		return ed
	}
	switch k := key.(type) {
	case model.WorkloadEndpointKey:
		if ep == nil {
			ec.removeEndpoint(k)
		} else {
			endpoint := ep.(*model.WorkloadEndpoint)
			ed := createEndpointData()
			ec.addOrUpdateEndpoint(k, ed, extractIPsFromWorkloadEndpoint(endpoint))
		}
	case model.HostEndpointKey:
		if ep == nil {
			ec.removeEndpoint(k)
		} else {
			endpoint := ep.(*model.HostEndpoint)
			ed := createEndpointData()
			ec.addOrUpdateEndpoint(k, ed, extractIPsFromHostEndpoint(endpoint))
		}
	}
	log.Infof("Updating endpoint cache with local endpoint data %v", key)
	return
}

// OnUpdate is the callback method registered with the RemoteEndpointDispatcher for
// model.WorkloadEndpoint and model.HostEndpoint types. This method updates the
// mapping between an remote endpoint and all the IPs that the endpoint contains.
// The difference between OnUpdate and OnEndpointTierUpdate is that this method
// does not track tier information for a remote endpoint endpoint whereas
// OnEndpointTierUpdate tracks a local endpoint and records it's corresponding tier
// information as well.
func (ec *EndpointLookupsCache) OnUpdate(epUpdate api.Update) (_ bool) {
	switch k := epUpdate.Key.(type) {
	case model.WorkloadEndpointKey:
		if epUpdate.Value == nil {
			ec.removeEndpoint(k)
		} else {
			endpoint := epUpdate.Value.(*model.WorkloadEndpoint)
			ed := &EndpointData{
				Key:      k,
				Endpoint: epUpdate.Value,
			}
			ec.addOrUpdateEndpoint(k, ed, extractIPsFromWorkloadEndpoint(endpoint))
		}
	case model.HostEndpointKey:
		if epUpdate.Value == nil {
			ec.removeEndpoint(k)
		} else {
			endpoint := epUpdate.Value.(*model.HostEndpoint)
			ed := &EndpointData{
				Key:      k,
				Endpoint: epUpdate.Value,
			}
			ec.addOrUpdateEndpoint(k, ed, extractIPsFromHostEndpoint(endpoint))
		}
	default:
		log.Infof("Ignoring unexpected update: %v %#v",
			reflect.TypeOf(epUpdate.Key), epUpdate)
		return
	}
	log.Infof("Updating endpoint cache with remote endpoint data %v", epUpdate.Key)
	return
}

// addOrUpdateEndpoint tracks endpoint to IP mapping as well as IP to endpoint reverse mapping
// for a workload or host endpoint.
func (ec *EndpointLookupsCache) addOrUpdateEndpoint(key model.Key, ed *EndpointData, nets [][16]byte) {
	// If the endpoint exists, it was updated, then we might have to add or
	// remove IPs.
	// First up, get all current ip addresses.
	// Note: We don't acquire a lock when reading this map to get the current IP addresses
	// because the only writes are done via the calc_graph thread.
	var ipsToRemove set.Set
	currentIPs, ok := ec.endpointToIps[key]
	// Create a copy so that we can figure out which IPs to keep and
	// which ones to remove.
	if !ok {
		ipsToRemove = set.New()
	} else {
		ipsToRemove = currentIPs.Copy()
	}

	// Collect all IPs that correspond to this endpoint and mark
	// any IP that shouldn't be discarded.
	newIPs := set.New()
	for _, ip := range nets {
		// If this is an already existing IP, then remove it,
		// but skip adding the the new IP list to avoid adding
		// duplicate endpoints.
		if ipsToRemove.Contains(ip) {
			ipsToRemove.Discard(ip)
			continue
		}
		newIPs.Add(ip)
	}

	ec.epMutex.Lock()
	defer ec.epMutex.Unlock()
	newIPs.Iter(func(item interface{}) error {
		newIP := item.([16]byte)
		ec.updateIPToEndpointMapping(newIP, ed)
		return nil
	})
	ipsToRemove.Iter(func(item interface{}) error {
		ip := item.([16]byte)
		ec.removeEndpointIpMapping(key, ip)
		return set.RemoveItem
	})
	if newIPs.Len() != 0 {
		ec.endpointToIps[key] = newIPs
	}

	// At this point, we can check if we need to update endpoint data
	// for existing IP addresses.
	cips, ok := ec.endpointToIps[key]
	if !ok {
		return
	}
	cips.Iter(func(item interface{}) error {
		curIP := item.([16]byte)
		if newIPs.Contains(curIP) {
			// Already updated above.
			return nil
		}
		ec.updateIPToEndpointMapping(curIP, ed)
		return nil
	})
}

// updateIPToEndpointMapping creates or appends the EndpointData to a corresponding
// ip address in the ipToEndpoints map.
// This method isn't safe to be used concurrently and the caller should acquire the
// EndpointLookupsCache.epMutex before calling this method.
func (ec *EndpointLookupsCache) updateIPToEndpointMapping(ip [16]byte, ed *EndpointData) {
	// Check if this IP is already has a corresponding endpoint.
	// If it has one, then append the endpoint to it. This is
	// expected to happen if an IP address is reused in a very
	// short interval. Otherwise, create a new IP to endpoint
	// mapping entry.
	existingEps, ok := ec.ipToEndpoints[ip]
	if !ok {
		ec.ipToEndpoints[ip] = []*EndpointData{ed}
	} else {
		isExistingEp := false
		for i := range existingEps {
			// Check if this is an existing endpoint. If it is,
			// then just store the updated endpoint.
			if ed.Key == existingEps[i].Key {
				existingEps[i] = ed
				isExistingEp = true
				break
			}
		}
		if !isExistingEp {
			existingEps = append(existingEps, ed)
		}
		ec.ipToEndpoints[ip] = existingEps
	}
}

// removeEndpoint removes the endpoint from the EndpointLookupsCache.endpointToIps map
// and all also removes all correspondoing IP to endpoint mapping as well.
// This method acquires (and releases) the EndpointLookupsCache.epMutex before (and after)
// manipulating the maps.
func (ec *EndpointLookupsCache) removeEndpoint(key model.Key) {
	ec.epMutex.Lock()
	defer ec.epMutex.Unlock()
	currentIPs, ok := ec.endpointToIps[key]
	if !ok {
		// We don't know about this endpoint. Nothing to do here.
		return
	}
	currentIPs.Iter(func(item interface{}) error {
		ip := item.([16]byte)
		ec.removeEndpointIpMapping(key, ip)
		return nil
	})
	delete(ec.endpointToIps, key)
}

// removeEndpointIpMapping checks if  there is an existing
//  - IP to WEP/HEP relation that is being tracked and removes it if there is one.
//  - Endpoint to IP relation that is being tracked and removes it if there is one.
// This method isn't safe to be used concurrently and the caller should acquire the
// EndpointLookupsCache.epMutex before calling this method.
func (ec *EndpointLookupsCache) removeEndpointIpMapping(key model.Key, ip [16]byte) {
	// Remove existing IP to endpoint mapping.
	existingEps, ok := ec.ipToEndpoints[ip]
	if !ok || len(existingEps) == 1 {
		// There are no entries or only a single endpoint corresponding
		// to this IP address so it is safe to remove this mapping.
		delete(ec.ipToEndpoints, ip)
	} else {
		// If there is more than one endpoint, then keep the reverse ip
		// to endpoint mapping but only remove the endpoint corresponding
		// to this remove call.
		newEps := make([]*EndpointData, 0, len(existingEps)-1)
		for _, ep := range existingEps {
			if ep.Key == key {
				continue
			}
			newEps = append(newEps, ep)
		}
		ec.ipToEndpoints[ip] = newEps
	}

	// Remove endpoint to IP mapping.
	existingIps, ok := ec.endpointToIps[key]
	if !ok {
		return
	}
	if existingIps.Len() == 1 {
		// There is only a single ip corresponding to this
		// key so it is safe to remove this mapping.
		delete(ec.endpointToIps, key)
	} else {
		existingIps.Discard(ip)
		ec.endpointToIps[key] = existingIps
	}
}

// IsEndpoint returns true if the supplied address is a endpoint, otherwise returns false.
// Use the EndpointData.IsLocal() method to check if an EndpointData object (returned by the
// EndpointLookupsCache.GetEndpoint() method) is a local endpoint or not.
func (ec *EndpointLookupsCache) IsEndpoint(addr [16]byte) bool {
	_, ok := ec.GetEndpoint(addr)
	return ok
}

// GetEndpoint returns the ordered list of tiers for a particular endpoint.
func (ec *EndpointLookupsCache) GetEndpoint(addr [16]byte) (*EndpointData, bool) {
	ec.epMutex.RLock()
	defer ec.epMutex.RUnlock()
	eps, ok := ec.ipToEndpoints[addr]
	if len(eps) >= 1 {
		// We return the last observed endpoint.
		return eps[len(eps)-1], ok
	}
	return nil, ok
}

// endpointName is a convenience function to return a printable name for an endpoint.
func endpointName(key model.Key) (name string) {
	switch k := key.(type) {
	case model.WorkloadEndpointKey:
		name = workloadEndpointName(k)
	case model.HostEndpointKey:
		name = hostEndpointName(k)
	}
	return
}

// workloadEndpointName returns a single string rep of the workload endpoint.
func workloadEndpointName(wep model.WorkloadEndpointKey) string {
	return "WEP(" + wep.Hostname + "/" + wep.OrchestratorID + "/" + wep.WorkloadID + "/" + wep.EndpointID + ")"
}

// hostEndpointName returns a single string rep of the host endpoint.
func hostEndpointName(hep model.HostEndpointKey) string {
	return "HEP(" + hep.Hostname + "/" + hep.EndpointID + ")"
}

// extractIPsFromHostEndpoint converts the expected IPs of the host endpoint into [16]byte
func extractIPsFromHostEndpoint(endpoint *model.HostEndpoint) [][16]byte {
	v4Addrs := endpoint.ExpectedIPv4Addrs
	v6Addrs := endpoint.ExpectedIPv6Addrs
	combined := make([][16]byte, 0, len(v4Addrs)+len(v6Addrs))
	for _, addr := range v4Addrs {
		var addrB [16]byte
		copy(addrB[:], addr.IP.To16()[:16])
		combined = append(combined, addrB)
	}
	for _, addr := range v6Addrs {
		var addrB [16]byte
		copy(addrB[:], addr.IP.To16()[:16])
		combined = append(combined, addrB)
	}
	return combined
}

// extractIPsFromWorkloadEndpoint converts the IPv[46]Nets fields of the WorkloadEndpoint into
// [16]bytes. It ignores any prefix length.
func extractIPsFromWorkloadEndpoint(endpoint *model.WorkloadEndpoint) [][16]byte {
	v4Nets := endpoint.IPv4Nets
	v6Nets := endpoint.IPv6Nets
	combined := make([][16]byte, 0, len(v4Nets)+len(v6Nets))
	for _, addr := range v4Nets {
		var addrB [16]byte
		copy(addrB[:], addr.IP.To16()[:16])
		combined = append(combined, addrB)
	}
	for _, addr := range v6Nets {
		var addrB [16]byte
		copy(addrB[:], addr.IP.To16()[:16])
		combined = append(combined, addrB)
	}
	return combined
}

func (ec *EndpointLookupsCache) DumpEndpoints() string {
	ec.epMutex.RLock()
	defer ec.epMutex.RUnlock()
	lines := []string{}
	for ip, eds := range ec.ipToEndpoints {
		edNames := []string{}
		for _, ed := range eds {
			edNames = append(edNames, endpointName(ed.Key))
		}
		lines = append(lines, net.IP(ip[:16]).String()+": "+strings.Join(edNames, ","))
	}
	lines = append(lines, "-------")
	for key, ips := range ec.endpointToIps {
		ipStr := []string{}
		ips.Iter(func(item interface{}) error {
			ip := item.([16]byte)
			ipStr = append(ipStr, net.IP(ip[:16]).String())
			return nil
		})
		lines = append(lines, endpointName(key), ": ", strings.Join(ipStr, ","))
	}
	return strings.Join(lines, "\n")
}
