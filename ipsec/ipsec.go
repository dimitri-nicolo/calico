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

const (
	anyAddress = ""
	reqID      = 50 // Used to correlate between the policy and state tables.
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
		config:           NewCharonConfig(charonConfigRootDir, charonMainConfigFile),
	}

	// Initialise charon main config file.
	d.config.SetLogLevel(logLevel)
	d.config.SetBooleanOption(CharonFollowRedirects, false)
	d.config.SetBooleanOption(CharonMakeBeforeBreak, true)
	d.config.RenderToFile()
	log.Infof("Initialising charon config %+v", d.config)

	ikeDaemon, err := NewCharonIKEDaemon(context.Background(), &d.wg, espProposal)
	if err != nil {
		panic(fmt.Errorf("error creating CharonIKEDaemon struct: %v", err))
	}
	d.ikeDaemon = ikeDaemon

	// FIXME The following LoadSharedKey call fails if we don't wait for the Charon to start first.
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

func (d *Dataplane) AddBlacklist(workloadAddress string) {
	log.Warning("Adding IPsec blacklist", workloadAddress)
	AddBlock(workloadAddress+"/32", netlink.XFRM_DIR_FWD)
	AddBlock(workloadAddress+"/32", netlink.XFRM_DIR_OUT)
}

func (d *Dataplane) RemoveBlacklist(workloadAddress string) {
	log.Warning("Removing IPsec blacklist", workloadAddress)
	RemoveBlock(workloadAddress+"/32", netlink.XFRM_DIR_FWD)
	RemoveBlock(workloadAddress+"/32", netlink.XFRM_DIR_OUT)
}

func (d *Dataplane) AddBinding(remoteTunnelAddr, workloadAddress string) {
	log.Debug("Adding IPsec binding", workloadAddress, "via tunnel", remoteTunnelAddr)
	if _, ok := d.bindingsByTunnel[remoteTunnelAddr]; !ok {
		d.configureTunnel(remoteTunnelAddr)
		d.bindingsByTunnel[remoteTunnelAddr] = set.New()

		if remoteTunnelAddr != d.localTunnelAddr {
			// Allow the remote host to send encrypted traffic to our local workloads.  This balances the OUT rule
			// that will get programmed on the remote host in order to send traffic to our workloads.
			panicIfErr(AddXFRMPolicy(remoteTunnelAddr+"/32", "", remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, reqID))
			// Allow iptables to selectively encrypt packets to the host itself.  This allows us to encrypt traffic
			// from local workloads to the remote host.
			panicIfErr(AddXFRMPolicy(
				"0.0.0.0/0", remoteTunnelAddr+"/32",
				d.localTunnelAddr, remoteTunnelAddr,
				netlink.XFRM_DIR_OUT, reqID,
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
			panicIfErr(DeleteXFRMPolicy(remoteTunnelAddr+"/32", "", remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, reqID))
			panicIfErr(DeleteXFRMPolicy(
				"0.0.0.0/0", remoteTunnelAddr+"/32",
				d.localTunnelAddr, remoteTunnelAddr,
				netlink.XFRM_DIR_OUT, reqID,
				&netlink.XfrmMark{Value: d.forwardMark, Mask: d.forwardMark}))
		}

		d.removeTunnel(remoteTunnelAddr)
		delete(d.bindingsByTunnel, remoteTunnelAddr)
	}
}

func (d *Dataplane) configureXfrm(remoteTunnelAddr, workloadAddr string) {
	if remoteTunnelAddr == d.localTunnelAddr {
		return
	}
	log.Debugf("Adding IPsec policy: %s %s %s %s", anyAddress, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
	workloadAddr += "/32"
	// Remote workload to local workload traffic, hits the FWD xfrm policy.
	panicIfErr(AddXFRMPolicy(workloadAddr, anyAddress, remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, reqID))
	// Remote workload to local host, hits the IN xfrm policy.
	panicIfErr(AddXFRMPolicy(workloadAddr, d.localTunnelAddr+"/32", remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_IN, reqID))
	// Local traffic to remote workload.
	panicIfErr(AddXFRMPolicy(anyAddress, workloadAddr, d.localTunnelAddr, remoteTunnelAddr, netlink.XFRM_DIR_OUT, reqID))
	log.Debugf("Added IPsec policy: %s %s %s %s", anyAddress, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
}

func panicIfErr(err error) {
	if err == nil {
		return
	}
	log.WithError(err).Panic("IPsec operation failed")
}

func (d *Dataplane) removeXfrm(remoteTunnelAddr, workloadAddr string) {
	if remoteTunnelAddr == d.localTunnelAddr {
		return
	}
	log.Debugf("Removing IPsec policy: %s %s %s %s", anyAddress, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
	workloadAddr += "/32"
	panicIfErr(DeleteXFRMPolicy(workloadAddr, anyAddress, remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, reqID))
	panicIfErr(DeleteXFRMPolicy(workloadAddr, d.localTunnelAddr+"/32", remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_IN, reqID))
	panicIfErr(DeleteXFRMPolicy(anyAddress, workloadAddr, d.localTunnelAddr, remoteTunnelAddr, netlink.XFRM_DIR_OUT, reqID))
	log.Debugf("Removing IPsec policy: %s %s %s %s", anyAddress, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
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
