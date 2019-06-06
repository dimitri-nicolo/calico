// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package updateprocessors

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
)

// Create a new SyncerUpdateProcessor to sync Node data in v1 format for
// consumption by Felix.
func NewFelixNodeUpdateProcessor() watchersyncer.SyncerUpdateProcessor {
	return &FelixNodeUpdateProcessor{}
}

// FelixNodeUpdateProcessor implements the SyncerUpdateProcessor interface.
// This converts the v3 node configuration into the v1 data types consumed by confd.
type FelixNodeUpdateProcessor struct {
}

func (c *FelixNodeUpdateProcessor) Process(kvp *model.KVPair) ([]*model.KVPair, error) {
	// Extract the name.
	name, err := c.extractName(kvp.Key)
	if err != nil {
		return nil, err
	}

	// Extract the separate bits of BGP config - these are stored as separate keys in the
	// v1 model.  For a delete these will all be nil.  If we fail to convert any value then
	// just treat that as a delete on the underlying key and return the error alongside
	// the updates.
	//
	// Note: it's important that these variables are type interface{} because we use them directly in the
	// KVPair{} literals below.  If we used non-interface types here then we'd end up with zero values for
	// the non=interface types in the KVPair.Value field instead of nil interface{} values (and we want nil
	// interface{} values).
	var ipv4, ipv4Tunl, ipv4Str, vxlanTunlIp, vxlanTunlMac interface{}
	if kvp.Value != nil {
		node, ok := kvp.Value.(*apiv3.Node)
		if !ok {
			return nil, errors.New("Incorrect value type - expecting resource of kind Node")
		}

		if bgp := node.Spec.BGP; bgp != nil {
			var ip *cnet.IP
			var cidr *cnet.IPNet

			// Parse the IPv4 address, Felix expects this as a HostIPKey.  If we fail to parse then
			// treat as a delete (i.e. leave ipv4 as nil).
			if len(bgp.IPv4Address) != 0 {
				ip, cidr, err = cnet.ParseCIDROrIP(bgp.IPv4Address)
				if err == nil {
					log.WithFields(log.Fields{"ip": ip, "cidr": cidr}).Debug("Parsed IPv4 address")
					ipv4 = ip
					ipv4Str = ip.String()
				} else {
					log.WithError(err).WithField("IPv4Address", bgp.IPv4Address).Warn("Failed to parse IPv4Address")
				}
			}

			// Parse the IPv4 IPIP tunnel address, Felix expects this as a HostConfigKey.  If we fail to parse then
			// treat as a delete (i.e. leave ipv4Tunl as nil).
			if len(bgp.IPv4IPIPTunnelAddr) != 0 {
				ip := cnet.ParseIP(bgp.IPv4IPIPTunnelAddr)
				if ip != nil {
					log.WithField("ip", ip).Debug("Parsed IPIP tunnel address")
					ipv4Tunl = ip.String()
				} else {
					log.WithField("IPv4IPIPTunnelAddr", bgp.IPv4IPIPTunnelAddr).Warn("Failed to parse IPv4IPIPTunnelAddr")
					err = fmt.Errorf("failed to parsed IPv4IPIPTunnelAddr as an IP address")
				}
			}
		}

		// Parse the IPv4 VXLAN tunnel address, Felix expects this as a HostConfigKey.  If we fail to parse then
		// treat as a delete (i.e. leave ipv4Tunl as nil).
		if len(node.Spec.IPv4VXLANTunnelAddr) != 0 {
			ip := cnet.ParseIP(node.Spec.IPv4VXLANTunnelAddr)
			if ip != nil {
				log.WithField("ip", ip).Debug("Parsed VXLAN tunnel address")
				vxlanTunlIp = ip.String()
			} else {
				log.WithField("IPv4VXLANTunnelAddr", node.Spec.IPv4VXLANTunnelAddr).Warn("Failed to parse IPv4VXLANTunnelAddr")
				err = fmt.Errorf("failed to parsed IPv4VXLANTunnelAddr as an IP address")
			}
		}
		// Parse the VXLAN tunnel MAC address, Felix expects this as a HostConfigKey.  If we fail to parse then
		// treat as a delete (i.e. leave ipv4Tunl as nil).
		if len(node.Spec.VXLANTunnelMACAddr) != 0 {
			mac := node.Spec.VXLANTunnelMACAddr
			if mac != "" {
				log.WithField("mac addr", mac).Debug("Parsed VXLAN tunnel MAC address")
				vxlanTunlMac = mac
			} else {
				log.WithField("VXLANTunnelMACAddr", node.Spec.VXLANTunnelMACAddr).Warn("VXLANTunnelMACAddr not populated")
				err = fmt.Errorf("failed to update VXLANTunnelMACAddr")
			}
		}
	}

	// Return the add/delete updates and any errors.
	return []*model.KVPair{
		{
			Key: model.HostIPKey{
				Hostname: name,
			},
			Value:    ipv4,
			Revision: kvp.Revision,
		},
		{
			Key: model.HostConfigKey{
				Hostname: name,
				Name:     "NodeIP",
			},
			Value:    ipv4Str,
			Revision: kvp.Revision,
		},
		{
			Key: model.HostConfigKey{
				Hostname: name,
				Name:     "IpInIpTunnelAddr",
			},
			Value:    ipv4Tunl,
			Revision: kvp.Revision,
		},
		{
			Key: model.HostConfigKey{
				Hostname: name,
				Name:     "IPv4VXLANTunnelAddr",
			},
			Value:    vxlanTunlIp,
			Revision: kvp.Revision,
		},
		{
			Key: model.HostConfigKey{
				Hostname: name,
				Name:     "VXLANTunnelMACAddr",
			},
			Value:    vxlanTunlMac,
			Revision: kvp.Revision,
		},
	}, err
}

// Sync is restarting - nothing to do for this processor.
func (c *FelixNodeUpdateProcessor) OnSyncerStarting() {
	log.Debug("Sync starting called on Felix node update processor")
}

func (c *FelixNodeUpdateProcessor) extractName(k model.Key) (string, error) {
	rk, ok := k.(model.ResourceKey)
	if !ok || rk.Kind != apiv3.KindNode {
		return "", errors.New("Incorrect key type - expecting resource of kind Node")
	}
	return rk.Name, nil
}
