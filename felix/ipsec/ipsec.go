// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package ipsec

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

const (
	anyAddress = ""
	ReqID      = 0xca11c0 // Used to correlate between the policy and state tables.
)

type polTable interface {
	SetRule(sel PolicySelector, rule *PolicyRule)
	DeleteRule(sel PolicySelector)
}

type ikeDaemon interface {
	LoadSharedKey(remoteIP, password string) error
	UnloadSharedKey(remoteIP string) error
	LoadConnection(localIP, remoteIP string) error
	UnloadCharonConnection(localIP, remoteIP string) error
}

func NewDataplane(
	localTunnelAddr net.IP,
	preSharedKey string,
	forwardMark uint32,
	polTable polTable,
	ikeDaemon ikeDaemon,
	allowUnsecuredTraffic bool,
) *Dataplane {
	return NewDataplaneWithShims(localTunnelAddr, preSharedKey, forwardMark, polTable, ikeDaemon, allowUnsecuredTraffic, time.Sleep)
}

func NewDataplaneWithShims(
	localTunnelAddr net.IP,
	preSharedKey string,
	forwardMark uint32,
	polTable polTable,
	ikeDaemon ikeDaemon,
	allowUnsecuredTraffic bool,
	sleep func(duration time.Duration),
) *Dataplane {
	if forwardMark == 0 {
		log.Panic("IPsec forward mark shouldn't be 0")
	}

	d := &Dataplane{
		preSharedKey:          preSharedKey,
		localTunnelAddr:       localTunnelAddr.String(),
		forwardMark:           forwardMark,
		allowUnsecuredTraffic: allowUnsecuredTraffic,

		bindingsByTunnel: map[string]set.Set[string]{},

		polTable:  polTable,
		ikeDaemon: ikeDaemon,
		sleep:     sleep,
	}

	// Load the shared key into the charon for our end of the tunnels.
	attempts := 10
	for {
		err := d.ikeDaemon.LoadSharedKey(d.localTunnelAddr, preSharedKey)
		if err != nil {
			log.WithError(err).Info("Failed to load our shared key into the Charon")
			attempts--
			if attempts == 0 {
				log.WithError(err).Panic("Failed to load our shared key into the Charon after retries")
			}
			sleep(time.Second)
			continue
		}
		break
	}

	return d
}

type Dataplane struct {
	preSharedKey          string
	forwardMark           uint32
	localTunnelAddr       string
	allowUnsecuredTraffic bool

	bindingsByTunnel map[string]set.Set[string]

	ikeDaemon ikeDaemon
	polTable  polTable

	sleep func(duration time.Duration)
}

func (d *Dataplane) AddTunnel(remoteTunnelAddr string) {
	log.Infof("Adding IPsec tunnel to %v", remoteTunnelAddr)
	if d.bindingsByTunnel[remoteTunnelAddr] != nil {
		log.WithField("addr", remoteTunnelAddr).Panic("IPsec tunnel already exists")
	}

	d.configureTunnel(remoteTunnelAddr)
	d.bindingsByTunnel[remoteTunnelAddr] = set.New[string]()

	if remoteTunnelAddr != d.localTunnelAddr {
		// Allow the remote host to send encrypted traffic to our local workloads.  This balances the OUT rule
		// that will get programmed on the remote host in order to send traffic to our workloads.
		d.polTable.SetRule(PolicySelector{
			TrafficSrc: stringToV4CIDR(remoteTunnelAddr),
			Dir:        netlink.XFRM_DIR_FWD,
		}, &PolicyRule{
			TunnelSrc: stringToV4IP(remoteTunnelAddr),
			TunnelDst: stringToV4IP(d.localTunnelAddr),
			Optional:  d.allowUnsecuredTraffic,
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
			Optional:  d.allowUnsecuredTraffic,
		})
	}
}

func (d *Dataplane) RemoveTunnel(remoteTunnelAddr string) {
	log.Infof("Removing IPsec tunnel to %v", remoteTunnelAddr)
	if d.bindingsByTunnel[remoteTunnelAddr].Len() != 0 {
		log.WithField("tunnelAddr", remoteTunnelAddr).Panic("IPsec tunnel deleted while in use")
	}

	delete(d.bindingsByTunnel, remoteTunnelAddr)
	if remoteTunnelAddr != d.localTunnelAddr {
		d.polTable.DeleteRule(PolicySelector{
			TrafficSrc: stringToV4CIDR(remoteTunnelAddr),
			Dir:        netlink.XFRM_DIR_FWD,
		})
		d.polTable.DeleteRule(PolicySelector{
			TrafficDst: stringToV4CIDR(remoteTunnelAddr),
			Dir:        netlink.XFRM_DIR_OUT,
			Mark:       d.forwardMark,
			MarkMask:   d.forwardMark,
		})
	}
	d.removeTunnel(remoteTunnelAddr)
}

