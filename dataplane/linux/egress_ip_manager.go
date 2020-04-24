// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/projectcalico/libcalico-go/lib/set"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/golang-collections/collections/stack"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
)

// Egress IP manager watches EgressIPSet and WEP updates.
// One WEP defines one route rule which maps WEP IP to an egress routing table.
// One EgressIPSet defines one egress routing table which consists of ECMP routes.
// One ECMP route is associated with one vxlan L2 route (static ARP and FDB entry)
//
//
//            WEP  WEP  WEP                    WEP  WEP  WEP
//              \   |   /                        \   |   /
//               \  |  / (Match Src FWMark)       \  |  /
//                \ | /                            \ | /
//          Route Table (EgressIPSet)           Route Table n
//             <Index 200>                        <Index n>
//               default                           default
//                / | \                              / | \
//               /  |  \                            /  |  \
//              /   |   \                          /   |   \
// L3 route GatewayIP...GatewayIP_n            GatewayIP...GatewayIP_n
//
// L2 routes  ARP/FDB...ARP/FDB                   ARP/FDB...ARP/FDB
//
// All Routing Rules are managed by a routerule instance.
// Each routing table is managed by a routetable instance for both L3 and L2 routes.
//
// Egress IP manager ensures vxlan interface is configured according to the configuration.
var (
	TableIndexRunout = errors.New("no table index left")
	defaultCidr, _   = ip.ParseCIDROrIP("0.0.0.0/0")
)

type routeRules interface {
	SetRule(rule *routerule.Rule)
	RemoveRule(rule *routerule.Rule)
	QueueResync()
	Apply() error
}

type routeTableGenerator interface {
	NewRouteTable(interfacePrefixes []string,
		ipVersion uint8,
		tableIndex int,
		vxlan bool,
		netlinkTimeout time.Duration,
		deviceRouteSourceAddress net.IP,
		deviceRouteProtocol int,
		removeExternalRoutes bool) routeTable
}

type routeTableFactory struct {
	count int
}

func (f *routeTableFactory) NewRouteTable(interfacePrefixes []string,
	ipVersion uint8,
	tableIndex int,
	vxlan bool,
	netlinkTimeout time.Duration,
	deviceRouteSourceAddress net.IP,
	deviceRouteProtocol int,
	removeExternalRoutes bool) routeTable {

	f.count += 1
	return routetable.New(interfacePrefixes,
		ipVersion,
		true,
		netlinkTimeout,
		deviceRouteSourceAddress,
		deviceRouteProtocol,
		true,
		tableIndex)
}

type egressIPManager struct {
	routerules  routeRules
	rtGenerator routeTableGenerator

	// Routing table index stack.
	tableIndexStack *stack.Stack

	// routetable is allocated on demand and associated to a table index permanently.
	// When an egress ipset is not valid anymore, we still need to remove routes from
	// the table so routetable shoud not be freed immediately.
	// We could have code to free the unused routetable if it is inSync. However, since
	// the total number of routetables is limited, we may just avoid the complexity.
	// Just keep it and it could be reused by another EgressIPSet.
	tableIndexToRouteTable map[int]routeTable

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

	NodeIP net.IP

	nlHandle netlinkHandle
	dpConfig Config
}

func newEgressIPManager(
	deviceName string,
	dpConfig Config,
) *egressIPManager {
	nlHandle, err := netlink.NewHandle()
	if err != nil {
		log.WithError(err).Panic("Failed to get netlink handle.")
	}

	// Prepare table index stack for allocation.
	tableIndexStack := stack.New()
	// Prepare table index set to be passed to routerules.
	tableIndexSet := set.New()
	rtableIndices := dpConfig.RouteTableManager.GrabAllRemainingIndices()
	rtableIndices.Iter(func(item interface{}) error {
		tableIndexStack.Push(item.(int))
		tableIndexSet.Add(item.(int))
		return nil
	})

	// Create routerules to manage routing rules.
	rr, err := routerule.New(4, dpConfig.EgressIPRoutingRulePriority, tableIndexSet, routerule.RulesMatchSrcFWMarkTable, routerule.RulesMatchSrcFWMark, dpConfig.NetlinkTimeout)
	if err != nil {
		// table index has been checked by config.
		// This should not happen.
		log.Panicf("error creating routerule instance")
	}

	return newEgressIPManagerWithShims(
		rr,
		&routeTableFactory{count: 0},
		tableIndexStack,
		deviceName,
		dpConfig,
		nlHandle,
	)
}

