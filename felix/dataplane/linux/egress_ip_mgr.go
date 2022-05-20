// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/projectcalico/calico/libcalico-go/lib/names"
	"github.com/projectcalico/calico/libcalico-go/lib/set"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/golang-collections/collections/stack"

	"github.com/projectcalico/calico/felix/ethtool"
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/logutils"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/routerule"
	"github.com/projectcalico/calico/felix/routetable"
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
	ErrInsufficientRouteTables  = errors.New("ran out of egress ip route tables, increased routeTableRanges required")
	ErrVxlanDeviceNotConfigured = errors.New("egress VXLAN device not configured")
	defaultCidr, _              = ip.ParseCIDROrIP("0.0.0.0/0")
)

const (
	egressHealthName = "egress-networking-in-sync"
)

type healthAggregator interface {
	RegisterReporter(name string, reports *health.HealthReport, timeout time.Duration)
	Report(name string, report *health.HealthReport)
}

type routeTableGenerator interface {
	NewRouteTable(interfacePrefixes []string,
		ipVersion uint8,
		tableIndex int,
		vxlan bool,
		netlinkTimeout time.Duration,
		deviceRouteSourceAddress net.IP,
		deviceRouteProtocol int,
		removeExternalRoutes bool,
		opRecorder logutils.OpRecorder) routeTable
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
	removeExternalRoutes bool,
	opRecorder logutils.OpRecorder) routeTable {

	f.count += 1
	return routetable.New(interfacePrefixes,
		ipVersion,
		vxlan,
		netlinkTimeout,
		deviceRouteSourceAddress,
		netlink.RouteProtocol(deviceRouteProtocol),
		removeExternalRoutes,
		tableIndex,
		opRecorder)
}

type routeRulesGenerator interface {
	NewRouteRules(
		ipVersion int,
		tableIndexSet set.Set,
		updateFunc, removeFunc routerule.RulesMatchFunc,
		netlinkTimeout time.Duration,
		recorder logutils.OpRecorder,
	) routeRules
}

type routeRulesFactory struct {
	count int
}

func (f *routeRulesFactory) NewRouteRules(
	ipVersion int,
	tableIndexSet set.Set,
	updateFunc, removeFunc routerule.RulesMatchFunc,
	netlinkTimeout time.Duration,
	opRecorder logutils.OpRecorder,
) routeRules {

	f.count += 1
	rr, err := routerule.New(
		ipVersion,
		tableIndexSet,
		updateFunc,
		removeFunc,
		netlinkTimeout,
		func() (routerule.HandleIface, error) {
			return netlink.NewHandle(syscall.NETLINK_ROUTE)
		},
		opRecorder)
	if err != nil {
		// table index has been checked by config.
		// This should not happen.
		log.Panicf("error creating routerule instance")
	}

	return rr
}

// gateway stores an IPSet member's cidr and maintenance window.
// If the maintenanceStarted.IsZero() or maintenanceFinished.IsZero() then the member is not terminating.
// Otherwise it is in the process of terminating, and will be deleted at the given maintenanceFinished timestamp.
type gateway struct {
	cidr                string
	maintenanceStarted  time.Time
	maintenanceFinished time.Time
}

func (g gateway) String() string {
	start, err := g.maintenanceStarted.MarshalText()
	if err != nil {
		start = []byte("<invalid_start_time>")
	}
	finish, err := g.maintenanceFinished.MarshalText()
	if err != nil {
		finish = []byte("<invalid_finish_time>")
	}
	return fmt.Sprintf("gateway: [cidr=%s, maintenanceStarted=%s, maintenanceFinished=%s]", g.cidr, string(start), string(finish))
}

// gatewaysByIP maps a member's IP to a gateway
type gatewaysByIP map[string]gateway

func (g gatewaysByIP) getIPs() []string {
	var ips []string
	for _, m := range g {
		ipAddr := ip.MustParseCIDROrIP(m.cidr).Addr()
		ips = append(ips, ipAddr.String())
	}
	return ips
}

func (g gatewaysByIP) getActiveGateways() gatewaysByIP {
	active := make(map[string]gateway)
	now := time.Now()
	for _, m := range g {
		m := m
		if now.Before(m.maintenanceStarted) || now.After(m.maintenanceFinished) {
			active[m.cidr] = m
		}
	}
	return active
}

func (g gatewaysByIP) getTerminatingGateways() gatewaysByIP {
	terminating := make(map[string]gateway)
	now := time.Now()
	for _, m := range g {
		m := m
		if (now.Equal(m.maintenanceStarted) || now.After(m.maintenanceStarted)) &&
			(now.Equal(m.maintenanceFinished) || now.Before(m.maintenanceFinished)) {
			terminating[m.cidr] = m
		}
	}
	return terminating
}

func (g gatewaysByIP) getFilteredGateways(hopIPs []string) gatewaysByIP {
	terminating := make(map[string]gateway)
	hopIPsSet := set.FromArray(hopIPs)
	for cidr, m := range g {
		m := m
		ipAddr := strings.TrimSuffix(cidr, "/32")
		if hopIPsSet.Contains(ipAddr) {
			terminating[cidr] = m
		}
	}
	return terminating
}

// Finds the latest maintenance window on the supplied egress gateway pods.
func (g gatewaysByIP) latestTerminatingGateway() gateway {
	member := gateway{
		cidr:                "",
		maintenanceStarted:  time.Time{},
		maintenanceFinished: time.Time{},
	}
	for _, m := range g.getTerminatingGateways() {
		m := m
		if m.maintenanceFinished.After(member.maintenanceFinished) {
			member = m
		}
	}
	return member
}

type egressRule struct {
	used       bool
	srcIP      string
	priority   int
	family     int
	mark       int
	tableIndex int
}

func newEgressRule(nlRule *netlink.Rule) *egressRule {
	return &egressRule{
		srcIP:      nlRule.Src.IP.String(),
		tableIndex: nlRule.Table,
		priority:   nlRule.Priority,
		family:     nlRule.Family,
		mark:       nlRule.Mark,
	}
}

type egressTable struct {
	used   bool
	index  int
	hopIPs []string
}

func newEgressTable(index int, hopIPs []string) *egressTable {
	return &egressTable{
		index:  index,
		hopIPs: hopIPs,
	}
}

type initialKernelState struct {
	// rules is a map from src IP to egressRule
	rules map[string]*egressRule
	// tables is a map from table index to egressTable
	tables map[int]*egressTable
}

func newInitialKernelState() *initialKernelState {
	return &initialKernelState{
		rules:  make(map[string]*egressRule),
		tables: make(map[int]*egressTable),
	}
}

func (s *initialKernelState) String() string {
	var rules []string
	for srcIP, r := range s.rules {
		rules = append(rules, fmt.Sprintf("%s: [%#v]", srcIP, *r))
	}
	rulesOutput := fmt.Sprintf("rules: {%s}", strings.Join(rules, ","))

	var tables []string
	for index, t := range s.tables {
		tables = append(tables, fmt.Sprintf("%d: [%#v]", index, *t))
	}
	tablesOutput := fmt.Sprintf("tables: {%s}", strings.Join(tables, ","))

	return fmt.Sprintf("initialKernelState:{%s; %s}", rulesOutput, tablesOutput)
}