func (d *Dataplane) AddBlacklist(workloadAddress string) {
	if d.allowUnsecuredTraffic {
		log.Debug("Unsecured IPsec traffic allowed, not populating the blacklist")
		return
	}
	log.Warningf("Adding IPsec blacklist for %v", workloadAddress)

	cidr := ip.FromString(workloadAddress).AsCIDR().(ip.V4CIDR)

	d.polTable.SetRule(PolicySelector{
		TrafficSrc: cidr,
		Dir:        netlink.XFRM_DIR_IN,
	}, &blockRule)
	d.polTable.SetRule(PolicySelector{
		TrafficSrc: cidr,
		Dir:        netlink.XFRM_DIR_FWD,
	}, &blockRule)
	d.polTable.SetRule(PolicySelector{
		TrafficDst: cidr,
		Dir:        netlink.XFRM_DIR_OUT,
	}, &blockRule)
	d.polTable.SetRule(PolicySelector{
		TrafficDst: cidr,
		Dir:        netlink.XFRM_DIR_FWD,
	}, &blockRule)
}

func (d *Dataplane) RemoveBlacklist(workloadAddress string) {
	if d.allowUnsecuredTraffic {
		return
	}
	log.Warningf("Removing IPsec blacklist for %v", workloadAddress)
	cidr := stringToV4CIDR(workloadAddress)

	d.polTable.DeleteRule(PolicySelector{
		TrafficSrc: cidr,
		Dir:        netlink.XFRM_DIR_IN,
	})
	d.polTable.DeleteRule(PolicySelector{
		TrafficSrc: cidr,
		Dir:        netlink.XFRM_DIR_FWD,
	})
	d.polTable.DeleteRule(PolicySelector{
		TrafficDst: cidr,
		Dir:        netlink.XFRM_DIR_OUT,
	})
	d.polTable.DeleteRule(PolicySelector{
		TrafficDst: cidr,
		Dir:        netlink.XFRM_DIR_FWD,
	})
}

func stringToV4CIDR(addr string) (cidr ip.V4CIDR) {
	return ip.FromString(addr).AsCIDR().(ip.V4CIDR)
}

func stringToV4IP(addr string) ip.V4Addr {
	return ip.FromString(addr).(ip.V4Addr)
}

func (d *Dataplane) AddBinding(remoteTunnelAddr, workloadAddress string) {
	log.Debug("Adding IPsec binding ", workloadAddress, " via tunnel ", remoteTunnelAddr)
	d.bindingsByTunnel[remoteTunnelAddr].Add(workloadAddress)
	d.addXfrm(remoteTunnelAddr, workloadAddress)
}

func (d *Dataplane) RemoveBinding(remoteTunnelAddr, workloadAddress string) {
	log.Debug("Removing IPsec binding ", workloadAddress, " via tunnel ", remoteTunnelAddr)
	d.removeXfrm(remoteTunnelAddr, workloadAddress)
	d.bindingsByTunnel[remoteTunnelAddr].Discard(workloadAddress)
}

func (d *Dataplane) addXfrm(remoteTunnelAddr, workloadAddr string) {
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
		Optional:  d.allowUnsecuredTraffic,
	})
	// Remote workload to local host, hits the IN xfrm policy.
	d.polTable.SetRule(PolicySelector{
		TrafficSrc: stringToV4CIDR(workloadAddr),
		TrafficDst: stringToV4CIDR(d.localTunnelAddr),
		Dir:        netlink.XFRM_DIR_IN,
	}, &PolicyRule{
		TunnelSrc: stringToV4IP(remoteTunnelAddr),
		TunnelDst: stringToV4IP(d.localTunnelAddr),
		Optional:  d.allowUnsecuredTraffic,
	})
	// Local traffic to remote workload.
	d.polTable.SetRule(PolicySelector{
		TrafficDst: stringToV4CIDR(workloadAddr),
		Dir:        netlink.XFRM_DIR_OUT,
	}, &PolicyRule{
		TunnelSrc: stringToV4IP(d.localTunnelAddr),
		TunnelDst: stringToV4IP(remoteTunnelAddr),
		Optional:  d.allowUnsecuredTraffic,
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
	panicIfErr(d.ikeDaemon.UnloadSharedKey(tunnelAddr))
}
