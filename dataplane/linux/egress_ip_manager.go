// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"

	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/golang-collections/collections/stack"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
	"golang.org/x/sys/unix"
)

// Egress IP manager watches EgressIPSet and WEP updates.
// One WEP defines one ip rule which matches WEP IP to egress routing table.
// One EgressIPSet defines one egress routing table which consists of ECMP routes.
// One ECMP route defines one vxlan L2 route (static ARP and FDB entry)

// Egress IP manager also ensures vxlan interface is configured according to the configuration.

const (
	invalidTableIndex = 0xffff
)

var (
	TableIndexRunout = errors.New("no table index left")
)

type egressIPManager struct {
	sync.Mutex

	// Main routing table for L2 routes.
	// mainRT     routetable.RouteTable

	fwmark     uint32
	routerules *routerule.RouteRules

	// Routing table index stack.
	tableIndexStack *stack.Stack

	// routetable is allocated on demand and associated to a table index permanently.
	// When an egress ipset is not valid anymore, we still need to remove routes from
	// the table so routetable shoud not be freed immediately.
	// We could have code to free the unused routetable if it is inSync. However, since
	// the total number of routetables is limited, we may just avoid the complexity.
	// Just keep it and it could be reused by another EgressIPSet.
	tableIndexToRouteTable map[int]*routetable.RouteTable

	activeEgressIPSet       map[string]set.Set
	egressIPSetToTableIndex map[string]int

	activeWlEndpoints map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint

	// Pending workload endpoints and egress ipset updates, we store these up as OnUpdate is called, then process them
	// in CompleteDeferredWork.
	pendingWlEpUpdates map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint

	// Dirty Egress IPSet to be processed in CompleteDeferredWork.
	dirtyEgressIPSet set.Set

	// VXLAN configuration.
	vxlanDevice string
	vxlanID     int
	vxlanPort   int

	nlHandle netlinkHandle
	dpConfig Config
}

func newEgressIPManager(
	deviceName string,
	dpConfig Config,
) (*egressIPManager, error) {
	nlHandle, _ := netlink.NewHandle()

	return newEgressIPManagerWithShims(
		0, 0,
		deviceName,
		dpConfig,
		nlHandle,
	)
}

func newEgressIPManagerWithShims(
	firstTableIndex, tableCount int,
	deviceName string,
	dpConfig Config,
	nlHandle netlinkHandle,
) (*egressIPManager, error) {

	// Check and push available table index to stack.
	if firstTableIndex == 0 {
		return nil, errors.New("routing table index is zero")
	}
	if (firstTableIndex + tableCount) > unix.RT_TABLE_COMPAT {
		return nil, errors.New("routing table index out of range")
	}

	// Prepare table index stack for allocation.
	tableIndexStack := stack.New()
	// Prepare table index set to be passed to routerules.
	tableIndexSet := set.New()
	for i := 0; i < int(tableCount); i++ {
		tableIndexStack.Push(firstTableIndex + i)
		tableIndexSet.Add(firstTableIndex + i)
	}

	// Create routerules to manage routing rules.
	rr, err := routerule.New(4, 1000, tableIndexSet, routerule.RulesMatchSrcFWMarkTable, dpConfig.NetlinkTimeout)
	if err != nil {
		return nil, err
	}

	return &egressIPManager{
		fwmark:                  0x400,
		routerules:              rr,
		tableIndexStack:         tableIndexStack,
		tableIndexToRouteTable:  map[int]*routetable.RouteTable{},
		egressIPSetToTableIndex: map[string]int{},
		pendingWlEpUpdates:      make(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint),
		activeEgressIPSet:       make(map[string]set.Set),
		vxlanDevice:             deviceName,
		vxlanID:                 dpConfig.RulesConfig.VXLANVNI,
		vxlanPort:               dpConfig.RulesConfig.VXLANPort,
		dirtyEgressIPSet:        set.New(),
		dpConfig:                dpConfig,
		nlHandle:                nlHandle,
	}, nil
}