type egressIPManager struct {
	routeRules routeRules

	initialKernelState *initialKernelState

	// route table for programming L2 routes.
	l2Table routeTable

	// rrGenerator dynamically creates routeRules instance to program route rules.
	rrGenerator routeRulesGenerator

	// rtGenerator dynamically creates route tables to program L3 routes.
	rtGenerator routeTableGenerator

	// Routing table index stack for egress workloads
	tableIndexStack *stack.Stack

	// routetable is allocated on demand and associated to a table index permanently.
	// When an egress ipset is not valid anymore, we still need to remove routes from
	// the table so routetable should not be freed immediately.
	// We could have code to free the unused routetable if it is inSync. However, since
	// the total number of routetables is limited, we may just avoid the complexity.
	// Just keep it and it could be reused by another EgressIPSet.
	tableIndexToRouteTable map[int]routeTable
	// Tracks next hops for all route tables in use.
	tableIndexToNextHops map[int][]string

	ipSetIDToGateways map[string]gatewaysByIP

	activeWorkloads      map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint
	workloadToTableIndex map[proto.WorkloadEndpointID]int

	workloadMaintenanceWindows map[proto.WorkloadEndpointID]gateway

	// Pending workload endpoints updates, we store these up as OnUpdate is called, then process them
	// in CompleteDeferredWork.
	pendingWorkloadUpdates map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint

	// Dirty Egress IPSet to be processed in CompleteDeferredWork.
	dirtyEgressIPSet set.Set

	// VXLAN configuration.
	vxlanDevice string
	vxlanID     int
	vxlanPort   int

	vxlanDeviceLinkIndex int

	NodeIP net.IP

	nlHandle netlinkHandle
	dpConfig Config

	// represents the entire block of table indices the manager is allowed to use.
	// gets passed to routerule package when creating rules
	tableIndexSet set.Set

	opRecorder logutils.OpRecorder

	disableChecksumOffload func(ifName string) error

	// Callback function used to notify of workload pods impacted by a terminating egress gateway pod
	statusCallback func(namespace, name, cidr string, maintenanceStarted, maintenanceFinished time.Time) error

	healthAgg healthAggregator

	// to rate-limit retries, track if the last kernel sync failed, and if our state has changed since then
	lastUpdateFailed, unblockingUpdateOccurred bool

	updateLock sync.Mutex // lock for any flags being used concurrently

	hopRand *rand.Rand
}

func newEgressIPManager(
	deviceName string,
	rtTableIndices set.Set,
	dpConfig Config,
	opRecorder logutils.OpRecorder,
	statusCallback func(namespace, name, cidr string, maintenanceStarted, maintenanceFinished time.Time) error,
	healthAgg healthAggregator,
) *egressIPManager {
	nlHandle, err := netlink.NewHandle()
	if err != nil {
		log.WithError(err).Panic("Failed to get netlink handle.")
	}

	// Prepare table index stack for allocation.
	tableIndexStack := stack.New()
	// Prepare table index set to be passed to routeRules.
	tableIndexSet := set.New()
	// Sort indices to make route table allocation deterministic.
	sorted := sortIntSet(rtTableIndices)
	for _, element := range sorted {
		tableIndexStack.Push(element)
		tableIndexSet.Add(element)
	}

	// Create main route table to manage L2 routing rules.
	l2Table := routetable.New([]string{"^" + deviceName + "$"},
		4, true, dpConfig.NetlinkTimeout, nil,
		dpConfig.DeviceRouteProtocol, true, unix.RT_TABLE_UNSPEC,
		opRecorder)

	hopRandSource := rand.NewSource(time.Now().UTC().UnixNano())

	mgr := newEgressIPManagerWithShims(
		l2Table,
		&routeRulesFactory{count: 0},
		&routeTableFactory{count: 0},
		tableIndexSet,
		tableIndexStack,
		deviceName,
		dpConfig,
		nlHandle,
		opRecorder,
		ethtool.EthtoolTXOff,
		statusCallback,
		healthAgg,
		rand.New(hopRandSource),
	)
	return mgr
}

func newEgressIPManagerWithShims(
	mainTable routeTable,
	rrGenerator routeRulesGenerator,
	rtGenerator routeTableGenerator,
	tableIndexSet set.Set,
	tableIndexStack *stack.Stack,
	deviceName string,
	dpConfig Config,
	nlHandle netlinkHandle,
	opRecorder logutils.OpRecorder,
	disableChecksumOffload func(ifName string) error,
	statusCallback func(namespace, name, cidr string, maintenanceStarted, maintenanceFinished time.Time) error,
	healthAgg healthAggregator,
	hopRandSource rand.Source,
) *egressIPManager {

	mgr := egressIPManager{
		l2Table:                    mainTable,
		rrGenerator:                rrGenerator,
		rtGenerator:                rtGenerator,
		initialKernelState:         newInitialKernelState(),
		tableIndexSet:              tableIndexSet,
		tableIndexStack:            tableIndexStack,
		tableIndexToRouteTable:     make(map[int]routeTable),
		tableIndexToNextHops:       make(map[int][]string),
		pendingWorkloadUpdates:     make(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint),
		ipSetIDToGateways:          make(map[string]gatewaysByIP),
		activeWorkloads:            make(map[proto.WorkloadEndpointID]*proto.WorkloadEndpoint),
		workloadToTableIndex:       make(map[proto.WorkloadEndpointID]int),
		workloadMaintenanceWindows: make(map[proto.WorkloadEndpointID]gateway),
		vxlanDevice:                deviceName,
		vxlanID:                    dpConfig.RulesConfig.EgressIPVXLANVNI,
		vxlanPort:                  dpConfig.RulesConfig.EgressIPVXLANPort,
		dirtyEgressIPSet:           set.New(),
		dpConfig:                   dpConfig,
		nlHandle:                   nlHandle,
		opRecorder:                 opRecorder,
		disableChecksumOffload:     disableChecksumOffload,
		statusCallback:             statusCallback,
		healthAgg:                  healthAgg,
		hopRand:                    rand.New(hopRandSource),
	}

	if healthAgg != nil {
		healthAgg.RegisterReporter(egressHealthName, &health.HealthReport{Ready: true}, 0)
		healthAgg.Report(egressHealthName, &health.HealthReport{Ready: false})
	}

	return &mgr
}

func (m *egressIPManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.IPSetDeltaUpdate:
		log.WithField("msg", msg).Debug("IP set delta update")
		if _, found := m.ipSetIDToGateways[msg.Id]; found {
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
		if _, found := m.ipSetIDToGateways[msg.Id]; found {
			m.handleEgressIPSetRemove(msg)
			m.dirtyEgressIPSet.Add(msg.Id)
		}
	case *proto.WorkloadEndpointUpdate:
		log.WithField("msg", msg).Debug("workload endpoint update")
		m.pendingWorkloadUpdates[*msg.Id] = msg.Endpoint
	case *proto.WorkloadEndpointRemove:
		log.WithField("msg", msg).Debug("workload endpoint remove")
		m.pendingWorkloadUpdates[*msg.Id] = nil
	case *proto.HostMetadataUpdate:
		log.WithField("msg", msg).Debug("host meta update")
		if msg.Hostname == m.dpConfig.FelixHostname {
			log.WithField("msg", msg).Debug("Local host update")
			m.NodeIP = net.ParseIP(msg.Ipv4Addr)
		}
	default:
		return
	}

	// when an update we care about is seen (when the default switch case isn't hit), we track its occurrence
	m.updateLock.Lock()
	defer m.updateLock.Unlock()
	m.unblockingUpdateOccurred = true
}

