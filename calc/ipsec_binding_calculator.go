// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package calc

import (
	"github.com/projectcalico/felix/dispatcher"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/set"
)

func NewIPSecBindingCalculator() *IPSecBindingCalculator {
	return &IPSecBindingCalculator{
		nodeNameToAddress:  map[string]ip.Addr{},
		addressToNodeNames: map[ip.Addr][]string{},

		ipToEndpointKeys:  map[ip.Addr][]model.WorkloadEndpointKey{},
		endpointKeysToIPs: map[model.WorkloadEndpointKey][]ip.Addr{},
	}
}

// IPSecBindingCalculator resolves the set of IPs behind each IPsec tunnel.  There is an IPsec tunnel
// to each host's IP, and the IPs behind its tunnel are all the workloads on that host.
//
// In order to make the calculation robust against misconfiguration, we need to deal with:
//
// - host IPs being missing (these are populated asynchronously by the kubelet, for example)
// - host IPs being duplicated (where one node is being deleted async, while its IP is being reused)
// - host IPs being deleted before/after workload IPs (where felix's general contract applies: it should apply
//   the same policy given the same state of the datastore, no matter what path was taken to get there)
// - workload IPs being reused (and so transiently appearing on multiple workload endpoints on a host).
type IPSecBindingCalculator struct {
	nodeNameToAddress  map[string]ip.Addr
	addressToNodeNames map[ip.Addr][]string

	ipToEndpointKeys  map[ip.Addr][]model.WorkloadEndpointKey
	endpointKeysToIPs map[model.WorkloadEndpointKey][]ip.Addr

	OnBindingAdded   func(b IPSecBinding)
	OnBindingRemoved func(b IPSecBinding)
}

type IPSecBinding struct {
	TunnelAddr, WorkloadAddr ip.Addr
}

func (c *IPSecBindingCalculator) RegisterWith(allUpdDispatcher *dispatcher.Dispatcher) {
	allUpdDispatcher.Register(model.WorkloadEndpointKey{}, c.OnEndpointUpdate)
	allUpdDispatcher.Register(model.HostIPKey{}, c.OnHostIPUpdate)
}

func (c *IPSecBindingCalculator) OnHostIPUpdate(update api.Update) (_ bool) {
	hostIPKey := update.Key.(model.HostIPKey)
	nodeName := hostIPKey.Hostname
	var oldIP = c.nodeNameToAddress[nodeName]
	var newIP ip.Addr
	if update.Value != nil {
		newIP = ip.FromNetIP(update.Value.(*net.IP).IP)
	}

	if oldIP == newIP {
		// No change. Ignore.
		return
	}

	// First remove the old IP.
	if oldIP != nil {
		if numHostsSharingIP := len(c.addressToNodeNames[oldIP]); numHostsSharingIP == 1 {
			// IP was uniquely owned by this host; there may have been bindings associated with it, clean those up.
			c.removeBindingsForNode(nodeName, oldIP)
			delete(c.addressToNodeNames, oldIP)
		} else if numHostsSharingIP > 1 {
			// Remove the node from the index first, so we can easily find the other node that shares the IP below.
			c.addressToNodeNames[oldIP] = filterOutString(c.addressToNodeNames[oldIP], nodeName)
			if numHostsSharingIP == 2 {
				// Removing the old IP means that the IP is now uniquely owned by a single other node, we now need to emit
				// bindings for that node.
				otherNode := c.addressToNodeNames[oldIP][0]
				c.emitBindingsForNode(otherNode, oldIP)
			}
		}
		// Fix up the index.
		delete(c.nodeNameToAddress, nodeName)
	}

	if newIP != nil {
		// Add the new IP.
		if numHostsSharingIP := len(c.addressToNodeNames[newIP]); numHostsSharingIP == 0 {
			// IP previously had no owner so IP is now uniquely owned by this host.  Check for any pre-existing
			// workloads that were waiting for this binding to show up.
			c.emitBindingsForNode(nodeName, newIP)
		} else if numHostsSharingIP == 1 {
			// IP previously belonged solely to another node but now it's ambiguous, need to remove the bindings that
			// were associated with the old node.
			otherNode := c.addressToNodeNames[newIP][0]
			c.removeBindingsForNode(otherNode, newIP)
		}
		// Put the new IP in the node IP indexes.
		c.nodeNameToAddress[nodeName] = newIP
		c.addressToNodeNames[newIP] = append(c.addressToNodeNames[newIP], nodeName)
	}
	return
}

func (c *IPSecBindingCalculator) emitBindingsForNode(nodeName string, nodeIP ip.Addr) {
	for wepKey, addrs := range c.endpointKeysToIPs {
		if wepKey.Hostname != nodeName {
			continue
		}

		for _, addr := range addrs {
			// Check the reverse index to verify that this binding is unique.
			numWepsWithThatIP := len(c.ipToEndpointKeys[addr])
			if numWepsWithThatIP != 1 {
				continue
			}
			// This is a unique binding, emit it.
			c.OnBindingAdded(IPSecBinding{WorkloadAddr: addr, TunnelAddr: nodeIP})
		}
	}
}

