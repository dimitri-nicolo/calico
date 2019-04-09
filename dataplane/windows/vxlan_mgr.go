// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package windataplane

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/Microsoft/hcsshim/hcn"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type vxlanManager struct {
	// Our dependencies.
	hostname string

	// Hold pending updates.
	routesByDest map[string]*proto.RouteUpdate
	vtepsByNode  map[string]*proto.VXLANTunnelEndpointUpdate

	// Holds this node's VTEP information.
	myVTEP *proto.VXLANTunnelEndpointUpdate

	// VXLAN configuration.
	networkName *regexp.Regexp
	vxlanID     int
	vxlanPort   int

	// Indicates if configuration has changed since the last apply.
	dirty bool
}

func newVXLANManager(hostname string, networkName *regexp.Regexp, vxlanID, port int) *vxlanManager {
	return &vxlanManager{
		hostname:     hostname,
		routesByDest: map[string]*proto.RouteUpdate{},
		vtepsByNode:  map[string]*proto.VXLANTunnelEndpointUpdate{},
		networkName:  networkName,
		vxlanID:      vxlanID,
		vxlanPort:    port,
	}
}

func (m *vxlanManager) OnUpdate(protoBufMsg interface{}) {
	switch msg := protoBufMsg.(type) {
	case *proto.RouteUpdate:
		if msg.Type == proto.RouteType_VXLAN {
			logrus.WithField("msg", msg).Debug("VXLAN data plane received route update")
			m.routesByDest[msg.Dst] = msg
			m.dirty = true
		}
	case *proto.RouteRemove:
		if msg.Type == proto.RouteType_VXLAN {
			logrus.WithField("msg", msg).Debug("VXLAN data plane received route remove")
			delete(m.routesByDest, msg.Dst)
			m.dirty = true
		}
	case *proto.VXLANTunnelEndpointUpdate:
		logrus.WithField("msg", msg).Debug("VXLAN data plane received VTEP update")
		if msg.Node == m.hostname {
			m.setLocalVTEP(msg)
		} else {
			m.vtepsByNode[msg.Node] = msg
		}
		m.dirty = true
	case *proto.VXLANTunnelEndpointRemove:
		logrus.WithField("msg", msg).Debug("VXLAN data plane received VTEP remove")
		if msg.Node == m.hostname {
			m.setLocalVTEP(nil)
		} else {
			delete(m.vtepsByNode, msg.Node)
		}
		m.dirty = true
	}
}

func (m *vxlanManager) setLocalVTEP(vtep *proto.VXLANTunnelEndpointUpdate) {
	m.myVTEP = vtep
}

func (m *vxlanManager) getLocalVTEP() *proto.VXLANTunnelEndpointUpdate {
	return m.myVTEP
}