func (m *egressIPManager) handleEgressIPSetUpdate(msg *proto.IPSetUpdate) {
	log.Infof("Update whole EgressIP set: msg=%v", msg)
	gateways := make(map[string]gateway)
	for _, mStr := range msg.Members {
		member, err := parseMember(mStr)
		if err != nil {
			log.WithError(err).Errorf("error parsing details from memberStr: %s", mStr)
		}
		gateways[member.cidr] = member
	}
	m.ipSetIDToGateways[msg.Id] = gateways
}

func (m *egressIPManager) handleEgressIPSetRemove(msg *proto.IPSetRemove) {
	log.Infof("Remove whole EgressIP set: msg=%v", msg)
	delete(m.ipSetIDToGateways, msg.Id)
}

func (m *egressIPManager) handleEgressIPSetDeltaUpdate(ipSetId string, membersRemoved []string, membersAdded []string) {
	log.Infof("EgressIP set delta update: id=%v removed=%v added=%v", ipSetId, membersRemoved, membersAdded)

	gateways, exists := m.ipSetIDToGateways[ipSetId]
	if !exists {
		gateways = make(map[string]gateway)
		m.ipSetIDToGateways[ipSetId] = gateways
	}

	// The member string contains cidr,deletionTimestamp, and so we could get the same cidr in membersAdded
	// and in membersRemoved, with different timestamps. For this reason, process the removes before the adds.
	for _, mStr := range membersRemoved {
		member, err := parseMember(mStr)
		if err != nil {
			log.WithError(err).Errorf("error parsing ip set member from member string %s", mStr)
		}
		delete(gateways, member.cidr)
	}

	for _, mStr := range membersAdded {
		member, err := parseMember(mStr)
		if err != nil {
			log.WithError(err).Errorf("error parsing ip set member from member string %s", mStr)
		}
		gateways[member.cidr] = member
	}
}

// CompleteDeferredWork attempts to process all updates received by this manager.
// Will attempt a retry if the first attempt fails, and reports health based on its success
func (m *egressIPManager) CompleteDeferredWork() error {
	m.updateLock.Lock()
	defer func() {
		// reset flag after attempting to apply an unblocking update
		m.unblockingUpdateOccurred = false
		m.updateLock.Unlock()
	}()

	// Retry completing deferred work once.
	// The VXLAN device may have come online, or
	// a routetable may have been free'd following a starvation error
	var err error
	if !m.lastUpdateFailed || m.unblockingUpdateOccurred {
		for i := 0; i < 2; i += 1 {
			if err = m.completeDeferredWork(); err == nil {
				m.lastUpdateFailed = false
				break
			}
		}
	}

	// report health
	if err != nil {
		m.lastUpdateFailed = true
		log.WithError(err).Warn("Failed to configure egress networking for one or more workloads")
		m.healthAgg.Report(egressHealthName, &health.HealthReport{Ready: false})
	} else {
		m.healthAgg.Report(egressHealthName, &health.HealthReport{Ready: true})
	}

	return nil // we manage our own retries and health, so never report an error to the dp driver
}

// completeDeferredWork processes all received updates and queues kernel networking updates.
// When called for the first time, will init egressIPManager config with existing kernel data
func (m *egressIPManager) completeDeferredWork() error {
	var lastErr error
	if m.dirtyEgressIPSet.Len() == 0 && len(m.pendingWorkloadUpdates) == 0 {
		log.Debug("No change since last application, nothing to do")
		return nil
	}

	if m.vxlanDeviceLinkIndex == 0 {
		// vxlan device not configured yet. Defer processing updates.
		log.Debug("Wait for Egress-IP VXLAN device to be configured")
		return ErrVxlanDeviceNotConfigured
	}

	if m.routeRules == nil {
		// Create routeRules to manage routing rules.
		// We create routerule inside CompleteDeferredWork to make sure datastore is in sync and all WEP/EgressIPSet updates
		// will be processed before routerule's apply() been called.
		m.routeRules = m.rrGenerator.NewRouteRules(
			4,
			m.tableIndexSet,
			routerule.RulesMatchSrcFWMarkTable,
			routerule.RulesMatchSrcFWMarkTable,
			m.dpConfig.NetlinkTimeout,
			m.opRecorder,
		)
	}

	if m.dirtyEgressIPSet.Len() > 0 {
		// Work out all L2 routes updates.
		m.setL2Routes()
	}

	if m.initialKernelState != nil {
		log.Info("Reading initial kernel state.")
		// Query kernel rules and tables to see what is already in place and can be reused.
		err := m.readInitialKernelState()
		if err != nil {
			log.WithError(err).Info("Couldn't read initial kernel state.")
			// If we can't read the initial state, return now to avoid causing damage.
			return err
		}
	}

	if m.dirtyEgressIPSet.Len() > 0 {
		log.Info("Processing gateway updates.")
		err := m.processGatewayUpdates()
		if err != nil {
			log.WithError(err).Info("Couldn't process gateway updates.")
			lastErr = err
		}
	}

	log.Info("Processing workload updates.")
	err := m.processWorkloadUpdates()
	if err != nil {
		log.WithError(err).Info("Couldn't process workload updates.")
		lastErr = err
	}

	log.Info("Notifying workloads of any terminating gateways.")
	err = m.notifyWorkloadsOfEgressGatewayMaintenanceWindows()
	if err != nil {
		log.WithError(err).Info("Couldn't notify workloads of gateway termination.")
		lastErr = err
	}

	if m.initialKernelState != nil {
		log.Info("Cleaning up any unused initial kernel state.")
		// Cleanup any kernel routes and tables which were not needed for reuse.
		err = m.cleanupInitialKernelState()
		if err != nil {
			log.WithError(err).Info("Couldn't cleanup initial kernel state.")
			lastErr = err
		}
	}

	return lastErr
}

func (m *egressIPManager) readInitialKernelState() error {
	if m.routeRules == nil {
		return errors.New("cannot read rules and tables from kernel during initial read")
	}

	// Read routing rules within the egress manager table range from the kernel.
	m.routeRules.InitFromKernel()
	rules := m.routeRules.GetAllActiveRules()
	ruleTableIndices := set.New()
	for _, rule := range rules {
		nlRule := rule.NetLinkRule()
		r := newEgressRule(nlRule)
		m.initialKernelState.rules[r.srcIP] = r
		ruleTableIndices.Add(r.tableIndex)
	}

	// Read routing tables referenced by a routing rule from the kernel.
	reservedTables := set.New()
	ruleTableIndices.Iter(func(item interface{}) error {
		index := item.(int)
		hopIPs, err := m.getTableNextHops(index)
		if err != nil {
			log.WithError(err).WithField("table", index).Error("failed to get route table targets")
			return nil
		}
		if len(hopIPs) > 0 {
			// Ensure table index isn't in the tableIndexStack, so it won't be used by another workload
			reservedTables.Add(index)
			t := newEgressTable(index, hopIPs)
			m.initialKernelState.tables[t.index] = t
		}
		return nil
	})
	m.removeIndicesFromTableStack(reservedTables)

	log.WithFields(log.Fields{
		"initialKernelState": m.initialKernelState,
	}).Info("Read existing route rules and tables from kernel.")
	return nil
}