func (c *IPSecBindingCalculator) removeBindingsForNode(nodeName string, nodeIP ip.Addr) {
	for wepKey, addrs := range c.endpointKeysToIPs {
		if wepKey.Hostname != nodeName {
			continue
		}

		for _, addr := range addrs {
			// Check the reverse index to verify that this binding is unique.
			numWepsWithThatIP := len(c.ipToEndpointKeys[addr])
			if numWepsWithThatIP != 1 {
				continue
			}
			// This was a unique binding, remove it.
			c.OnBindingRemoved(IPSecBinding{WorkloadAddr: addr, TunnelAddr: nodeIP})
		}
	}
}

func (c *IPSecBindingCalculator) OnEndpointUpdate(update api.Update) (_ bool) {
	wepKey := update.Key.(model.WorkloadEndpointKey)

	// Look up the old (possibly nil) and new (possibly nil) IPs for this endpoint.
	oldIPs := c.endpointKeysToIPs[wepKey]
	var newIPs []ip.Addr
	if update.Value != nil {
		for _, addr := range update.Value.(*model.WorkloadEndpoint).IPv4Nets {
			felixAddr := ip.FromNetIP(addr.IP)
			newIPs = append(newIPs, felixAddr)
		}
	}

	// Look up its node and IP and do a reverse lookup to check whether the node has a unique IP.
	node := wepKey.Hostname
	nodeIP := c.nodeNameToAddress[node]
	nodesWithThatIP := c.addressToNodeNames[nodeIP]

	oldBindings := set.New()
	newBindings := set.New()

	for _, addr := range oldIPs {
		// Check the old reverse index to verify that this binding was unique.
		numWepsWithThatIP := len(c.ipToEndpointKeys[addr])

		// Remove old reverse index.  We'll add it back below if it's still present.
		c.removeIPToKey(addr, wepKey)

		if numWepsWithThatIP != 1 {
			// There was previously more than one endpoint with this IP so the binding was ambiguous, which means
			// that we'll have previously ignored it.
			continue
		}

		if len(nodesWithThatIP) == 1 {
			// Exactly one  node with that IP and exactly one workload, the binding was previously active.  Add it to
			// our working set.
			oldBindings.Add(IPSecBinding{TunnelAddr: nodeIP, WorkloadAddr: addr})
		}
	}

	for _, addr := range newIPs {
		// Add the new IP to the index so we can check if the new binding is unique or ambiguous.
		c.addIPToKey(addr, wepKey)

		// Check the new reverse index to verify that this binding is unique.
		numWepsWithThatIP := len(c.ipToEndpointKeys[addr])
		if numWepsWithThatIP != 1 {
			// More than one endpoint with this IP so the binding is ambiguous, ignore it.
			continue
		}

		if len(nodesWithThatIP) == 1 {
			// Binding is unique, check if it's really new...
			b := IPSecBinding{TunnelAddr: nodeIP, WorkloadAddr: addr}
			if oldBindings.Contains(b) {
				// Binding hasn't changed.
				oldBindings.Discard(b)
				continue
			}

			// Binding is genuinely new.
			newBindings.Add(b)
		}
	}

	c.endpointKeysToIPs[wepKey] = newIPs

	// oldBindings now contains only bindings that have been removed as part of this update.
	oldBindings.Iter(func(item interface{}) error {
		c.OnBindingRemoved(item.(IPSecBinding))
		return nil
	})
	// ...and new bindings contains only the new ones.
	newBindings.Iter(func(item interface{}) error {
		c.OnBindingAdded(item.(IPSecBinding))
		return nil
	})

	return
}

func (c *IPSecBindingCalculator) addIPToKey(addr ip.Addr, wepKey model.WorkloadEndpointKey) {
	c.ipToEndpointKeys[addr] = append(c.ipToEndpointKeys[addr], wepKey)
}

func (c *IPSecBindingCalculator) removeIPToKey(addr ip.Addr, wepKey model.WorkloadEndpointKey) {
	updatedWeps := filterOutWepKey(c.ipToEndpointKeys[addr], wepKey)
	if len(updatedWeps) > 0 {
		c.ipToEndpointKeys[addr] = updatedWeps
	} else {
		delete(c.ipToEndpointKeys, addr)
	}
}

func filterOutWepKey(a []model.WorkloadEndpointKey, toSkip model.WorkloadEndpointKey) []model.WorkloadEndpointKey {
	var filtered []model.WorkloadEndpointKey
	for _, k := range a {
		if k == toSkip {
			continue
		}
		filtered = append(filtered, k)
	}
	return filtered
}

func filterOutString(a []string, toSkip string) []string {
	var filtered []string
	for _, s := range a {
		if s == toSkip {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}
