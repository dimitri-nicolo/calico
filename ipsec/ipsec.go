// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package ipsec

import (
	"context"
	"fmt"
	"sync"

	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/libcalico-go/lib/set"
)

const (
	anyAddress = ""
	ReqID      = 50 // Used to correlate between the policy and state tables.
)

func NewDataplane(localTunnelAddr string, preSharedKey, ikeProposal, espProposal, logLevel string, forwardMark uint32, polTable *PolicyTable) *Dataplane { // Start the charon

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
		polTable:         polTable,
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

	polTable *PolicyTable
}

func (d *Dataplane) AddBlacklist(workloadAddress string) {
	log.Warningf("Adding IPsec blacklist for %v", workloadAddress)

	cidr := ip.FromString(workloadAddress).AsCIDR().(ip.V4CIDR)
	fwdSel := PolicySelector{
		TrafficDst: cidr,
		Dir:        netlink.XFRM_DIR_FWD,
	}
	outSel := PolicySelector{
		TrafficDst: cidr,
		Dir:        netlink.XFRM_DIR_OUT,
	}

	d.polTable.SetRule(fwdSel, &blockRule)
	d.polTable.SetRule(outSel, &blockRule)
}

func (d *Dataplane) RemoveBlacklist(workloadAddress string) {
	log.Warningf("Removing IPsec blacklist for %v", workloadAddress)
	cidr := stringToV4CIDR(workloadAddress)
	fwdSel := PolicySelector{
		TrafficDst: cidr,
		Dir:        netlink.XFRM_DIR_FWD,
	}
	outSel := PolicySelector{
		TrafficDst: cidr,
		Dir:        netlink.XFRM_DIR_OUT,
	}

	d.polTable.DeleteRule(fwdSel)
	d.polTable.DeleteRule(outSel)
}

func stringToV4CIDR(addr string) (cidr ip.V4CIDR) {
	return ip.FromString(addr).AsCIDR().(ip.V4CIDR)
}

func stringToV4IP(addr string) ip.V4Addr {
	return ip.FromString(addr).(ip.V4Addr)
}

func (d *Dataplane) AddBinding(remoteTunnelAddr, workloadAddress string) {
	log.Debug("Adding IPsec binding ", workloadAddress, " via tunnel ", remoteTunnelAddr)
	if _, ok := d.bindingsByTunnel[remoteTunnelAddr]; !ok {
		d.configureTunnel(remoteTunnelAddr)
		d.bindingsByTunnel[remoteTunnelAddr] = set.New()

		if remoteTunnelAddr != d.localTunnelAddr {
			// Allow the remote host to send encrypted traffic to our local workloads.  This balances the OUT rule
			// that will get programmed on the remote host in order to send traffic to our workloads.
			d.polTable.SetRule(PolicySelector{
				TrafficSrc: stringToV4CIDR(remoteTunnelAddr),
				Dir:        netlink.XFRM_DIR_FWD,
			}, &PolicyRule{
				TunnelSrc: stringToV4IP(remoteTunnelAddr),
				TunnelDst: stringToV4IP(d.localTunnelAddr),
			})

			// Allow iptables to selectively encrypt packets to the host itself.  This allows us to encrypt traffic
			// from local workloads to the remote host.
			d.polTable.SetRule(PolicySelector{
				TrafficDst: stringToV4CIDR(remoteTunnelAddr),
				Dir:        netlink.XFRM_DIR_OUT,
				Mark:       d.forwardMark,
				MarkMask:   d.forwardMark,
			}, &PolicyRule{
				TunnelSrc: stringToV4IP(d.localTunnelAddr),
				TunnelDst: stringToV4IP(remoteTunnelAddr),
			})
		}
	}
	d.bindingsByTunnel[remoteTunnelAddr].Add(workloadAddress)
	d.configureXfrm(remoteTunnelAddr, workloadAddress)
}

func (d *Dataplane) RemoveBinding(remoteTunnelAddr, workloadAddress string) {
	log.Debug("Removing IPsec binding ", workloadAddress, " via tunnel ", remoteTunnelAddr)
	d.removeXfrm(remoteTunnelAddr, workloadAddress)
	d.bindingsByTunnel[remoteTunnelAddr].Discard(workloadAddress)
	if d.bindingsByTunnel[remoteTunnelAddr].Len() == 0 {
		if remoteTunnelAddr != d.localTunnelAddr {
			d.polTable.DeleteRule(PolicySelector{
				TrafficSrc: stringToV4CIDR(remoteTunnelAddr),
				Dir:        netlink.XFRM_DIR_FWD,
			})
			d.polTable.DeleteRule(PolicySelector{
				TrafficDst: stringToV4CIDR(remoteTunnelAddr),
				Dir:        netlink.XFRM_DIR_OUT,
			})
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
	// Remote workload to local workload traffic, hits the FWD xfrm policy.
	d.polTable.SetRule(PolicySelector{
		TrafficSrc: stringToV4CIDR(workloadAddr),
		Dir:        netlink.XFRM_DIR_FWD,
	}, &PolicyRule{
		TunnelSrc: stringToV4IP(remoteTunnelAddr),
		TunnelDst: stringToV4IP(d.localTunnelAddr),
	})
	// Remote workload to local host, hits the IN xfrm policy.
	d.polTable.SetRule(PolicySelector{
		TrafficSrc: stringToV4CIDR(workloadAddr),
		TrafficDst: stringToV4CIDR(d.localTunnelAddr),
		Dir:        netlink.XFRM_DIR_IN,
	}, &PolicyRule{
		TunnelSrc: stringToV4IP(remoteTunnelAddr),
		TunnelDst: stringToV4IP(d.localTunnelAddr),
	})
	// Local traffic to remote workload.
	d.polTable.SetRule(PolicySelector{
		TrafficDst: stringToV4CIDR(workloadAddr),
		Dir:        netlink.XFRM_DIR_OUT,
	}, &PolicyRule{
		TunnelSrc: stringToV4IP(d.localTunnelAddr),
		TunnelDst: stringToV4IP(remoteTunnelAddr),
	})
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
	// Remote workload to local workload traffic, hits the FWD xfrm policy.
	d.polTable.DeleteRule(PolicySelector{
		TrafficSrc: stringToV4CIDR(workloadAddr),
		Dir:        netlink.XFRM_DIR_FWD,
	})
	// Remote workload to local host, hits the IN xfrm policy.
	d.polTable.DeleteRule(PolicySelector{
		TrafficSrc: stringToV4CIDR(workloadAddr),
		TrafficDst: stringToV4CIDR(d.localTunnelAddr),
		Dir:        netlink.XFRM_DIR_IN,
	})
	// Local traffic to remote workload.
	d.polTable.DeleteRule(PolicySelector{
		TrafficDst: stringToV4CIDR(workloadAddr),
		Dir:        netlink.XFRM_DIR_OUT,
	})
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