func (m *egressIPManager) cleanupInitialKernelState() error {
	if m.routeRules == nil {
		return errors.New("cannot read rules and tables from kernel during initial cleanup")
	}
	defer func() {
		m.initialKernelState = nil
	}()

	// Remove unused rules.
	for _, r := range m.initialKernelState.rules {
		if !r.used {
			log.WithField("rule", *r).Info("Deleting unused route rule")
			m.deleteRouteRule(r.srcIP, r.tableIndex)
		}
	}

	// Remove unused tables.
	for _, t := range m.initialKernelState.tables {
		if !t.used {
			log.WithField("table", *t).Info("Deleting unused route table")
			// This looks odd to create it then delete it, but is necessary since delete needs it to be tracked.
			m.createRouteTable(t.index, t.hopIPs)
			m.deleteRouteTable(t.index)
		}
	}
	return nil
}

// processGatewayUpdates handles all gateway updates. Any route tables which contain next hops for gateways which no
// longer exist are deleted and recreated with new valid hops.
func (m *egressIPManager) processGatewayUpdates() error {
	sortedUpdates := sortStringSet(m.dirtyEgressIPSet)

	var lastErr error
	for _, id := range sortedUpdates {
		gateways, exists := m.ipSetIDToGateways[id]
		if !exists {
			log.WithField("IPSetID", id).Info("Could not find gateways for IPSet, it will be removed.")
			gateways = make(map[string]gateway)
		}
		gatewayIPs := gateways.getIPs()

		// Check if any existing workloads have next hops for deleted gateways.
		var workloadIDs []proto.WorkloadEndpointID
		for workloadID := range m.activeWorkloads {
			workloadIDs = append(workloadIDs, workloadID)
		}
		sort.Slice(workloadIDs, func(i, j int) bool {
			if workloadIDs[i].EndpointId != workloadIDs[j].EndpointId {
				return workloadIDs[i].EndpointId < workloadIDs[j].EndpointId
			}
			if workloadIDs[i].WorkloadId != workloadIDs[j].WorkloadId {
				return workloadIDs[i].WorkloadId < workloadIDs[j].WorkloadId
			}
			return workloadIDs[i].OrchestratorId < workloadIDs[j].OrchestratorId
		})
		for _, workloadID := range workloadIDs {
			workload := m.activeWorkloads[workloadID]
			// Check if this workload uses the current gateways
			if workload.EgressIpSetId == id {
				index, exists := m.workloadToTableIndex[workloadID]
				if !exists {
					lastErr = fmt.Errorf("table index not found for workload with id %s", workloadID)
					continue
				}
				nextHops, exists := m.tableIndexToNextHops[index]
				if !exists {
					lastErr = fmt.Errorf("next hops not found for table with index %d", index)
					continue
				}
				log.WithFields(log.Fields{
					"ipSetID":    id,
					"gateways":   gateways,
					"workloadID": workloadID,
					"workload":   workload,
					"tableIndex": index,
					"nextHops":   nextHops,
				}).Info("Processing gateway update.")

				workloadHasLessHopsThanDesired := (int(workload.EgressMaxNextHops) == 0 && len(nextHops) < len(gateways.getActiveGateways())) ||
					(int(workload.EgressMaxNextHops) > 0 && len(nextHops) < (int(workload.EgressMaxNextHops)) && len(nextHops) < len(gateways.getActiveGateways()))
				workloadHasNonExistentHop := !set.FromArray(gatewayIPs).ContainsAll(set.FromArray(nextHops))
				if workloadHasLessHopsThanDesired || workloadHasNonExistentHop {
					// Delete the old route rules and table as they contain an invalid hop.
					m.deleteWorkloadRuleAndTable(workloadID, workload)

					// Create new route rules and a route table for this workload.
					log.WithFields(log.Fields{
						"ipSetID":     id,
						"workloadIPs": workload.Ipv4Nets,
						"workloadID":  workloadID,
						"tableIndex":  index,
					}).Info("Processing gateway update - recreating route rules and table for workload.")
					err := m.createWorkloadRuleAndTable(workloadID, workload, len(gateways.getActiveGateways()))
					if err != nil {
						lastErr = err
						continue
					}
				}
			}
		}
		// Remove id from dirtyEgressIPSet.
		m.dirtyEgressIPSet.Discard(id)
	}
	return lastErr
}