func (m *egressIPManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.IPSetDeltaUpdate:
		log.WithField("ipSetId", msg.Id).Debug("IP set delta update")
		if _, found := m.activeEgressIPSet[msg.Id]; found {
			m.handleEgressIPSetDeltaUpdate(msg.Id, msg.RemovedMembers, msg.AddedMembers)
			m.dirtyEgressIPSet.Add(msg.Id)
		}
	case *proto.IPSetUpdate:
		log.WithField("ipSetId", msg.Id).Debug("IP set update")
		if msg.Type == proto.IPSetUpdate_EGRESS_IP {
			m.handleEgressIPSetUpdate(msg)
			m.dirtyEgressIPSet.Add(msg.Id)
		}
	case *proto.IPSetRemove:
		log.WithField("ipSetId", msg.Id).Debug("IP set remove")
		if _, found := m.activeEgressIPSet[msg.Id]; found {
			m.handleEgressIPSetRemove(msg)
			m.dirtyEgressIPSet.Add(msg.Id)
		}
	case *proto.WorkloadEndpointUpdate:
		m.pendingWlEpUpdates[*msg.Id] = msg.Endpoint
	case *proto.WorkloadEndpointRemove:
		m.pendingWlEpUpdates[*msg.Id] = nil
	}
}

func (m *egressIPManager) handleEgressIPSetUpdate(msg *proto.IPSetUpdate) {
	log.Infof("Update whole EgressIP set: msg=%v", msg)

	if _, found := m.activeEgressIPSet[msg.Id]; found {
		log.Info("EgressIP IPSetUpdate for existing IP set")
		membersToRemove := []string{}
		membersToAdd := msg.Members

		m.handleEgressIPSetDeltaUpdate(msg.Id, membersToRemove, membersToAdd)
		return
	}

	m.activeEgressIPSet[msg.Id] = set.FromArray(msg.Members)
	m.egressIPSetToTableIndex[msg.Id] = invalidTableIndex
}

func (m *egressIPManager) handleEgressIPSetRemove(msg *proto.IPSetRemove) {
	log.Infof("Remove whole EgressIP set: msg=%v", msg)

	delete(m.activeEgressIPSet, msg.Id)
}

func (m *egressIPManager) handleEgressIPSetDeltaUpdate(ipSetId string, membersRemoved []string, membersAdded []string) {
	log.Infof("EgressIP set delta update: id=%v removed=%v added=%v", ipSetId, membersRemoved, membersAdded)

	for _, member := range membersAdded {
		m.activeEgressIPSet[ipSetId].Add(member)
	}

	for _, member := range membersRemoved {
		m.activeEgressIPSet[ipSetId].Discard(member)
	}
}

// Construct arrays of routing rules without table value (matching conditions only) related to a workload.
func (m *egressIPManager) workloadToRulesWithoutTable(workload *proto.WorkloadEndpoint) []*routerule.Rule {
	rules := []*routerule.Rule{}
	for _, s := range workload.Ipv4Nets {
		cidr := ip.MustParseCIDROrIP(s)
		rule := routerule.NewRule(m.routerules.IPVersion, m.routerules.Priority)
		rules = append(rules, rule.MatchSrcAddress(cidr.ToIPNet()).MatchFWMark(m.fwmark))
	}
	return rules
}

// Construct arrays of full routing rules related to a workload.
func (m *egressIPManager) workloadToRules(workload *proto.WorkloadEndpoint, tableIndex int) []*routerule.Rule {
	rules := []*routerule.Rule{}
	for _, s := range workload.Ipv4Nets {
		cidr := ip.MustParseCIDROrIP(s)
		rule := routerule.NewRule(m.routerules.IPVersion, m.routerules.Priority)
		rules = append(rules, rule.MatchSrcAddress(cidr.ToIPNet()).MatchFWMark(m.fwmark).GoToTable(tableIndex))
	}
	return rules
}

