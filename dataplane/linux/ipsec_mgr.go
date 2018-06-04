// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package intdataplane

import (
	log "github.com/sirupsen/logrus"

	"os/exec"

	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

func newIPSecManager(ipSecDataplane ipSecDataplane) *ipsecManager {
	// FIXME Until we have resync, need to flush the policy table on each restart.
	err := exec.Command("ip", "xfrm", "policy", "flush").Run()
	if err != nil {
		log.WithError(err).Panic("Failed to flush")
	}
	return &ipsecManager{
		dataplane: ipSecDataplane,
	}
}

type ipSecDataplane interface {
	AddBinding(tunnelAddress, workloadAddress string)
	RemoveBinding(tunnelAddress, workloadAddress string)
	AddBlacklist(workloadAddress string)
	RemoveBlacklist(workloadAddress string)
}

type ipsecManager struct {
	preSharedKey string
	dataplane    ipSecDataplane

	// activeHostnameToIP maps hostname to string IP address.
	activeHostnameToIP map[string]string
	dirtyHosts         set.Set
}

func (d *ipsecManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.IPSecBindingUpdate:
		log.WithFields(log.Fields{
			"tunnel_addr": msg.TunnelAddr,
			"num_added":   len(msg.AddedAddrs),
			"num_removed": len(msg.RemovedAddrs),
		}).Debug("IPSec bindings updated")
		if msg.TunnelAddr == "" {
			// Blacklist.
			for _, removed := range msg.RemovedAddrs {
				d.dataplane.RemoveBlacklist(removed)
			}
			for _, added := range msg.AddedAddrs {
				d.dataplane.AddBlacklist(added)
			}
		} else {
			for _, removed := range msg.RemovedAddrs {
				d.dataplane.RemoveBinding(msg.TunnelAddr, removed)
			}
			for _, added := range msg.AddedAddrs {
				d.dataplane.AddBinding(msg.TunnelAddr, added)
			}
		}
	}
}

func (d *ipsecManager) CompleteDeferredWork() error {
	return nil
}