// processWorkloadUpdates takes WorkLoadEndpoints from state and programs route rules for their CIDR's pointing to an egress route table
// dedicated to that workload. The table will contain maxNextHops ECMP routes if specified, otherwise it will contain an ECMP route
// for every member of the workload's IP set.
// To minimize the effect on existing traffic, route rules and tables discovered from the kernel on initialization will be left in place
// if possible, rather than creating new rules and tables.
func (m *egressIPManager) processWorkloadUpdates() error {
	var lastErr error
	// Handle pending workload endpoint updates.
	var ids []proto.WorkloadEndpointID
	for id := range m.pendingWorkloadUpdates {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].EndpointId != ids[j].EndpointId {
			return ids[i].EndpointId < ids[j].EndpointId
		}
		if ids[i].WorkloadId != ids[j].WorkloadId {
			return ids[i].WorkloadId < ids[j].WorkloadId
		}
		return ids[i].OrchestratorId < ids[j].OrchestratorId
	})

	existingTables := make(map[proto.WorkloadEndpointID]*egressTable)
	var workloadsToUseExistingTable []proto.WorkloadEndpointID
	var workloadsToUseNewTable []proto.WorkloadEndpointID
	if m.initialKernelState != nil {
		log.Info("Processing workloads after restart. Will attempt to reuse existing rules and tables to preserve traffic.")
		// Look for any routing rules and tables which can be reused.
		for _, id := range ids {
			workload := m.pendingWorkloadUpdates[id]
			if workload == nil || workload.EgressIpSetId == "" {
				continue
			}
			gateways, exists := m.ipSetIDToGateways[workload.EgressIpSetId]
			if !exists {
				gateways = make(map[string]gateway)
			}
			activeGatewayIPs := gateways.getActiveGateways().getIPs()
			priority := m.dpConfig.EgressIPRoutingRulePriority
			family := syscall.AF_INET
			mark := int(m.dpConfig.RulesConfig.IptablesMarkEgress)
			numHops := workloadNumHops(int(workload.EgressMaxNextHops), len(activeGatewayIPs))
			_, existingTable, exists := m.reserveFromInitialState(workload.Ipv4Nets, priority, family, mark, numHops, activeGatewayIPs)
			if exists {
				log.WithFields(log.Fields{
					"workloadID": id,
					"table":      *existingTable,
				}).Info("Pre-processing workload - reserving table")
				existingTables[id] = existingTable
				workloadsToUseExistingTable = append(workloadsToUseExistingTable, id)
			} else {
				workloadsToUseNewTable = append(workloadsToUseNewTable, id)
			}
		}

		// Process workloads reusing existing tables first, so that workloads needing new tables can be created in such a way
		// as to even out the distribution of hops across workloads.
		for _, id := range workloadsToUseExistingTable {
			logCtx := log.WithField("workloadID", id)
			workload := m.pendingWorkloadUpdates[id]

			log.WithFields(log.Fields{
				"workloadID":             id,
				"workload":               workload,
				"workload.maxNextHops":   workload.EgressMaxNextHops,
				"workload.egressIPSetID": workload.EgressIpSetId,
			}).Info("Processing workload create.")

			existingTable, exists := existingTables[id]
			if exists {
				logCtx.Info("Processing workload - suitable route rules pointing to a table with active gateway hops were found.")
				err := m.createWorkloadRuleAndTableWithIndex(id, workload, existingTable.hopIPs, existingTable.index)
				if err != nil {
					logCtx.WithError(err).Info("Couldn't create route table and rules for workload.")
					lastErr = err
					continue
				}
			}
			m.activeWorkloads[id] = workload
			delete(m.pendingWorkloadUpdates, id)
		}
	} else {
		workloadsToUseNewTable = append(workloadsToUseNewTable, ids...)
	}

	// Process workloads needing new tables last, to even out the distribution of hops across workloads.
	for _, id := range workloadsToUseNewTable {
		logCtx := log.WithField("workloadID", id)
		workload := m.pendingWorkloadUpdates[id]
		oldWorkload := m.activeWorkloads[id]

		if workload != nil && oldWorkload != nil {
			log.WithFields(log.Fields{
				"workloadID":                id,
				"workload":                  workload,
				"workload.maxNextHops":      workload.EgressMaxNextHops,
				"workload.egressIPSetID":    workload.EgressIpSetId,
				"oldWorkload":               oldWorkload,
				"oldWorkload.maxNextHops":   oldWorkload.EgressMaxNextHops,
				"oldWorkload.egressIPSetID": oldWorkload.EgressIpSetId,
			}).Info("Processing workload update.")
		}

		if workload != nil && oldWorkload == nil {
			log.WithFields(log.Fields{
				"workloadID":             id,
				"workload":               workload,
				"workload.maxNextHops":   workload.EgressMaxNextHops,
				"workload.egressIPSetID": workload.EgressIpSetId,
			}).Info("Processing workload create.")
		}

		if workload == nil && oldWorkload != nil {
			log.WithFields(log.Fields{
				"workloadID":                id,
				"oldWorkload":               oldWorkload,
				"oldWorkload.maxNextHops":   oldWorkload.EgressMaxNextHops,
				"oldWorkload.egressIPSetID": oldWorkload.EgressIpSetId,
			}).Info("Processing workload delete.")
		}

		workloadCreated := workload != nil && oldWorkload == nil
		workloadDeleted := workload == nil && oldWorkload != nil
		workloadChanged := workload != nil && oldWorkload != nil

		workloadDeletedWasUsingEgress := workloadDeleted && oldWorkload.EgressIpSetId != ""
		workloadChangedToStopUsingEgress := workloadChanged && workload.EgressIpSetId == "" && oldWorkload.EgressIpSetId != ""
		workloadChangedToUseDifferentEgress := workloadChanged && workload.EgressIpSetId != "" && oldWorkload.EgressIpSetId != "" &&
			workload.EgressIpSetId != oldWorkload.EgressIpSetId

		if workloadDeletedWasUsingEgress || workloadChangedToStopUsingEgress || workloadChangedToUseDifferentEgress {
			logCtx.Info("Processing workload - workload deleted or no longer using egress gateway.")
			m.deleteWorkloadRuleAndTable(id, oldWorkload)
			delete(m.activeWorkloads, id)
		}

		workloadCreatedUsingEgress := workloadCreated && workload.EgressIpSetId != ""
		workloadChangedToStartUsingEgress := workloadChanged && workload.EgressIpSetId != "" && oldWorkload.EgressIpSetId == ""

		if workloadCreatedUsingEgress || workloadChangedToStartUsingEgress || workloadChangedToUseDifferentEgress {
			gateways, exists := m.ipSetIDToGateways[workload.EgressIpSetId]
			if !exists {
				gateways = make(map[string]gateway)
			}
			activeGatewayIPs := gateways.getActiveGateways().getIPs()

			logCtx.Info("Processing workload - creating new route rules and table.")
			err := m.createWorkloadRuleAndTable(id, workload, len(activeGatewayIPs))
			if err != nil {
				logCtx.WithError(err).Info("Couldn't create route table and rules for workload.")
				lastErr = err
				continue
			}
			m.activeWorkloads[id] = workload
		}

		delete(m.pendingWorkloadUpdates, id)
	}

	return lastErr
}