func (m *vxlanManager) CompleteDeferredWork() error {
	if !m.dirty {
		logrus.Debug("No change since last application, nothing to do")
		return nil
	}
	// Find the right network
	networks, err := hcn.ListNetworks()
	if err != nil {
		logrus.WithError(err).Error("Failed to look up HNS networks.")
		return err
	}

	var network *hcn.HostComputeNetwork
	for _, n := range networks {
		if m.networkName.MatchString(n.Name) {
			network = &n
			break
		}
	}

	if network == nil {
		return fmt.Errorf("didn't find any HNS networks matching regular expression %s", m.networkName.String())
	}

	if network.Type != "Overlay" {
		if len(m.routesByDest) > 0 || len(m.vtepsByNode) > 0 {
			return fmt.Errorf("have VXLAN routes but HNS network, %s, is of wrong type: %s",
				network.Name, network.Type)
		}
	}

	// Calculate what should be there as a whole, then, below, we'll remove items that are already there from this set.
	netPolsToAdd := set.New()
	for n, r := range m.routesByDest {
		logrus.WithFields(logrus.Fields{
			"node": n,
			"vtep": r,
		}).Debug("Currently-active VTEP")

		vtep := m.vtepsByNode[r.Node]
		if vtep == nil {
			logrus.WithField("node", r.Node).Error("Received route without corresponding VTEP")
		}

		networkPolicySettings := hcn.RemoteSubnetRoutePolicySetting{
			IsolationId:                 uint16(m.vxlanID),
			DistributedRouterMacAddress: macToWindowsFormat(vtep.Mac),
			ProviderAddress:             vtep.ParentDeviceIp,
			DestinationPrefix:           r.Dst,
		}

		netPolsToAdd.Add(networkPolicySettings)
	}

	// Load what's actually there.
	netPolsToRemove := set.New()
	for _, policy := range network.Policies {
		if policy.Type == hcn.RemoteSubnetRoute {
			existingPolSettings := hcn.RemoteSubnetRoutePolicySetting{}
			err = json.Unmarshal(policy.Settings, &existingPolSettings)
			if err != nil {
				logrus.Error("Failed to unmarshal existing route policy")
				return err
			}

			// Filter down to only the
			filteredPolSettings := hcn.RemoteSubnetRoutePolicySetting{
				IsolationId:                 existingPolSettings.IsolationId,
				DistributedRouterMacAddress: existingPolSettings.DistributedRouterMacAddress,
				ProviderAddress:             existingPolSettings.ProviderAddress,
				DestinationPrefix:           existingPolSettings.DestinationPrefix,
			}
			if netPolsToAdd.Contains(filteredPolSettings) {
				netPolsToAdd.Discard(filteredPolSettings)
			} else {
				netPolsToRemove.Add(existingPolSettings)
			}
		}
	}

	// Remove routes that are no longer needed.
	netPolsToRemove.Iter(func(item interface{}) error {
		polSettings := item.(hcn.RemoteSubnetRoutePolicySetting)
		polJSON, err := json.Marshal(polSettings)
		if err != nil {
			logrus.WithError(err).WithField("policy", polSettings).Error("Failed to martial HCN policy")
			return nil
		}
		pol := hcn.NetworkPolicy{
			Type:     hcn.RemoteSubnetRoute,
			Settings: polJSON,
		}
		polReq := hcn.PolicyNetworkRequest{
			Policies: []hcn.NetworkPolicy{pol},
		}
		err = network.RemovePolicy(polReq)
		if err != nil {
			logrus.WithError(err).WithField("request", polReq).Error("Failed to remove unwanted VXLAN route policy")
			return nil
		}
		return set.RemoveItem
	})

	// Add new routes.
	netPolsToAdd.Iter(func(item interface{}) error {
		polSettings := item.(hcn.RemoteSubnetRoutePolicySetting)
		polJSON, err := json.Marshal(polSettings)
		if err != nil {
			logrus.WithError(err).WithField("policy", polSettings).Error("Failed to martial HCN policy")
			return nil
		}
		pol := hcn.NetworkPolicy{
			Type:     hcn.RemoteSubnetRoute,
			Settings: polJSON,
		}
		polReq := hcn.PolicyNetworkRequest{
			Policies: []hcn.NetworkPolicy{pol},
		}
		err = network.AddPolicy(polReq)
		if err != nil {
			logrus.WithError(err).WithField("request", polReq).Error("Failed to add VXLAN route policy")
			return nil
		}
		return set.RemoveItem
	})

	if netPolsToAdd.Len() == 0 && netPolsToRemove.Len() == 0 {
		logrus.Info("All VXLAN route updates succeeded.")
		m.dirty = false
	} else {
		logrus.WithFields(logrus.Fields{
			"numFailedAdds":    netPolsToAdd.Len(),
			"numFailedRemoves": netPolsToRemove.Len(),
		}).Error("Not all VXLAN route updates succeeded.")
	}

	return nil
}

func macToWindowsFormat(mac string) string {
	parsed := net.HardwareAddr(mac)
	colonFormat := parsed.String()
	windowsFormat := strings.Replace(colonFormat, ":", "-", 0)
	return windowsFormat
}
