// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package ipsec

import (
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

type ikeDaemon interface {
	LoadSharedKey(remoteIP, password string) error
	LoadConnection(localIP, remoteIP string) error
	UnloadCharonConnection(localIP, remoteIP string) error
}

type polTable interface {
	SetRule(sel PolicySelector, rule *PolicyRule)
	DeleteRule(sel PolicySelector)
}

func NewDataplane(
	localTunnelAddr string,
	preSharedKey string,
	forwardMark uint32,
	polTable polTable,
	ikeDaemon ikeDaemon,
) *Dataplane {
	if forwardMark == 0 {
		log.Panic("IPsec forward mark shouldn't be 0")
	}

	d := &Dataplane{
		preSharedKey:    preSharedKey,
		localTunnelAddr: localTunnelAddr,
		forwardMark:     forwardMark,

		bindingsByTunnel: map[string]set.Set{},

		polTable:  polTable,
		ikeDaemon: ikeDaemon,
	}

	// Load the shared key into the charon for our end of the tunnels.
	tries := 10
	for {
		err := d.ikeDaemon.LoadSharedKey(localTunnelAddr, preSharedKey)
		if err != nil {
			log.WithError(err).Info("Failed to load our shared key into the Charon")
			if tries == 0 {
				log.WithError(err).Panic("Failed to load our shared key into the Charon after retries")
			}
			tries--
			time.Sleep(time.Second)
			continue
		}
		break
	}

	return d
}

type Dataplane struct {
	preSharedKey    string
	forwardMark     uint32
	localTunnelAddr string

	bindingsByTunnel map[string]set.Set

	ikeDaemon ikeDaemon
	polTable  polTable
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
	panicIfErr(d.ikeDaemon.LoadConnection(d.localTunnelAddr, tunnelAddr))
}

func (d *Dataplane) removeTunnel(tunnelAddr string) {
	if tunnelAddr == d.localTunnelAddr {
		log.Debug("Skipping IPsec for local tunnel")
		return
	}
	panicIfErr(d.ikeDaemon.UnloadCharonConnection(d.localTunnelAddr, tunnelAddr))
}