// Notifies all workloads of maintenance windows on egress gateway pods they're using by annotating the workload pods.
func (m *egressIPManager) notifyWorkloadsOfEgressGatewayMaintenanceWindows() error {
	// Cleanup any orphaned maintenance windows.
	for id := range m.workloadMaintenanceWindows {
		if _, exists := m.activeWorkloads[id]; !exists {
			delete(m.workloadMaintenanceWindows, id)
		}
	}

	for id, workload := range m.activeWorkloads {
		gateways, exists := m.ipSetIDToGateways[workload.EgressIpSetId]
		if !exists {
			log.Debugf("Workload with ID: %s references an empty set of gateways: %s. No notification required.", id, workload.EgressIpSetId)
			continue
		}
		namespace, name, err := parseNameAndNamespace(id.WorkloadId)
		if err != nil {
			return err
		}
		wepids := names.WorkloadEndpointIdentifiers{
			Node:         m.dpConfig.FelixHostname,
			Orchestrator: id.OrchestratorId,
			Endpoint:     id.EndpointId,
			Pod:          name,
		}
		wepName, err := wepids.CalculateWorkloadEndpointName(false)
		if err != nil {
			return err
		}

		index, exists := m.workloadToTableIndex[id]
		if !exists {
			return fmt.Errorf("cannot find table for workload with id %s", id)
		}
		nextHops, exists := m.tableIndexToNextHops[index]
		if !exists {
			return fmt.Errorf("cannot find next hops for table with index %d", index)
		}

		terminatingGatewayHops := gateways.getTerminatingGateways().getFilteredGateways(nextHops)
		latest := terminatingGatewayHops.latestTerminatingGateway()
		existing, exists := m.workloadMaintenanceWindows[id]
		if !exists {
			existing = gateway{}
		}
		m.workloadMaintenanceWindows[id] = latest

		if latest.cidr != "" &&
			!latest.maintenanceStarted.IsZero() &&
			!latest.maintenanceFinished.IsZero() &&
			latest != existing {
			log.WithFields(log.Fields{
				"nextHops":             nextHops,
				"gateways":             gateways,
				"namespace":            namespace,
				"name":                 name,
				"latestTerminatingHop": latest.String(),
			}).Info("Notifying workload of its next hops which are terminating.")
			err = m.statusCallback(
				namespace,
				wepName,
				strings.TrimSuffix(latest.cidr, "/32"),
				latest.maintenanceStarted,
				latest.maintenanceFinished)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Construct a routing rule without table value (matching conditions only) related to a workload.
func (m *egressIPManager) workloadIPToRuleMatchSrcFWMark(workloadIP string) *routerule.Rule {
	cidr := ip.MustParseCIDROrIP(workloadIP)
	return routerule.
		NewRule(4, m.dpConfig.EgressIPRoutingRulePriority).
		MatchSrcAddress(cidr.ToIPNet()).
		MatchFWMark(m.dpConfig.RulesConfig.IptablesMarkEgress)
}

// Construct a full routing rule related to a workload.
func (m *egressIPManager) workloadIPToFullRule(workloadIP string, tableIndex int) *routerule.Rule {
	return m.workloadIPToRuleMatchSrcFWMark(workloadIP).GoToTable(tableIndex)
}

// Set L2 routes for all active EgressIPSet.
func (m *egressIPManager) setL2Routes() {
	gatewayIPs := set.New()
	for _, gateways := range m.ipSetIDToGateways {
		for _, g := range gateways {
			ipString := strings.Split(g.cidr, "/")[0]
			gatewayIPs.Add(ipString)
		}
	}

	// Sort gateways to make L2 target update deterministic.
	sorted := sortStringSet(gatewayIPs)

	var l2routes []routetable.L2Target
	for _, gatewayIP := range sorted {
		l2routes = append(l2routes, routetable.L2Target{
			// remote VTEP mac is generated based on gateway pod ip.
			VTEPMAC: ipStringToMac(gatewayIP),
			GW:      ip.FromString(gatewayIP),
			IP:      ip.FromString(gatewayIP),
		})
	}

	// Set L2 route. If there is no l2route target, old entries will be removed.
	log.WithField("l2routes", l2routes).Info("Egress ip manager sending L2 updates")
	m.l2Table.SetL2Routes(m.vxlanDevice, l2routes)
}

// Set L3 routes for an EgressIPSet.
func (m *egressIPManager) setL3Routes(rTable routeTable, ips set.Set) {
	logCxt := log.WithField("table", rTable.Index())
	var multipath []routetable.NextHop

	// Sort ips to make ECMP route deterministic.
	sorted := sortStringSet(ips)

	for _, element := range sorted {
		ipString := strings.Split(element, "/")[0]
		multipath = append(multipath, routetable.NextHop{
			Gw:        ip.FromString(ipString),
			LinkIndex: m.vxlanDeviceLinkIndex, // we have already acquired a lock for this data further up the call stack
		})
	}

	if len(multipath) > 1 {
		// Set multipath L3 route.
		// Note the interface is InterfaceNone for multipath.
		route := routetable.Target{
			Type:      routetable.TargetTypeVXLAN,
			CIDR:      defaultCidr,
			MultiPath: multipath,
		}
		logCxt.WithField("ecmproute", route).Info("Egress ip manager sending ECMP VXLAN L3 updates")
		rTable.RouteRemove(m.vxlanDevice, defaultCidr)
		rTable.SetRoutes(routetable.InterfaceNone, []routetable.Target{route})
	} else if len(multipath) == 1 {
		// If we send multipath routes with just one path, netlink will program it successfully.
		// However, we will read back a route via netlink with GW set to nexthop GW
		// and len(Multipath) set to 0. To keep route target consistent with netlink route,
		// we should not send a multipath target with just one GW.
		route := routetable.Target{
			Type: routetable.TargetTypeVXLAN,
			CIDR: defaultCidr,
			GW:   multipath[0].Gw,
		}
		logCxt.WithField("route", route).Info("Egress ip manager sending single path VXLAN L3 updates," +
			" may see couple of warnings if an ECMP route was previously programmed")

		// Route table module may report warning of `file exists` on programming route for egress.vxlan device.
		// This is because route table module processes route updates organized by interface names.
		// In this case, default route for egress.calico interface could not be programmed unless
		// the default route linked with InterfaceNone been removed. After couple of failures on processing
		// egress.calico updates, route table module will continue on processing InterfaceNone updates
		// and remove default route (see RouteRemove below).
		// Route updates for egress.vxlan will be successful at next dataplane apply().
		rTable.RouteRemove(routetable.InterfaceNone, defaultCidr)

		rTable.SetRoutes(m.vxlanDevice, []routetable.Target{route})

	} else {
		// Set unreachable route.
		route := routetable.Target{
			Type: routetable.TargetTypeUnreachable,
			CIDR: defaultCidr,
		}

		logCxt.WithField("route", route).Info("Egress ip manager sending unreachable route")
		rTable.RouteRemove(m.vxlanDevice, defaultCidr)
		rTable.SetRoutes(routetable.InterfaceNone, []routetable.Target{route})
	}
}

func (m *egressIPManager) createWorkloadRuleAndTable(workloadID proto.WorkloadEndpointID, workload *proto.WorkloadEndpoint, numHops int) error {
	adjustedNumHops := workloadNumHops(int(workload.EgressMaxNextHops), numHops)
	hopIPs, err := m.determineTableNextHops(workloadID, workload.EgressIpSetId, adjustedNumHops)
	if err != nil {
		return err
	}
	index, err := m.getNextTableIndex()
	if err != nil {
		return err
	}
	return m.createWorkloadRuleAndTableWithIndex(workloadID, workload, hopIPs, index)
}

func (m *egressIPManager) createWorkloadRuleAndTableWithIndex(workloadID proto.WorkloadEndpointID, workload *proto.WorkloadEndpoint, hopIPs []string, index int) error {
	// Create new route rules and a route table for this workload.
	log.WithFields(log.Fields{
		"workloadID":  workloadID,
		"workloadIPs": workload.Ipv4Nets,
		"tableIndex":  index,
		"hopIPs":      hopIPs,
	}).Info("Creating route rules and table for this workload.")
	m.createRouteTable(index, hopIPs)
	m.workloadToTableIndex[workloadID] = index
	for _, srcIP := range workload.Ipv4Nets {
		m.createRouteRule(srcIP, index)
	}
	return nil
}

func (m *egressIPManager) deleteWorkloadRuleAndTable(id proto.WorkloadEndpointID, workload *proto.WorkloadEndpoint) {
	if index, exists := m.workloadToTableIndex[id]; !exists {
		// This can occur if the workload has already been deleted as a result of an IPSet becoming empty, and then a
		// workload being removed in the same batch of updates.
		log.WithField("workloadID", id).Debug("Cannot delete routing table for workload, it has already been deleted.")
	} else {
		for _, ipAddr := range workload.Ipv4Nets {
			m.deleteRouteRule(ipAddr, index)
		}
		m.deleteRouteTable(index)
		delete(m.workloadToTableIndex, id)
	}
}

func (m *egressIPManager) newRouteTable(tableNum int) routeTable {
	return m.rtGenerator.NewRouteTable(
		[]string{"^" + m.vxlanDevice + "$", routetable.InterfaceNone},
		4,
		tableNum,
		true,
		m.dpConfig.NetlinkTimeout,
		nil,
		int(m.dpConfig.DeviceRouteProtocol),
		true,
		m.opRecorder)
}

func (m *egressIPManager) newRouteRule(srcIP string) *routerule.Rule {
	ipAddr := ip.MustParseCIDROrIP(srcIP).ToIPNet()
	return routerule.
		NewRule(4, m.dpConfig.EgressIPRoutingRulePriority).
		MatchSrcAddress(ipAddr).
		MatchFWMark(m.dpConfig.RulesConfig.IptablesMarkEgress)
}

func (m *egressIPManager) getNextTableIndex() (int, error) {
	if m.tableIndexStack.Len() == 0 {
		return 0, ErrInsufficientRouteTables
	}
	index := m.tableIndexStack.Pop().(int)
	log.WithField("index", index).Debug("Popped table index off the stack for table creation.")
	return index, nil
}

func (m *egressIPManager) createRouteTable(index int, hopIPs []string) {
	log.WithFields(log.Fields{
		"index":  index,
		"hopIPs": strings.Join(hopIPs, ","),
	}).Debug("Creating route table.")
	table := m.newRouteTable(index)
	m.setL3Routes(table, set.FromArray(hopIPs))
	m.tableIndexToRouteTable[index] = table
	m.tableIndexToNextHops[index] = hopIPs
}

func (m *egressIPManager) deleteRouteTable(index int) {
	log.WithField("index", index).Debug("Deleting route table.")
	table, exists := m.tableIndexToRouteTable[index]
	if !exists {
		log.WithField("tableIndex", index).Debug("Cannot delete routing table, it does not exist.")
		return
	}
	table.RouteRemove(routetable.InterfaceNone, defaultCidr)
	table.RouteRemove(m.vxlanDevice, defaultCidr)
	delete(m.tableIndexToNextHops, index)
	// Don't remove the entry from m.tableIndexToRouteTable, it is needed in GetRouteTableSyncers()
	// so the dataplane knows which route tables to sync. If we remove it, the dataplane will not
	// be able to remove the routes.
	log.WithField("index", index).Debug("Pushing table index to the stack after table deletion.")
	m.tableIndexStack.Push(index)
}

func (m *egressIPManager) createRouteRule(srcIP string, tableIndex int) {
	log.WithFields(log.Fields{
		"srcIP":      srcIP,
		"tableIndex": tableIndex,
	}).Debug("Creating route rule.")
	rule := m.newRouteRule(srcIP).GoToTable(tableIndex)
	m.routeRules.SetRule(rule)
}

func (m *egressIPManager) deleteRouteRule(srcIP string, tableIndex int) *routerule.Rule {
	log.WithField("srcIP", srcIP).Debug("Deleting route rule.")
	ipAddr := ip.MustParseCIDROrIP(srcIP).ToIPNet()
	rule := routerule.NewRule(4, m.dpConfig.EgressIPRoutingRulePriority).
		MatchSrcAddress(ipAddr).
		MatchFWMark(m.dpConfig.RulesConfig.IptablesMarkEgress).
		GoToTable(tableIndex)
	m.routeRules.RemoveRule(rule)
	return rule
}

func (m *egressIPManager) getTableNextHops(index int) ([]string, error) {
	table := m.newRouteTable(index)
	// get targets for both possible interface names
	vxlanTargets, err := table.ReadRoutesFromKernel(m.vxlanDevice)
	if err != nil {
		return nil, err
	}
	noneTargets, err := table.ReadRoutesFromKernel(routetable.InterfaceNone)
	if err != nil {
		return nil, err
	}
	var hopIPs []string
	for _, vxlanTarget := range vxlanTargets {
		if vxlanTarget.GW != nil {
			hopIPs = append(hopIPs, vxlanTarget.GW.String())
		} else {
			for _, hop := range vxlanTarget.MultiPath {
				hopIPs = append(hopIPs, hop.Gw.String())
			}
		}
	}
	for _, noneTarget := range noneTargets {
		if noneTarget.GW != nil {
			hopIPs = append(hopIPs, noneTarget.GW.String())
		} else {
			for _, hop := range noneTarget.MultiPath {
				hopIPs = append(hopIPs, hop.Gw.String())
			}
		}
	}
	return hopIPs, nil
}

func (m *egressIPManager) GetRouteTableSyncers() []routeTableSyncer {
	rts := []routeTableSyncer{m.l2Table.(routeTableSyncer)}
	for _, t := range m.tableIndexToRouteTable {
		rts = append(rts, t.(routeTableSyncer))
	}

	return rts
}

func (m *egressIPManager) GetRouteRules() []routeRules {
	if m.routeRules != nil {
		return []routeRules{m.routeRules}
	}
	return nil
}

// ipStringToMac defines how an egress gateway pod's MAC is generated
func ipStringToMac(s string) net.HardwareAddr {
	ipAddr := ip.FromString(s)
	if ipAddr == nil {
		log.Errorf("could not parse ip from string: %s", s)
	}
	netIP := ipAddr.AsNetIP()
	// Any MAC address that has the values 2, 3, 6, 7, A, B, E, or F
	// as the second most significant nibble are locally administered.
	hw := net.HardwareAddr(append([]byte{0xa2, 0x2a}, netIP...))
	return hw
}

func (m *egressIPManager) KeepVXLANDeviceInSync(mtu int, wait time.Duration) {
	log.Info("egress ip VXLAN tunnel device thread started.")
	logNextTime := true
	for {
		err := m.configureVXLANDevice(mtu)
		if err != nil {
			log.WithError(err).Warn("Failed to configure egress ip VXLAN tunnel device, retrying...")
			time.Sleep(1 * time.Second)
			logNextTime = true
			continue
		}

		// src_valid_mark must be enabled for RPF to accurately check returning egress packets coming through egress.calico
		err = writeProcSys(fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/src_valid_mark", m.vxlanDevice), "1")
		if err != nil {
			log.WithError(err).Warnf("Failed to enable src_valid_mark system flag for device '%s", m.vxlanDevice)
			logNextTime = true
			goto next
		}

		if logNextTime {
			log.Info("Egress ip VXLAN tunnel device configured.")
			logNextTime = false
		}
	next:
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
				log.Debugf("Found parent interface: %#v", link)
				return link, nil
			}
		}
	}
	return nil, fmt.Errorf("unable to find parent interface with address %s", m.NodeIP.String())
}

// configureVXLANDevice ensures the VXLAN tunnel device is up and configured correctly.
func (m *egressIPManager) configureVXLANDevice(mtu int) error {
	logCxt := log.WithFields(log.Fields{"device": m.vxlanDevice})
	logCxt.Debug("Configuring egress ip VXLAN tunnel device")
	parent, err := m.getParentInterface()
	if err != nil {
		return err
	}

	// Egress ip vxlan device does not need to have tunnel address.
	// We generate a predictable MAC here that we can reproduce here https://github.com/tigera/egress-gateway/blob/18133f0b37119b3463cd5af75539e27fec69b16b/util/net/mac.go#L20
	// in an identical manner.
	mac, err := hardwareAddrForNode(m.dpConfig.Hostname)
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

	// Disable checksum offload.  Otherwise we end up with invalid checksums when a
	// packet is encapped for egress gateway and then double-encapped for the regular
	// cluster IP-IP or VXLAN overlay.
	if err := m.disableChecksumOffload(m.vxlanDevice); err != nil {
		return fmt.Errorf("failed to disable checksum offload: %s", err)
	}

	// And the device is up.
	if err := m.nlHandle.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set interface up: %s", err)
	}

	// Save link index
	m.updateLock.Lock()
	defer m.updateLock.Unlock()
	m.vxlanDeviceLinkIndex = attrs.Index
	m.unblockingUpdateOccurred = true

	return nil
}

func (m *egressIPManager) determineTableNextHops(workloadID proto.WorkloadEndpointID, ipSetID string, maxNextHops int) ([]string, error) {
	members, exists := m.ipSetIDToGateways[ipSetID]
	if !exists {
		log.Infof("Workload with ID: %s references an empty set of gateways: %s. Setting its next hops to none.", workloadID, ipSetID)
		return nil, nil
	}
	gatewayIPs := members.getActiveGateways().getIPs()
	usage := usageMap(workloadID, gatewayIPs, m.tableIndexToNextHops)
	var freqs []int
	for n := range usage {
		freqs = append(freqs, n)
	}
	sort.Ints(freqs)

	var hops []string
	for _, n := range freqs {
		nHops := usage[n]
		m.hopRand.Shuffle(len(nHops), func(i, j int) { nHops[i], nHops[j] = nHops[j], nHops[i] })
		hops = append(hops, nHops...)
	}
	numHops := workloadNumHops(maxNextHops, len(gatewayIPs))
	index := numHops
	if len(hops) < numHops {
		index = len(hops)
	}
	return hops[:index], nil
}

// reserveFromInitialState searches the rules and tables found from the kernel, and looks for route rules for all the
// workload's IP addresses which point to a route table with the correct number of hops, which are currently not
// terminating.
func (m *egressIPManager) reserveFromInitialState(srcIPs []string, priority int, family int, mark int, numHops int, activeGateways []string) ([]*egressRule, *egressTable, bool) {
	state := m.initialKernelState
	if state == nil {
		return nil, nil, false
	}

	log.WithFields(log.Fields{
		"srcIPs":             srcIPs,
		"priority":           priority,
		"family":             family,
		"mark":               mark,
		"numHops":            numHops,
		"activeGateways":     activeGateways,
		"initialKernelState": state,
	}).Info("Looking for matching rule and table to reuse.")

	// Check for unused matching rules.
	var rules []*egressRule
	var index int
	for i, srcIP := range srcIPs {
		ipAddr := ip.MustParseCIDROrIP(srcIP).Addr()
		rule, exists := state.rules[ipAddr.String()]
		if !exists || rule.used {
			return nil, nil, false
		}
		if rule.priority != priority || rule.family != family || rule.mark != mark {
			return nil, nil, false
		}
		if i == 0 {
			index = rule.tableIndex
		} else {
			if index != rule.tableIndex {
				// Multiple rules for the workload point to different tables.
				return nil, nil, false
			}
		}
		rules = append(rules, rule)
	}

	// Check for unused matching table.
	table, exists := state.tables[index]
	if !exists || table.used {
		return nil, nil, false
	}
	if len(table.hopIPs) != numHops {
		return nil, nil, false
	}
	if !set.FromArray(activeGateways).ContainsAll(set.FromArray(table.hopIPs)) {
		return nil, nil, false
	}

	// Mark them as used.
	table.used = true
	for _, r := range rules {
		r.used = true
	}

	return rules, table, true
}

func (m *egressIPManager) removeIndicesFromTableStack(indices set.Set) {
	s := stack.New()
	// Pop items off the stack until the index to be removed has been popped.
	for {
		item := m.tableIndexStack.Pop()
		if item == nil {
			break
		}
		i := item.(int)
		if !indices.Contains(i) {
			s.Push(i)
		}
	}
	// Push all items back on, except the indices to be removed.
	for {
		item := s.Pop()
		if item == nil {
			break
		}
		i := item.(int)
		m.tableIndexStack.Push(i)
	}
}

// hardwareAddrForNode deterministically creates a unique hardware address from a hostname.
// IMPORTANT: an egress gateway pod needs to perform an identical operation when programming its own L2 routes to this node,
// as shown here https://github.com/tigera/egress-gateway/blob/18133f0b37119b3463cd5af75539e27fec69b16b/util/net/mac.go#L20 (change with caution).
func hardwareAddrForNode(hostname string) (net.HardwareAddr, error) {
	hasher := sha1.New()
	_, err := hasher.Write([]byte(hostname))
	if err != nil {
		return nil, err
	}
	sha := hasher.Sum(nil)
	hw := net.HardwareAddr(append([]byte("f"), sha[0:5]...))

	return hw, nil
}

func workloadNumHops(egressMaxNextHops int, ipSetSize int) int {
	// egressMaxNextHops set to 0 on a workload indicates it should use all hops
	if egressMaxNextHops == 0 {
		return ipSetSize
	}
	// egressMaxNextHops set to larger than the size of the IPSet could indicate a misconfiguration, or else the deployment has been scaled
	// down since the wl was created. Either way, default to the size of the IPSet.
	if egressMaxNextHops > ipSetSize {
		return ipSetSize
	}
	return egressMaxNextHops
}

// usageMap returns a map from the number of workloads using the hop to a slice of the hops
func usageMap(workloadID proto.WorkloadEndpointID, gatewayIPs []string, nextHopsMap map[int][]string) map[int][]string {
	// calculate the number of wl pods referencing each gw pod.
	gwPodRefs := make(map[string]int)
	for _, ipAddr := range gatewayIPs {
		gwPodRefs[ipAddr] = 0
	}
	for _, wlHops := range nextHopsMap {
		for _, hop := range wlHops {
			_, exists := gwPodRefs[hop]
			if exists {
				gwPodRefs[hop] = gwPodRefs[hop] + 1
			}
		}
	}
	// calculate the reverse-mapping, i.e. the mapping from reference count to the gw pods with that number of refs.
	usage := make(map[int][]string)
	for hop, n := range gwPodRefs {
		usage[n] = append(usage[n], hop)
	}

	// sort hops slices
	for n := range usage {
		sort.Strings(usage[n])
	}

	log.WithFields(log.Fields{
		"gatewayIPs":           gatewayIPs,
		"tableIndexToNextHops": nextHopsMap,
		"gwPodRefs":            gwPodRefs,
		"usage":                usage,
	}).Infof("Calculated egress hop usage for workload with id: %s.", workloadID)

	return usage
}

func parseMember(memberStr string) (gateway, error) {
	var cidr string
	maintenanceStarted := time.Time{}
	maintenanceFinished := time.Time{}

	a := strings.Split(memberStr, ",")
	if len(a) == 0 || len(a) > 3 {
		return gateway{}, fmt.Errorf("error parsing member str, expected \"cidr,maintenanceStartedTimestamp,maintenanceFinishedTimestamp\" but got: %s", memberStr)
	}

	cidr = a[0]
	if len(a) == 3 {
		if err := maintenanceStarted.UnmarshalText([]byte(strings.ToUpper(a[1]))); err != nil {
			log.WithField("memberStr", memberStr).Warn("unable to parse maintenance started timestamp from member str, defaulting to zero value.")
		}
		if err := maintenanceFinished.UnmarshalText([]byte(strings.ToUpper(a[2]))); err != nil {
			log.WithField("memberStr", memberStr).Warn("unable to parse maintenance finished timestamp from member str, defaulting to zero value.")
		}
	}

	return gateway{
		cidr:                cidr,
		maintenanceStarted:  maintenanceStarted,
		maintenanceFinished: maintenanceFinished,
	}, nil
}

func parseNameAndNamespace(wlId string) (string, string, error) {
	parts := strings.Split(wlId, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("could not parse name and namespace from workload id: %s", wlId)
	}
	return parts[0], parts[1], nil
}

func sortStringSet(s set.Set) []string {
	var sorted []string
	s.Iter(func(item interface{}) error {
		sorted = append(sorted, item.(string))
		return nil
	})

	sort.Slice(sorted, func(p, q int) bool {
		return sorted[p] < sorted[q]
	})
	return sorted
}

func sortIntSet(s set.Set) []int {
	var sorted []int
	s.Iter(func(item interface{}) error {
		sorted = append(sorted, item.(int))
		return nil
	})
	sort.Slice(sorted, func(p, q int) bool {
		return sorted[p] < sorted[q]
	})
	return sorted
}
