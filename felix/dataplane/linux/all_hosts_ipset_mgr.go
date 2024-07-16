// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.
package intdataplane

import (
	log "github.com/sirupsen/logrus"

	dpsets "github.com/projectcalico/calico/felix/dataplane/ipsets"
	"github.com/projectcalico/calico/felix/ipsets"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/rules"
)

// allHostsIpsetManager manages the all-hosts IP set, which is used by some rules in our static chains
// when IPIP or IPSec are enabled.  It doesn't actually program the rules, because they are part of the
// top-level static chains.
type allHostsIpsetManager struct {
	ipsetsDataplane dpsets.IPSetsDataplane

	// activeHostnameToIP maps hostname to string IP address.  We don't bother to parse into
	// net.IPs because we're going to pass them directly to the IPSet API.
	activeHostnameToIP map[string]string
	ipSetInSync        bool

	// Configured list of external node ip cidr's to be added to the ipset.
	externalNodeCIDRs []string

	// Config for creating/refreshing the IP set.
	ipSetMetadata ipsets.IPSetMetadata
}

func newAllHostsIpsetManager(ipsetsDataplane dpsets.IPSetsDataplane, maxIPSetSize int, externalNodeCIDRs []string) *allHostsIpsetManager {
	return &allHostsIpsetManager{
		ipsetsDataplane:    ipsetsDataplane,
		activeHostnameToIP: map[string]string{},
		externalNodeCIDRs:  externalNodeCIDRs,
		ipSetMetadata: ipsets.IPSetMetadata{
			MaxSize: maxIPSetSize,
			SetID:   rules.IPSetIDAllHostNets,
			Type:    ipsets.IPSetTypeHashNet,
		},
	}
}

func (d *allHostsIpsetManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.HostMetadataUpdate:
		log.WithField("hostanme", msg.Hostname).Debug("Host update/create")
		d.activeHostnameToIP[msg.Hostname] = msg.Ipv4Addr
		d.ipSetInSync = false
	case *proto.HostMetadataRemove:
		log.WithField("hostname", msg.Hostname).Debug("Host removed")
		delete(d.activeHostnameToIP, msg.Hostname)
		d.ipSetInSync = false
	}
}

func (m *allHostsIpsetManager) CompleteDeferredWork() error {
	if !m.ipSetInSync {
		// For simplicity (and on the assumption that host add/removes are rare) rewrite
		// the whole IP set whenever we get a change.  To replace this with delta handling
		// would require reference counting the IPs because it's possible for two hosts
		// to (at least transiently) share an IP.  That would add occupancy and make the
		// code more complex.
		log.Info("All-hosts IP set out-of sync, refreshing it.")
		members := make([]string, 0, len(m.activeHostnameToIP)+len(m.externalNodeCIDRs))
		for _, ip := range m.activeHostnameToIP {
			members = append(members, ip)
		}
		members = append(members, m.externalNodeCIDRs...)
		m.ipsetsDataplane.AddOrReplaceIPSet(m.ipSetMetadata, members)
		m.ipSetInSync = true
	}
	return nil
}