func newEgressIPManagerWithShims(
	routerules routeRules,
	rtGenerator routeTableGenerator,
	tableIndexStack *stack.Stack,
	deviceName string,
	dpConfig Config,
	nlHandle netlinkHandle,
) *egressIPManager {

	return &egressIPManager{
		routerules:              routerules,
		rtGenerator:             rtGenerator,
		tableIndexStack:         tableIndexStack,
		tableIndexToRouteTable:  make(map[int]routeTable),
		egressIPSetToTableIndex: make(map[string]int),
		pendingWlEpUpdates:      make(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint),
		activeEgressIPSet:       make(map[string]set.Set),
		activeWlEndpoints:       make(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint),
		vxlanDevice:             deviceName,
		vxlanID:                 dpConfig.RulesConfig.EgressIPVXLANVNI,
		vxlanPort:               dpConfig.RulesConfig.EgressIPVXLANPort,
		dirtyEgressIPSet:        set.New(),
		dpConfig:                dpConfig,
		nlHandle:                nlHandle,
	}
}

func (m *egressIPManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.IPSetDeltaUpdate:
		log.WithField("msg", msg).Debug("IP set delta update")
		if _, found := m.activeEgressIPSet[msg.Id]; found {
			m.handleEgressIPSetDeltaUpdate(msg.Id, msg.RemovedMembers, msg.AddedMembers)
			m.dirtyEgressIPSet.Add(msg.Id)
		}
	case *proto.IPSetUpdate:
		log.WithField("msg", msg).Debug("IP set update")
		if msg.Type == proto.IPSetUpdate_EGRESS_IP {
			m.handleEgressIPSetUpdate(msg)
			m.dirtyEgressIPSet.Add(msg.Id)
		}
	case *proto.IPSetRemove:
		log.WithField("msg", msg).Debug("IP set remove")
		if _, found := m.activeEgressIPSet[msg.Id]; found {
			m.handleEgressIPSetRemove(msg)
			m.dirtyEgressIPSet.Add(msg.Id)
		}
	case *proto.WorkloadEndpointUpdate:
		log.WithField("msg", msg).Debug("workload endpoint update")
		m.pendingWlEpUpdates[*msg.Id] = msg.Endpoint
	case *proto.WorkloadEndpointRemove:
		log.WithField("msg", msg).Debug("workload endpoint remove")
		m.pendingWlEpUpdates[*msg.Id] = nil
	case *proto.HostMetadataUpdate:
		log.WithField("msg", msg).Debugf("song Local host update %s", m.dpConfig.FelixHostname)
		if msg.Hostname == m.dpConfig.FelixHostname {
			log.WithField("msg", msg).Debug("Local host update")
			m.NodeIP = net.ParseIP(msg.Ipv4Addr)
		}
	}
}

