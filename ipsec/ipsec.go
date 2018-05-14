// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package ipsec

import (
	"context"
	"fmt"
	"sync"

	"time"

	"github.com/projectcalico/libcalico-go/lib/set"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func NewDataplane(localTunnelAddr string, preSharedKey string) *Dataplane { // Start the charon
	d := &Dataplane{
		preSharedKey:     preSharedKey,
		localTunnelAddr:  localTunnelAddr,
		bindingsByTunnel: map[string]set.Set{},
	}

	ikeDaemon, err := NewCharonIKEDaemon(context.TODO(), &d.wg)
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
	ikeDaemon    *CharonIKEDaemon

	localTunnelAddr  string
	bindingsByTunnel map[string]set.Set

	wg sync.WaitGroup
}

func (d *Dataplane) AddBinding(tunnelAddress, workloadAddress string) {
	log.Debug("Adding IPsec binding", workloadAddress, "via tunnel", tunnelAddress)
	if _, ok := d.bindingsByTunnel[tunnelAddress]; !ok {
		d.configureTunnel(tunnelAddress)
		d.bindingsByTunnel[tunnelAddress] = set.New()
	}
	d.bindingsByTunnel[tunnelAddress].Add(workloadAddress)
	d.configureXfrm(tunnelAddress, workloadAddress)
}

func (d *Dataplane) RemoveBinding(tunnelAddress, workloadAddress string) {
	log.Debug("Removing IPsec binding", workloadAddress, "via tunnel", tunnelAddress)
	d.removeXfrm(tunnelAddress, workloadAddress)
	d.bindingsByTunnel[tunnelAddress].Discard(workloadAddress)
	if d.bindingsByTunnel[tunnelAddress].Len() == 0 {
		d.removeTunnel(tunnelAddress)
		delete(d.bindingsByTunnel, tunnelAddress)
	}
}

func (d *Dataplane) configureXfrm(remoteTunnelAddr, workloadAddr string) {
	if remoteTunnelAddr == d.localTunnelAddr {
		log.Debug("Skipping IPsec for local workload")
		return
	}

	reqID := 50        // FIXME: needs to be unique per workload?
	any := "0.0.0.0/0" // From any IP - this means host to pod and pod to host traffic will be encrypted too (hopefully)
	log.Debug("Adding IPsec policy: %s %s %s %s - ", any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
	workloadAddr += "/32"
	panicIfErr(AddXFRMPolicy(any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr, netlink.XFRM_DIR_OUT, reqID))
	panicIfErr(AddXFRMPolicy(workloadAddr, any, remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_IN, reqID))
	panicIfErr(AddXFRMPolicy(workloadAddr, any, remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, reqID))
	log.Debug("Added IPsec policy: %s %s %s %s - ", any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
}

func panicIfErr(err error) {
	if err == nil {
		return
	}
	log.WithError(err).Panic("IPsec operation failed")
}

func (d *Dataplane) removeXfrm(remoteTunnelAddr, workloadAddr string) {
	if remoteTunnelAddr == d.localTunnelAddr {
		log.Debug("Skipping IPsec for local workload")
		return
	}

	reqID := 50        // FIXME: needs to be unique per workload?
	any := "0.0.0.0/0" // From any IP - this means host to pod and pod to host traffic will be encrypted too (hopefully)
	log.Debug("Removing IPsec policy: %s %s %s %s - ", any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
	workloadAddr += "/32"
	panicIfErr(DeleteXFRMPolicy(any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr, netlink.XFRM_DIR_OUT, reqID))
	panicIfErr(DeleteXFRMPolicy(workloadAddr, any, remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_IN, reqID))
	panicIfErr(DeleteXFRMPolicy(workloadAddr, any, remoteTunnelAddr, d.localTunnelAddr, netlink.XFRM_DIR_FWD, reqID))
	log.Debug("Removing IPsec policy: %s %s %s %s - ", any, workloadAddr, d.localTunnelAddr, remoteTunnelAddr)
}

func (d *Dataplane) configureTunnel(tunnelAddr string) {
	if tunnelAddr == d.localTunnelAddr {
		log.Debug("Skipping IPsec for local tunnel")
		return
	}
	panicIfErr(d.ikeDaemon.LoadSharedKey(tunnelAddr, d.preSharedKey))
	panicIfErr(d.ikeDaemon.LoadConnection(d.localTunnelAddr, tunnelAddr))
}

func (d *Dataplane) removeTunnel(tunnelAddr string) {
	if tunnelAddr == d.localTunnelAddr {
		log.Debug("Skipping IPsec for local tunnel")
		return
	}
	panicIfErr(d.ikeDaemon.UnloadCharonConnection(d.localTunnelAddr, tunnelAddr))
}