func (m *egressIPManager) CompleteDeferredWork() error {
	if m.dirtyEgressIPSet.Len() == 0 && len(m.pendingWlEpUpdates) == 0 {
		logrus.Debug("No change since last application, nothing to do")
		return nil
	}

	// Work out egress ip set updates.
	m.dirtyEgressIPSet.Iter(func(item interface{}) error {
		id := item.(string)

		if _, found := m.activeEgressIPSet[id]; !found {
			// EgressIPSet been removed.
			index := m.egressIPSetToTableIndex[id]
			rTable := m.tableIndexToRouteTable[index]
			if index == 0 || rTable == nil {
				// Something wrong, this should not happen. return error and panic.
				return errors.New("Removing an egress IPSet with invalid table index")
			}

			// Remove routes.
			logrus.WithField("tableindex", index).Debug("EgressIPManager remove routes.")
			rTable.SetRoutes(m.vxlanDevice, nil)

			// Once routes pending being removed, we can safely push table index back to stack.
			m.tableIndexStack.Push(index)
			delete(m.egressIPSetToTableIndex, id)
		} else {
			if m.egressIPSetToTableIndex[id] == invalidTableIndex {
				// EgressIPSet been added. No table index yet.
				if m.tableIndexStack.Len() == 0 {
					// Run out of egress routing table. Log error and Panic. TODO: send to black hole table
					return errors.New("Run out of egress ip route table")
				}

				index := m.tableIndexStack.Pop().(int)
				m.egressIPSetToTableIndex[id] = index
				if m.tableIndexToRouteTable[index] == nil {
					// Allocate a routetable if it does not exists.
					routeTable := routetable.New([]string{"vxlan.calico"}, 4, true, m.dpConfig.NetlinkTimeout, m.dpConfig.DeviceRouteSourceAddress,
						m.dpConfig.DeviceRouteProtocol, true)
					m.tableIndexToRouteTable[index] = routeTable
				}
			}

			// Add ECMP routes for EgressIPSet.

		}

		// Remove id from dirtyEgressIPSet.
		return set.RemoveItem
	})

	// Work out WEP updates.
	for len(m.pendingWlEpUpdates) > 0 {
		// Handle pending workload endpoint updates.
		for id, workload := range m.pendingWlEpUpdates {
			logCxt := log.WithField("id", id)
			oldWorkload := m.activeWlEndpoints[id]
			if workload != nil {
				logCxt.Info("Updating endpoint routing rule.")
				if oldWorkload != nil && oldWorkload.EgressIpSetId != workload.EgressIpSetId {
					logCxt.Debug("EgressIPSet changed, cleaning up old state")
					for _, r := range m.workloadToRulesWithoutTable(oldWorkload) {
						m.routerules.RemoveRule(r, routerule.RulesMatchSrcFWMark)
					}
				}

				// We are not checking if workload state is active or not,
				// There is no big downside if we populate routing rule for
				// an inactive workload.
				IPSetId := workload.EgressIpSetId
				index := m.egressIPSetToTableIndex[IPSetId]
				if index == 0 {
					// EgressIPSet update has not been done yet.
					// The update stays in pendingWlEpUpdates and will be processed later.
					continue
				}

				// Set rules for new workload.
				// Pass full Rules to SetRule.
				for _, r := range m.workloadToRules(workload, index) {
					m.routerules.SetRule(r, routerule.RulesMatchSrcFWMarkTable)
				}
				m.activeWlEndpoints[id] = workload
				delete(m.pendingWlEpUpdates, id)
			} else {
				logCxt.Info("Workload removed, deleting its rules.")

				if oldWorkload != nil {
					for _, r := range m.workloadToRulesWithoutTable(oldWorkload) {
						m.routerules.RemoveRule(r, routerule.RulesMatchSrcFWMark)
					}
				}
				delete(m.pendingWlEpUpdates, id)
			}
		}
	}

	return nil
}

/*
func (m *egressIPManager) KeepVXLANDeviceInSync(mtu int, wait time.Duration) {
	logrus.Info("VXLAN tunnel device thread started.")
	for {
		localVTEP := m.getLocalVTEP()
		if localVTEP == nil {
			logrus.Debug("Missing local VTEP information, retrying...")
			time.Sleep(1 * time.Second)
			continue
		}

		if parent, err := m.getLocalVTEPParent(); err != nil {
			logrus.WithError(err).Warn("Failed configure VXLAN tunnel device, retrying...")
			time.Sleep(1 * time.Second)
			continue
		} else {
			if m.getNoEncapRouteTable() == nil {
				noEncapRouteTable := m.noEncapRTConstruct([]string{parent.Attrs().Name}, 4, false, m.dpConfig.NetlinkTimeout, m.dpConfig.DeviceRouteSourceAddress,
					m.noEncapProtocol, false)
				m.setNoEncapRouteTable(noEncapRouteTable)
			}
		}

		err := m.configureVXLANDevice(mtu, localVTEP)
		if err != nil {
			logrus.WithError(err).Warn("Failed configure VXLAN tunnel device, retrying...")
			time.Sleep(1 * time.Second)
			continue
		}
		logrus.Info("VXLAN tunnel device configured")
		time.Sleep(wait)
	}
}
*/

