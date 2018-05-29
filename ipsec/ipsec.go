// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package ipsec

import (
	"context"
	"fmt"
	"sync"

	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/projectcalico/libcalico-go/lib/set"
)

func NewDataplane(localTunnelAddr string, preSharedKey, ikeProposal, espProposal, logLevel string, forwardMark uint32) *Dataplane { // Start the charon

	if forwardMark == 0 {
		log.Panic("IPsec forward mark is 0")
	}

	d := &Dataplane{
		preSharedKey:     preSharedKey,
		localTunnelAddr:  localTunnelAddr,
		bindingsByTunnel: map[string]set.Set{},
		ikeProposal:      ikeProposal,
		forwardMark:      forwardMark,
		config:           newCharonConfig(charonFelixConfigFile),
	}

	d.config.SetLogLevel(logLevel)
	d.config.RenderToFile()

	ikeDaemon, err := NewCharonIKEDaemon(context.TODO(), &d.wg, espProposal)
	if err != nil {
		panic(fmt.Errorf("error creating CharonIKEDaemon struct: %v", err))
	}
	d.ikeDaemon = ikeDaemon

	time.Sleep(1 * time.Second)
	err = ikeDaemon.LoadSharedKey(localTunnelAddr, preSharedKey)
	if err != nil {
		panic(err)
	}

	return d
}

type Dataplane struct {
	preSharedKey string
	ikeProposal  string
	ikeDaemon    *CharonIKEDaemon

	forwardMark      uint32
	localTunnelAddr  string
	bindingsByTunnel map[string]set.Set

	config *CharonConfig

	wg sync.WaitGroup
}

func (d *Dataplane) AddBinding(remoteTunnelAddr, workloadAddress string) {
	log.Debug("Adding IPsec binding", workloadAddress, "via tunnel", remoteTunnelAddr)
	if _, ok := d.bindingsByTunnel[remoteTunnelAddr]; !ok {
		d.configureTunnel(remoteTunnelAddr)
		d.bindingsByTunnel[remoteTunnelAddr] = set.New()

		if remoteTunnelAddr != d.localTunnelAddr {
			// Allow the remote host to send encrypted traffic to our local workloads.  This balances the OUT rule
			// that will get programmed on the remote host in order to send traffic to our workloads.
			panicIfErr(AddXFRMPolicy(remoteTunnelAddr+"/32", "", remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, 50))
			// Allow iptables to selectively encrypt packets to the host itself.  This allows us to encrypt traffic
			// from local workloads to the remote host.
			panicIfErr(AddXFRMPolicy(
				"0.0.0.0/0", remoteTunnelAddr+"/32",
				d.localTunnelAddr, remoteTunnelAddr,
				netlink.XFRM_DIR_OUT, 50,
				&netlink.XfrmMark{Value: d.forwardMark, Mask: d.forwardMark},
			))
		}
	}
	d.bindingsByTunnel[remoteTunnelAddr].Add(workloadAddress)
	d.configureXfrm(remoteTunnelAddr, workloadAddress)
}

func (d *Dataplane) RemoveBinding(remoteTunnelAddr, workloadAddress string) {
	log.Debug("Removing IPsec binding", workloadAddress, "via tunnel", remoteTunnelAddr)
	d.removeXfrm(remoteTunnelAddr, workloadAddress)
	d.bindingsByTunnel[remoteTunnelAddr].Discard(workloadAddress)
	if d.bindingsByTunnel[remoteTunnelAddr].Len() == 0 {
		if remoteTunnelAddr != d.localTunnelAddr {
			panicIfErr(DeleteXFRMPolicy(remoteTunnelAddr+"/32", "", remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, 50))
			panicIfErr(DeleteXFRMPolicy(
				"0.0.0.0/0", remoteTunnelAddr+"/32",
				d.localTunnelAddr, remoteTunnelAddr,
				netlink.XFRM_DIR_OUT, 50,
				&netlink.XfrmMark{Value: d.forwardMark, Mask: d.forwardMark}))
		}

		d.removeTunnel(remoteTunnelAddr)
		delete(d.bindingsByTunnel, remoteTunnelAddr)
	}
}

func (d *Dataplane) configureXfrm(remoteTunnelAddr, workloadAddr string) {
	any := "0.0.0.0/0" // From any IP - this means host to pod and pod to host traffic will be encrypted too (hopefully)
	reqID := 50
	if remoteTunnelAddr == d.localTunnelAddr {
		return
	}
	log.Debug("Adding IPsec policy: %s %s %s %s - ", any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
	workloadAddr += "/32"
	// Remote workload to local workload traffic, hits the FWD xfrm policy.
	panicIfErr(AddXFRMPolicy(workloadAddr, any, remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, reqID))
	// Remote workload to local host, hits the IN xfrm policy.
	panicIfErr(AddXFRMPolicy(workloadAddr, d.localTunnelAddr+"/32", remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_IN, reqID))
	// Local traffic to remote workload.
	panicIfErr(AddXFRMPolicy(any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr, netlink.XFRM_DIR_OUT, reqID))
	log.Debug("Added IPsec policy: %s %s %s %s - ", any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
}

func panicIfErr(err error) {
	if err == nil {
		return
	}
	log.WithError(err).Panic("IPsec operation failed")
}

func (d *Dataplane) removeXfrm(remoteTunnelAddr, workloadAddr string) {
	reqID := 50        // FIXME: does this need to be unique per tunnel?
	any := "0.0.0.0/0" // From any IP - this means host to pod and pod to host traffic will be encrypted too (hopefully)
	if remoteTunnelAddr == d.localTunnelAddr {
		return
	}
	log.Debug("Removing IPsec policy: %s %s %s %s - ", any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
	workloadAddr += "/32"
	panicIfErr(DeleteXFRMPolicy(workloadAddr, any, remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, reqID))
	panicIfErr(DeleteXFRMPolicy(workloadAddr, d.localTunnelAddr+"/32", remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_IN, reqID))
	panicIfErr(DeleteXFRMPolicy(any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr, netlink.XFRM_DIR_OUT, reqID))
	log.Debug("Removing IPsec policy: %s %s %s %s - ", any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
}

func (d *Dataplane) configureTunnel(tunnelAddr string) {
	if tunnelAddr == d.localTunnelAddr {
		log.Debug("Skipping IPsec for local tunnel")
		return
	}
	panicIfErr(d.ikeDaemon.LoadSharedKey(tunnelAddr, d.preSharedKey))
	panicIfErr(d.ikeDaemon.LoadConnection(d.localTunnelAddr, tunnelAddr, d.ikeProposal))
}

func (d *Dataplane) removeTunnel(tunnelAddr string) {
	if tunnelAddr == d.localTunnelAddr {
		log.Debug("Skipping IPsec for local tunnel")
		return
	}
	panicIfErr(d.ikeDaemon.UnloadCharonConnection(d.localTunnelAddr, tunnelAddr))
}

func (d *Dataplane) reloadConfig() {
	log.Infof("Reloading IPSec config with %v", d.config)
	d.config.RenderToFile()
	panicIfErr(d.ikeDaemon.ReloadConfig())
}
