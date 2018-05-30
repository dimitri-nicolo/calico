// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package calc

import (
	log "github.com/sirupsen/logrus"

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
// In order to make the calculation robust against races and misconfiguration, we need to deal with:
//
// - host IPs being missing (these are populated asynchronously by the kubelet, for example)
// - host IPs being duplicated (where one node is being deleted async, while its IP is being reused)
// - host IPs being deleted before/after workload IPs (where felix's general contract applies: it should apply
//   the same policy given the same state of the datastore, no matter what path was taken to get there)
// - workload IPs being reused (and so transiently appearing on multiple workload endpoints on a host)
// - incorrect data created by a user (which may only be resolved much later when they spot their mistake).
//
// In particular, we need to do something safe while the misconfiguration is in place and then we need to
// correct the state when the misconfiguration is removed.
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
	allUpdDispatcher.Register(model.HostIPKey{}, c.OnHostIPUpdate)
	allUpdDispatcher.Register(model.WorkloadEndpointKey{}, c.OnEndpointUpdate)
}

func (c *IPSecBindingCalculator) OnHostIPUpdate(update api.Update) (_ bool) {
	hostIPKey := update.Key.(model.HostIPKey)
	nodeName := hostIPKey.Hostname
	var oldIP = c.nodeNameToAddress[nodeName]
	var newIP ip.Addr
	logCxt := log.WithField("host", hostIPKey.Hostname)
	if update.Value != nil {
		logCxt = logCxt.WithField("newIP", update.Value)
		logCxt.Debug("Updating IP for host")
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
			logCxt.Debug("IP was unique before, removing bindings")
			c.removeBindingsForNode(nodeName, oldIP)
			delete(c.addressToNodeNames, oldIP)
		} else if numHostsSharingIP > 1 {
			// Remove the node from the index first, so we can easily find the other node that shares the IP below.
			c.addressToNodeNames[oldIP] = filterOutString(c.addressToNodeNames[oldIP], nodeName)
			if numHostsSharingIP == 2 {
				// Removing the old IP means that the IP is now uniquely owned by a single other node, we now need to emit
				// bindings for that node.
				logCxt.Debug("Old IP is now unique, emitting bindings")
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
			logCxt.Debug("New IP is unique, emitting bindings")
			c.emitBindingsForNode(nodeName, newIP)
		} else if numHostsSharingIP == 1 {
			// IP previously belonged solely to another node but now it's ambiguous, need to remove the bindings that
			// were associated with the old node.
			logCxt.Warn("New IP was previously owned by another node but now it's shared, removing bindings")
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
	logCxt := log.WithFields(log.Fields{
		"oldIPs": oldIPs,
		"newIPs": newIPs,
	})
	logCxt.Debug("Updating endpoint IPs")

	// Look up its node and IP and do a reverse lookup to check whether the node has a unique IP.
	node := wepKey.Hostname
	nodeIP := c.nodeNameToAddress[node]
	nodesWithThatIP := c.addressToNodeNames[nodeIP]

	removedIPs := set.FromArray(oldIPs)
	addedIPs := set.New()
	for _, addr := range newIPs {
		if removedIPs.Contains(addr) {
			removedIPs.Discard(addr)
		} else {
			addedIPs.Add(addr)
		}
	}

	c.endpointKeysToIPs[wepKey] = newIPs

	removedIPs.Iter(func(item interface{}) error {
		addr := item.(ip.Addr)
		// Remove old reverse index.
		c.removeIPToKey(addr, wepKey)
		// Now check what that leaves behind.
		numWepsStillSharingIP := len(c.ipToEndpointKeys[addr])
		if numWepsStillSharingIP > 1 {
			// IP wasn't unique before and it's still not unique.  We won't have emitted any bindings for it before.
			log.WithField("ip", addr).Warn("Workload IP is not unique, unable to do IPsec to IP.")
			return nil
		}
		if numWepsStillSharingIP == 1 {
			// Must have been 2 workloads sharing that IP before but now there's only one.  We need to look up the
			// other workload and see if its binding is now unambiguous.
			otherWepKey := c.ipToEndpointKeys[addr][0]
			otherNode := otherWepKey.Hostname
			otherNodesIP := c.nodeNameToAddress[otherNode]
			if otherNodesIP == nil {
				log.WithField("node", otherNode).Warn(
					"Missing node IP, unable to do IPsec for workload on that node.")
				return nil
			}
			nodesWithOtherNodesIP := c.addressToNodeNames[otherNodesIP]
			if len(nodesWithOtherNodesIP) != 1 {
				log.WithFields(log.Fields{"ip": otherNodesIP, "nodes": nodesWithOtherNodesIP}).Warn(
					"Node's IP is not unique, unable to do IPsec for workloads on that node.")
				return nil
			}
			c.OnBindingAdded(IPSecBinding{TunnelAddr: otherNodesIP, WorkloadAddr: addr})
			return nil
		}

		// numWepsStillSharingIP == 0: there must have been exactly one workload with this IP. Check if its node was
		// unique.
		if len(nodesWithThatIP) != 1 {
			log.WithFields(log.Fields{"workloadIP": addr, "nodeIP": nodeIP}).Debug("Node IP wasn't unique.")
			return nil
		}

		// Removed IP was unique and its node IP was unique.  There should have been an active binding.
		c.OnBindingRemoved(IPSecBinding{TunnelAddr: nodeIP, WorkloadAddr: addr})

		return nil
	})

	addedIPs.Iter(func(item interface{}) error {
		addr := item.(ip.Addr)

		// Before we add the IP to the index, check who has that IP already.  If we're about to give the IP multiple
		// owners then we'll need to remove the old binding because it's no longer unique.
		numWepsWithThatIP := len(c.ipToEndpointKeys[addr])

		if numWepsWithThatIP == 1 {
			// The IP currently has a unique owner, check if their node has a unique address...
			otherWepKey := c.ipToEndpointKeys[addr][0]
			otherNode := otherWepKey.Hostname
			otherNodesIP := c.nodeNameToAddress[otherNode]
			if otherNodesIP != nil && len(c.addressToNodeNames[otherNodesIP]) == 1 {
				log.WithField("node", otherNode).Warn(
					"IP address now owned by multiple workloads, unable to do IPsec for that IP.")
				c.OnBindingRemoved(IPSecBinding{TunnelAddr: otherNodesIP, WorkloadAddr: addr})
			}
		}

		// Add the new IP to the index.
		c.addIPToKey(addr, wepKey)

		if numWepsWithThatIP != 0 {
			log.WithField("ip", addr).Warn(
				"Workload IP address not unique, unable to do IPsec for that IP.")
			return nil
		}

		if len(nodesWithThatIP) != 1 {
			log.WithFields(log.Fields{"workloadIP": addr, "nodeIP": nodeIP}).Debug(
				"Node IP wasn't unique. Unable to do IPsec for this workload.")
			return nil
		}

		// Added IP is unique, as is its node's IP, emit a binding.
		c.OnBindingAdded(IPSecBinding{TunnelAddr: nodeIP, WorkloadAddr: addr})

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