func (m *egressIPManager) handleEgressIPSetUpdate(msg *proto.IPSetUpdate) {
	log.Infof("Update whole EgressIP set: msg=%v", msg)
	m.activeEgressIPSet[msg.Id] = set.FromArray(msg.Members)
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
func (m *egressIPManager) workloadToRulesMatchSrcFWMark(workload *proto.WorkloadEndpoint) []*routerule.Rule {
	rules := []*routerule.Rule{}
	for _, s := range workload.Ipv4Nets {
		cidr := ip.MustParseCIDROrIP(s)
		rule := routerule.NewRule(4, m.dpConfig.EgressIPRoutingRulePriority)
		rules = append(rules, rule.MatchSrcAddress(cidr.ToIPNet()).MatchFWMark(m.dpConfig.RulesConfig.IptablesMarkEgress))
	}
	return rules
}

// Construct arrays of full routing rules related to a workload.
func (m *egressIPManager) workloadToFullRules(workload *proto.WorkloadEndpoint, tableIndex int) []*routerule.Rule {
	rules := []*routerule.Rule{}
	for _, s := range workload.Ipv4Nets {
		cidr := ip.MustParseCIDROrIP(s)
		rule := routerule.NewRule(4, m.dpConfig.EgressIPRoutingRulePriority)
		rules = append(rules, rule.MatchSrcAddress(cidr.ToIPNet()).MatchFWMark(m.dpConfig.RulesConfig.IptablesMarkEgress).GoToTable(tableIndex))
	}
	return rules
}

func sortStringSet(s set.Set) []string {
	sorted := []string{}
	s.Iter(func(item interface{}) error {
		sorted = append(sorted, item.(string))
		return nil
	})
	sort.Slice(sorted, func(p, q int) bool {
		return sorted[p] < sorted[q]
	})
	return sorted
}

// Set L3 and L2 routes for an EgressIPSet.
func (m *egressIPManager) setL3L2Routes(rTable routeTable, ips set.Set) {
	logCxt := log.WithField("table", rTable.Index())
	l2routes := []routetable.L2Target{}
	multipath := []ip.Addr{}

	// Sort ips to make ECMP route deterministic.
	sorted := sortStringSet(ips)

	for _, element := range sorted {
		ipString := strings.Split(element, "/")[0]
		l2routes = append(l2routes, routetable.L2Target{
			// remote VTEP mac is generated based on gateway pod ip.
			VTEPMAC: ipStringToMac(ipString),
			GW:      ip.FromString(ipString),
			IP:      ip.FromString(ipString),
		})
		multipath = append(multipath, ip.FromString(ipString))
	}

	// Set L2 route. If there is no l2route target, old entries will be removed.
	logCxt.WithField("l2routes", l2routes).Info("Egress ip manager sending L2 updates")
	rTable.SetL2Routes(m.vxlanDevice, l2routes)

	if len(multipath) > 0 {
		// Set L3 route.
		route := routetable.Target{
			Type:      routetable.TargetTypeVXLAN,
			CIDR:      defaultCidr,
			MultiPath: multipath,
		}
		logCxt.WithField("ecmproute", route).Info("Egress ip manager sending ECMP VXLAN L3 updates")
		rTable.SetRoutes(m.vxlanDevice, []routetable.Target{route})
	} else {
		// Set unreachable
		route := routetable.Target{
			Type: routetable.TargetTypeUnreachable,
			CIDR: defaultCidr,
		}

		// TODO: Should based on latest OS routetable code.
		logCxt.WithField("ecmproute", route).Info("Egress ip manager sending unreachable upates")
		rTable.SetRoutes("NoInterface", []routetable.Target{route})
	}
}

func (m *egressIPManager) CompleteDeferredWork() error {
	if m.dirtyEgressIPSet.Len() == 0 && len(m.pendingWlEpUpdates) == 0 {
		log.Debug("No change since last application, nothing to do")
		return nil
	}

	// Sort set to make table allocate/release deterministic.
	sorted := sortStringSet(m.dirtyEgressIPSet)

	// Work out egress ip set updates.
	for _, id := range sorted {
		currentIndex, ipsetToIndexExists := m.egressIPSetToTableIndex[id]
		if ips, found := m.activeEgressIPSet[id]; !found {
			// EgressIPSet been removed.
			rTable := m.tableIndexToRouteTable[currentIndex]
			if !ipsetToIndexExists || rTable == nil {
				// Something wrong, this should not happen. Panic.
				log.Panic("Removing an egress IPSet with invalid table index")
			}

			// Remove routes.
			log.WithField("tableindex", currentIndex).Info("EgressIPManager remove routes and release route table.")
			rTable.SetRoutes(m.vxlanDevice, nil)
			rTable.SetL2Routes(m.vxlanDevice, nil)

			// Once routes pending being removed, we can safely push table index back to stack.
			m.tableIndexStack.Push(currentIndex)
			delete(m.egressIPSetToTableIndex, id)
		} else {
			if !ipsetToIndexExists {
				// EgressIPSet been added. No table index yet.
				if m.tableIndexStack.Len() == 0 {
					// Run out of egress routing table. Panic.
					log.Panic("Run out of egress ip route table")
				}

				index := m.tableIndexStack.Pop().(int)
				if m.tableIndexToRouteTable[index] == nil {
					// Allocate a routetable if it does not exists.
					m.tableIndexToRouteTable[index] = m.rtGenerator.NewRouteTable([]string{m.vxlanDevice}, 4, index, true, m.dpConfig.NetlinkTimeout, nil,
						m.dpConfig.DeviceRouteProtocol, true)
					log.WithField("tableindex", index).Info("EgressIPManager allocate new route table.")
				}
				m.egressIPSetToTableIndex[id] = index
				currentIndex = index
			}

			rTable := m.tableIndexToRouteTable[currentIndex]

			// Add L3 and L2 routes for EgressIPSet.
			m.setL3L2Routes(rTable, ips)
		}

		// Remove id from dirtyEgressIPSet.
		m.dirtyEgressIPSet.Discard(id)
	}

	// Work out WEP updates.
	// Handle pending workload endpoint updates.
	for id, workload := range m.pendingWlEpUpdates {
		logCxt := log.WithField("id", id)
		oldWorkload := m.activeWlEndpoints[id]
		if workload != nil && workload.EgressIPSetId != "" {
			logCxt.WithFields(log.Fields{
				"workload":    workload,
				"oldworkload": oldWorkload,
			}).Info("Updating endpoint routing rule.")
			if oldWorkload != nil && oldWorkload.EgressIPSetId != workload.EgressIPSetId {
				logCxt.Debug("EgressIPSet changed, cleaning up old state")
				for _, r := range m.workloadToRulesMatchSrcFWMark(oldWorkload) {
					m.routerules.RemoveRule(r)
				}
			}

			// We are not checking if workload state is active or not,
			// There is no big downside if we populate routing rule for
			// an inactive workload.
			IPSetId := workload.EgressIPSetId
			index := m.egressIPSetToTableIndex[IPSetId]
			if index == 0 {
				// Have not received latest EgressIPSet update or WEP update is out of date.
				// The update stays in pendingWlEpUpdates and will be processed later.
				logCxt.WithField("workload", workload).Debug("wait for ipset update")
				continue
			}

			// Set rules for new workload.
			// Pass full Rules to SetRule.
			for _, r := range m.workloadToFullRules(workload, index) {
				m.routerules.SetRule(r)
			}
			logCxt.WithField("workload", workload).Debug("set workload")
			m.activeWlEndpoints[id] = workload
			delete(m.pendingWlEpUpdates, id)
		} else {
			logCxt.WithField("oldworkload", oldWorkload).Info("Workload removed or egress ipset id is empty, deleting its rules.")

			if oldWorkload != nil {
				for _, r := range m.workloadToRulesMatchSrcFWMark(oldWorkload) {
					m.routerules.RemoveRule(r)
				}
			}
			delete(m.activeWlEndpoints, id)
			delete(m.pendingWlEpUpdates, id)
		}
	}

	return nil
}

func (m *egressIPManager) GetRouteTables() []routeTable {
	rts := []routeTable{}
	for _, t := range m.tableIndexToRouteTable {
		rts = append(rts, t)
	}

	return rts
}

func (m *egressIPManager) GetRouteRules() []routeRules {
	return []routeRules{m.routerules}
}

func ipStringToMac(s string) net.HardwareAddr {
	ipAddr := ip.FromString(s).AsNetIP()
	// Any MAC address that has the values 2, 3, 6, 7, A, B, E, or F
	// as the second most significant nibble are locally administered.
	hw := net.HardwareAddr(append([]byte{0xa2, 0x2a}, ipAddr...))
	return hw
}

func (m *egressIPManager) KeepVXLANDeviceInSync(mtu int, wait time.Duration) {
	log.Info("egress ip VXLAN tunnel device thread started.")
	retry := 0
	for {
		err := m.configureVXLANDevice(mtu)
		if err != nil {
			log.WithError(err).Warn("Failed configure egress ip VXLAN tunnel device, retrying...")
			time.Sleep(1 * time.Second)
			retry += 1

			if retry > 3 {
				log.Warnf("song set host ip manually")
				m.NodeIP = m.dpConfig.NodeIP
			}
			continue
		}
		log.Info("egress ip VXLAN tunnel device configured")
		time.Sleep(wait)
	}
}

// getParentInterface returns the parent interface for the given local NodeIP based on IP address. This link returned is nil
// if, and only if, an error occurred
func (m *egressIPManager) getParentInterface() (netlink.Link, error) {
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
			if addr.IPNet.IP.Equal(m.NodeIP) {
				log.Debugf("Found parent interface: %s", link)
				return link, nil
			}
		}
	}
	return nil, fmt.Errorf("Unable to find parent interface with address %s", m.NodeIP.String())
}