// getParentInterface returns the parent interface for the given local VTEP based on IP address. This link returned is nil
// if, and only if, an error occurred
func (m *egressIPManager) getParentInterface(localVTEP *proto.VXLANTunnelEndpointUpdate) (netlink.Link, error) {
	links, err := m.nlHandle.LinkList()
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		addrs, err := m.nlHandle.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if addr.IPNet.IP.String() == localVTEP.ParentDeviceIp {
				logrus.Debugf("Found parent interface: %s", link)
				return link, nil
			}
		}
	}
	return nil, fmt.Errorf("Unable to find parent interface with address %s", localVTEP.ParentDeviceIp)
}

// configureVXLANDevice ensures the VXLAN tunnel device is up and configured correctly.
func (m *egressIPManager) configureVXLANDevice(mtu int, localVTEP *proto.VXLANTunnelEndpointUpdate) error {
	logCxt := logrus.WithFields(logrus.Fields{"device": m.vxlanDevice})
	logCxt.Debug("Configuring VXLAN tunnel device")
	parent, err := m.getParentInterface(localVTEP)
	if err != nil {
		return err
	}
	mac, err := net.ParseMAC(localVTEP.Mac)
	if err != nil {
		return err
	}
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:         m.vxlanDevice,
			HardwareAddr: mac,
		},
		VxlanId:      m.vxlanID,
		Port:         m.vxlanPort,
		VtepDevIndex: parent.Attrs().Index,
		SrcAddr:      ip.FromString(localVTEP.ParentDeviceIp).AsNetIP(),
	}

	// Try to get the device.
	link, err := m.nlHandle.LinkByName(m.vxlanDevice)
	if err != nil {
		logrus.WithError(err).Info("Failed to get VXLAN tunnel device, assuming it isn't present")
		if err := m.nlHandle.LinkAdd(vxlan); err == syscall.EEXIST {
			// Device already exists - likely a race.
			logrus.Debug("VXLAN device already exists, likely created by someone else.")
		} else if err != nil {
			// Error other than "device exists" - return it.
			return err
		}

		// The device now exists - requery it to check that the link exists and is a vxlan device.
		link, err = m.nlHandle.LinkByName(m.vxlanDevice)
		if err != nil {
			return fmt.Errorf("can't locate created vxlan device %v", m.vxlanDevice)
		}
	}

	// At this point, we have successfully queried the existing device, or made sure it exists if it didn't
	// already. Check for mismatched configuration. If they don't match, recreate the device.
	if incompat := vxlanLinksIncompat(vxlan, link); incompat != "" {
		// Existing device doesn't match desired configuration - delete it and recreate.
		logrus.Warningf("%q exists with incompatable configuration: %v; recreating device", vxlan.Name, incompat)
		if err = m.nlHandle.LinkDel(link); err != nil {
			return fmt.Errorf("failed to delete interface: %v", err)
		}
		if err = m.nlHandle.LinkAdd(vxlan); err != nil {
			if err == syscall.EEXIST {
				log.Warnf("Failed to create VXLAN device. Another device with this VNI may already exist")
			}
			return fmt.Errorf("failed to create vxlan interface: %v", err)
		}
		link, err = m.nlHandle.LinkByName(vxlan.Name)
		if err != nil {
			return err
		}
	}

	// Make sure the MTU is set correctly.
	attrs := link.Attrs()
	oldMTU := attrs.MTU
	if oldMTU != mtu {
		logCxt.WithFields(logrus.Fields{"old": oldMTU, "new": mtu}).Info("VXLAN device MTU needs to be updated")
		if err := m.nlHandle.LinkSetMTU(link, mtu); err != nil {
			log.WithError(err).Warn("Failed to set vxlan tunnel device MTU")
		} else {
			logCxt.Info("Updated vxlan tunnel MTU")
		}
	}

	// Make sure the IP address is configured.
	//if err := m.ensureV4AddressOnLink(localVTEP.Ipv4Addr, link); err != nil {
	//return fmt.Errorf("failed to ensure address of interface: %s", err)
	//}

	// And the device is up.
	if err := m.nlHandle.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set interface up: %s", err)
	}

	return nil
}
