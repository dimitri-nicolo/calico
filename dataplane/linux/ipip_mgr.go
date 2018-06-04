// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package intdataplane

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// ipipManager takes care of the configuration of the IPIP tunnel device.
type ipipManager struct {
	// Dataplane shim.
	dataplane ipipDataplane
}

func newIPIPManager() *ipipManager {
	return newIPIPManagerWithShim(realIPIPNetlink{})
}

func newIPIPManagerWithShim(dataplane ipipDataplane) *ipipManager {
	ipipMgr := &ipipManager{
		dataplane: dataplane,
	}
	return ipipMgr
}

// KeepIPIPDeviceInSync is a goroutine that configures the IPIP tunnel device, then periodically
// checks that it is still correctly configured.
func (d *ipipManager) KeepIPIPDeviceInSync(mtu int, address net.IP) {
	log.Info("IPIP thread started.")
	for {
		err := d.configureIPIPDevice(mtu, address)
		if err != nil {
			log.WithError(err).Warn("Failed configure IPIP tunnel device, retrying...")
			time.Sleep(1 * time.Second)
			continue
		}
		time.Sleep(10 * time.Second)
	}
}

// configureIPIPDevice ensures the IPIP tunnel device is up and configures correctly.
func (d *ipipManager) configureIPIPDevice(mtu int, address net.IP) error {
	logCxt := log.WithFields(log.Fields{
		"mtu":        mtu,
		"tunnelAddr": address,
	})
	logCxt.Debug("Configuring IPIP tunnel")
	link, err := d.dataplane.LinkByName("tunl0")
	if err != nil {
		log.WithError(err).Info("Failed to get IPIP tunnel device, assuming it isn't present")
		// We call out to "ip tunnel", which takes care of loading the kernel module if
		// needed.  The tunl0 device is actually created automatically by the kernel
		// module.
		err := d.dataplane.RunCmd("ip", "tunnel", "add", "tunl0", "mode", "ipip")
		if err != nil {
			log.WithError(err).Warning("Failed to add IPIP tunnel device")
			return err
		}
		link, err = d.dataplane.LinkByName("tunl0")
		if err != nil {
			log.WithError(err).Warning("Failed to get tunnel device")
			return err
		}
	}

	attrs := link.Attrs()
	oldMTU := attrs.MTU
	if oldMTU != mtu {
		logCxt.WithField("oldMTU", oldMTU).Info("Tunnel device MTU needs to be updated")
		if err := d.dataplane.LinkSetMTU(link, mtu); err != nil {
			log.WithError(err).Warn("Failed to set tunnel device MTU")
			return err
		}
		logCxt.Info("Updated tunnel MTU")
	}
	if attrs.Flags&net.FlagUp == 0 {
		logCxt.WithField("flags", attrs.Flags).Info("Tunnel wasn't admin up, enabling it")
		if err := d.dataplane.LinkSetUp(link); err != nil {
			log.WithError(err).Warn("Failed to set tunnel device up")
			return err
		}
		logCxt.Info("Set tunnel admin up")
	}

	if err := d.setLinkAddressV4("tunl0", address); err != nil {
		log.WithError(err).Warn("Failed to set tunnel device IP")
		return err
	}
	return nil
}

// setLinkAddressV4 updates the given link to set its local IP address.  It removes any other
// addresses.
func (d *ipipManager) setLinkAddressV4(linkName string, address net.IP) error {
	logCxt := log.WithFields(log.Fields{
		"link": linkName,
		"addr": address,
	})
	logCxt.Debug("Setting local IPv4 address on link.")
	link, err := d.dataplane.LinkByName(linkName)
	if err != nil {
		log.WithError(err).WithField("name", linkName).Warning("Failed to get device")
		return err
	}

	addrs, err := d.dataplane.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		log.WithError(err).Warn("Failed to list interface addresses")
		return err
	}

	found := false
	for _, oldAddr := range addrs {
		if address != nil && oldAddr.IP.Equal(address) {
			logCxt.Debug("Address already present.")
			found = true
			continue
		}
		logCxt.WithField("oldAddr", oldAddr).Info("Removing old address")
		if err := d.dataplane.AddrDel(link, &oldAddr); err != nil {
			log.WithError(err).Warn("Failed to delete address")
			return err
		}
	}

	if !found && address != nil {
		logCxt.Info("Address wasn't present, adding it.")
		mask := net.CIDRMask(32, 32)
		ipNet := net.IPNet{
			IP:   address.Mask(mask), // Mask the IP to match ParseCIDR()'s behaviour.
			Mask: mask,
		}
		addr := &netlink.Addr{
			IPNet: &ipNet,
		}
		if err := d.dataplane.AddrAdd(link, addr); err != nil {
			log.WithError(err).WithField("addr", address).Warn("Failed to add address")
			return err
		}
	}
	logCxt.Debug("Address set.")

	return nil
}