// configureVXLANDevice ensures the VXLAN tunnel device is up and configured correctly.
func (m *egressIPManager) configureVXLANDevice(mtu int) error {
	logCxt := log.WithFields(log.Fields{"device": m.vxlanDevice})
	logCxt.Debug("Configuring egress ip VXLAN tunnel device")
	parent, err := m.getParentInterface()
	if err != nil {
		return err
	}

	// Egress ip vxlan device does not need to have tunnel address and mac
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: m.vxlanDevice,
		},
		VxlanId:      m.vxlanID,
		Port:         m.vxlanPort,
		VtepDevIndex: parent.Attrs().Index,
		SrcAddr:      m.NodeIP,
	}

	// Try to get the device.
	link, err := m.nlHandle.LinkByName(m.vxlanDevice)
	if err != nil {
		log.WithError(err).Info("Failed to get egress ip VXLAN tunnel device, assuming it isn't present")
		if err := m.nlHandle.LinkAdd(vxlan); err == syscall.EEXIST {
			// Device already exists - likely a race.
			log.Debug("egress ip VXLAN device already exists, likely created by someone else.")
		} else if err != nil {
			// Error other than "device exists" - return it.
			return err
		}

		// The device now exists - requery it to check that the link exists and is a vxlan device.
		link, err = m.nlHandle.LinkByName(m.vxlanDevice)
		if err != nil {
			return fmt.Errorf("can't locate created egress ip vxlan device %v", m.vxlanDevice)
		}
	}

	// At this point, we have successfully queried the existing device, or made sure it exists if it didn't
	// already. Check for mismatched configuration. If they don't match, recreate the device.
	if incompat := vxlanLinksIncompat(vxlan, link); incompat != "" {
		// Existing device doesn't match desired configuration - delete it and recreate.
		log.Warningf("%q exists with incompatible configuration: %v; recreating device", vxlan.Name, incompat)
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
		logCxt.WithFields(log.Fields{"old": oldMTU, "new": mtu}).Info("VXLAN device MTU needs to be updated")
		if err := m.nlHandle.LinkSetMTU(link, mtu); err != nil {
			log.WithError(err).Warn("Failed to set vxlan tunnel device MTU")
		} else {
			logCxt.Info("Updated vxlan tunnel MTU")
		}
	}

	// And the device is up.
	if err := m.nlHandle.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set interface up: %s", err)
	}

	return nil
}
