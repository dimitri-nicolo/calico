// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/felix/aws"

	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/ipam"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

// awsIPManager tries to provision secondary ENIs and IP addresses in the AWS fabric for any local pods that are
// in an IP pool with an associated AWS subnet.  The work of attaching ENIs and IP addresses is done by a
// background instance of aws.SecondaryIfaceProvisioner.  The work to configure the local dataplane is done
// by this object.
//
// For thread safety, the aws.SecondaryIfaceProvisioner sends its responses via a channel that is read by the
// main loop in int_dataplane.go.
type awsIPManager struct {
	// Indexes of data we've learned from the datastore.

	poolsByID                 map[string]*proto.IPAMPool
	poolIDsBySubnetID         map[string]set.Set
	localAWSRoutesByDst       map[ip.CIDR]*proto.RouteUpdate
	localRouteDestsBySubnetID map[string]set.Set /*ip.CIDR*/
	awsResyncNeeded           bool

	// ifaceProvisioner manages the AWS fabric resources.  It runs in the background to decouple AWS fabric updates
	// from the main thread.  We send it datastore snapshots; in return, it sends back SecondaryIfaceState objects
	// telling us what state the AWS fabric is in.
	ifaceProvisioner        *aws.SecondaryIfaceProvisioner
	ifaceProvisionerStarted bool

	// awsState is the most recent update we've got from the background thread telling us what state it thinks
	// the AWS fabric should be in. <nil> means "don't know", i.e. we're not ready to touch the dataplane yet.
	awsState *aws.IfaceState

	// Dataplane state.

	routeTableIndexByIfaceName map[string]int
	freeRouteTableIndexes      []int
	routeTablesByTable         map[int]routeTable
	routeTablesByIfaceName     map[string]routeTable
	routeRules                 *routerule.RouteRules
	routeRulesInDataplane      map[awsRuleKey]*routerule.Rule
	dataplaneResyncNeeded      bool
	primaryIfaceMTU            int
	dpConfig                   Config
	expectedPrimaryIPs         map[string]string

	opRecorder logutils.OpRecorder
}

// awsRuleKey is a hashable struct containing the salient aspects of the routing rules that we need to program.
type awsRuleKey struct {
	addr           ip.Addr
	routingTableID int
}

func NewAWSSubnetManager(
	healthAgg *health.HealthAggregator,
	ipamClient ipam.Interface,
	k8sClient *kubernetes.Clientset,
	nodeName string,
	routeTableIndexes []int,
	dpConfig Config,
	opRecorder logutils.OpRecorder,
) *awsIPManager {
	logrus.WithFields(logrus.Fields{
		"nodeName":    nodeName,
		"routeTables": routeTableIndexes,
	}).Info("Creating AWS subnet manager.")
	rules, err := routerule.New(
		4,
		101,
		set.FromArray(routeTableIndexes),
		routerule.RulesMatchSrcFWMarkTable,
		routerule.RulesMatchSrcFWMark,
		dpConfig.NetlinkTimeout,
		func() (routerule.HandleIface, error) {
			return netlink.NewHandle(syscall.NETLINK_ROUTE)
		},
		opRecorder,
	)
	if err != nil {
		logrus.WithError(err).Panic("Failed to init routing rules manager.")
	}

	sm := &awsIPManager{
		poolsByID:                 map[string]*proto.IPAMPool{},
		poolIDsBySubnetID:         map[string]set.Set{},
		localAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
		localRouteDestsBySubnetID: map[string]set.Set{},

		routeTableIndexByIfaceName: map[string]int{},
		freeRouteTableIndexes:      routeTableIndexes,
		routeTablesByIfaceName:     map[string]routeTable{},
		routeTablesByTable:         map[int]routeTable{},
		expectedPrimaryIPs:         map[string]string{},

		routeRules:            rules,
		routeRulesInDataplane: map[awsRuleKey]*routerule.Rule{},
		dpConfig:              dpConfig,
		opRecorder:            opRecorder,

		ifaceProvisioner: aws.NewSecondaryIfaceProvisioner(
			healthAgg,
			ipamClient,
			k8sClient,
			nodeName,
			dpConfig.AWSRequestTimeout,
		),
	}
	sm.queueAWSResync("first run")
	return sm
}

func (a *awsIPManager) ResponseC() <-chan *aws.IfaceState {
	return a.ifaceProvisioner.ResponseC()
}

func (a *awsIPManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.IPAMPoolUpdate:
		a.onPoolUpdate(msg.Id, msg.Pool)
	case *proto.IPAMPoolRemove:
		a.onPoolUpdate(msg.Id, nil)
	case *proto.RouteUpdate:
		a.onRouteUpdate(ip.MustParseCIDROrIP(msg.Dst), msg)
	case *proto.RouteRemove:
		a.onRouteUpdate(ip.MustParseCIDROrIP(msg.Dst), nil)
	case *ifaceUpdate:
		logrus.WithField("update", msg).Debug("Interface state changed.")
		if _, ok := a.expectedPrimaryIPs[msg.Name]; ok {
			a.queueDataplaneResync("Interface changed state")
		}
	case *ifaceAddrsUpdate:
		logrus.WithField("update", msg).Debug("Interface addrs changed.")
		if expAddr, ok := a.expectedPrimaryIPs[msg.Name]; ok && msg.Addrs != nil {
			// This is an interface that we care about.  Check if the address it has corresponds with what we want.
			seenExpected := false
			seenUnexpected := false
			msg.Addrs.Iter(func(item interface{}) error {
				addrStr := item.(string)
				if strings.Contains(addrStr, ":") {
					return nil // Ignore IPv6
				}
				if expAddr == addrStr {
					seenExpected = true
				} else {
					seenUnexpected = true
				}
				return nil
			})
			if !seenExpected || seenUnexpected {
				a.queueDataplaneResync("IPs out of sync on a secondary interface " + msg.Name)
			}
		}
	}
}

func (a *awsIPManager) OnSecondaryIfaceStateUpdate(msg *aws.IfaceState) {
	logrus.WithField("awsState", msg).Debug("Received AWS state update.")
	a.queueDataplaneResync("AWS fabric updated")
	a.awsState = msg
}

func (a *awsIPManager) onPoolUpdate(id string, pool *proto.IPAMPool) {
	// Update the index from subnet ID to pool ID.  We do this first so we can look up the
	// old version of the pool (if any).
	oldSubnetID := ""
	newSubnetID := ""
	if oldPool := a.poolsByID[id]; oldPool != nil {
		oldSubnetID = oldPool.AwsSubnetId
	}
	if pool != nil {
		newSubnetID = pool.AwsSubnetId
	}
	if oldSubnetID != "" && oldSubnetID != newSubnetID {
		// Old AWS subnet is no longer correct. clean up the index.
		logrus.WithFields(logrus.Fields{
			"oldSubnet": oldSubnetID,
			"newSubnet": newSubnetID,
			"pool":      id,
		}).Info("IP pool no longer associated with AWS subnet.")
		a.poolIDsBySubnetID[oldSubnetID].Discard(id)
		if a.poolIDsBySubnetID[oldSubnetID].Len() == 0 {
			delete(a.poolIDsBySubnetID, oldSubnetID)
		}
	}
	if newSubnetID != "" && oldSubnetID != newSubnetID {
		logrus.WithFields(logrus.Fields{
			"oldSubnet": oldSubnetID,
			"newSubnet": newSubnetID,
			"pool":      id,
		}).Info("IP pool now associated with AWS subnet.")
		if _, ok := a.poolIDsBySubnetID[newSubnetID]; !ok {
			a.poolIDsBySubnetID[newSubnetID] = set.New()
		}
		a.poolIDsBySubnetID[newSubnetID].Add(id)
	}

	// Store off the pool update itself.
	if pool == nil {
		delete(a.poolsByID, id)
	} else {
		a.poolsByID[id] = pool
	}
	a.queueAWSResync("IP pool updated")
}

func (a *awsIPManager) onRouteUpdate(dst ip.CIDR, route *proto.RouteUpdate) {
	if route != nil && !route.LocalWorkload {
		route = nil
	}
	if route != nil && route.AwsSubnetId == "" {
		route = nil
	}

	// Update the index from subnet ID to route dest.  We do this first so we can look up the
	// old version of the route (if any).
	oldSubnetID := ""
	newSubnetID := ""

	if oldRoute := a.localAWSRoutesByDst[dst]; oldRoute != nil {
		oldSubnetID = oldRoute.AwsSubnetId
	}
	if route != nil {
		newSubnetID = route.AwsSubnetId
	}

	if oldSubnetID != "" && oldSubnetID != newSubnetID {
		// Old AWS subnet is no longer correct. clean up the index.
		a.localRouteDestsBySubnetID[oldSubnetID].Discard(dst)
		if a.localRouteDestsBySubnetID[oldSubnetID].Len() == 0 {
			delete(a.localRouteDestsBySubnetID, oldSubnetID)
		}
		a.queueAWSResync("route subnet changed")
	}
	if newSubnetID != "" && oldSubnetID != newSubnetID {
		if _, ok := a.localRouteDestsBySubnetID[newSubnetID]; !ok {
			a.localRouteDestsBySubnetID[newSubnetID] = set.New()
		}
		a.localRouteDestsBySubnetID[newSubnetID].Add(dst)
		a.queueAWSResync("route subnet added")
	}

	// Save off the route itself.
	if route == nil {
		if _, ok := a.localAWSRoutesByDst[dst]; !ok {
			return // Not a route we were tracking.
		}
		a.queueAWSResync("route deleted")
		delete(a.localAWSRoutesByDst, dst)
	} else {
		a.localAWSRoutesByDst[dst] = route
		a.queueAWSResync("route updated")
	}
}

func (a *awsIPManager) queueAWSResync(reason string) {
	if a.awsResyncNeeded {
		return
	}
	logrus.WithField("reason", reason).Debug("AWS resync needed")
	a.awsResyncNeeded = true
}

func (a *awsIPManager) queueDataplaneResync(reason string) {
	if a.dataplaneResyncNeeded {
		return
	}
	logrus.WithField("reason", reason).Debug("Dataplane resync needed")
	a.dataplaneResyncNeeded = true
}

func (a *awsIPManager) CompleteDeferredWork() error {
	if !a.ifaceProvisionerStarted {
		a.ifaceProvisioner.Start(context.Background())
		a.ifaceProvisionerStarted = true
	}
	if a.awsResyncNeeded {
		// Datastore has been updated, send a new snapshot to the background thread.  It will configure the AWS
		// fabric appropriately and then send us a SecondaryIfaceState.
		ds := aws.DatastoreState{
			LocalAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
			LocalRouteDestsBySubnetID: map[string]set.Set{},
			PoolIDsBySubnetID:         map[string]set.Set{},
		}
		for k, v := range a.localAWSRoutesByDst {
			// Shallow copy is fine, we always get a fresh route update from the datastore.
			ds.LocalAWSRoutesByDst[k] = v
		}
		for k, v := range a.localRouteDestsBySubnetID {
			ds.LocalRouteDestsBySubnetID[k] = v.Copy()
		}
		for k, v := range a.poolIDsBySubnetID {
			ds.PoolIDsBySubnetID[k] = v
		}
		a.ifaceProvisioner.OnDatastoreUpdate(ds)
		a.awsResyncNeeded = false
	}

	if a.dataplaneResyncNeeded {
		err := a.resyncWithDataplane()
		if err != nil {
			return err
		}
		a.dataplaneResyncNeeded = false
	}

	// TODO update k8s Node with capacities

	return nil
}

func (a *awsIPManager) resyncWithDataplane() error {
	if a.awsState == nil {
		logrus.Debug("No AWS information yet, not syncing dataplane.")
		return nil
	}

	// Find all the local NICs and match them up with AWS NICs.
	ifaces, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("failed to load local interfaces: %w", err)
	}
	activeRules := set.New() /* awsRuleKey */
	activeIfaceNames := set.New()
	var finalErr error

	if a.primaryIfaceMTU == 0 {
		mtu, err := a.findPrimaryInterfaceMTU(ifaces)
		if err != nil {
			return err
		}
		a.primaryIfaceMTU = mtu
	}

	for _, iface := range ifaces {
		// Skip NICs that don't match anything in AWS.
		mac := iface.Attrs().HardwareAddr.String()
		awsNIC, awsNICExists := a.awsState.SecondaryNICsByMAC[mac]
		if !awsNICExists {
			continue
		}
		ifaceName := iface.Attrs().Name
		logrus.WithFields(logrus.Fields{
			"mac":      mac,
			"name":     ifaceName,
			"awsNICID": awsNIC.ID,
		}).Debug("Matched local NIC with AWS NIC.")
		activeIfaceNames.Add(ifaceName)

		// Enable the NIC and configure its IPs.
		priAddrStr := awsNIC.PrimaryIPv4Addr.String()
		a.expectedPrimaryIPs[ifaceName] = priAddrStr
		err := a.configureNIC(iface, ifaceName, priAddrStr)
		if err != nil {
			finalErr = err
		}

		// For each IP assigned to the NIC, we'll add a routing rule that sends traffic _from_ that IP to
		// a dedicated routing table for the NIC.
		routingTableID := a.getOrAllocRoutingTableID(ifaceName)

		// Program routes into the NIC-specific routing table.
		rt := a.getOrAllocRoutingTable(routingTableID, ifaceName)
		a.programIfaceRoutes(rt, ifaceName)

		// Accumulate routing rules for all the active IPs.
		a.addIfaceActiveRules(activeRules, awsNIC, routingTableID)
	}

	// Scan for entries in expectedPrimaryIPs that are no longer needed.
	a.cleanUpPrimaryIPs(activeIfaceNames)

	// Scan for routing tables that are no longer needed.
	a.cleanUpRoutingTables(activeIfaceNames)

	// Queue up delta updates to add/remove routing rules.
	a.updateRouteRules(activeRules)

	return finalErr
}

var (
	errPrimaryMTUNotFound  = errors.New("failed to find primary interface MTU")
	errPrimaryIfaceZeroMTU = errors.New("primary interface had 0 MTU")
)

func (a *awsIPManager) findPrimaryInterfaceMTU(ifaces []netlink.Link) (int, error) {
	for _, iface := range ifaces {
		mac := iface.Attrs().HardwareAddr.String()
		if mac == a.awsState.PrimaryNIC.MAC.String() {
			// Found the primary interface.
			if iface.Attrs().MTU == 0 { // defensive
				return 0, errPrimaryIfaceZeroMTU
			}
			return iface.Attrs().MTU, nil
		}
	}
	return 0, errPrimaryMTUNotFound
}

func (a *awsIPManager) cleanUpPrimaryIPs(matchedNICs set.Set) {
	if matchedNICs.Len() != len(a.expectedPrimaryIPs) {
		// Clean up primary IPs of interfaces that no longer exist.
		for iface := range a.expectedPrimaryIPs {
			if matchedNICs.Contains(iface) {
				continue
			}
			delete(a.expectedPrimaryIPs, iface)
		}
	}
}

// configureNIC Brings the given NIC up and ensures its has the expected IP assigned.
func (a *awsIPManager) configureNIC(iface netlink.Link, ifaceName string, primaryIPStr string) error {
	if iface.Attrs().MTU != a.primaryIfaceMTU {
		// Set the MTU on the link to match the MTU of the primary ENI.  This ensures that we don't flap the
		// detected host MTU by bringing up the new NIC.
		err := netlink.LinkSetMTU(iface, a.primaryIfaceMTU)
		if err != nil {
			logrus.WithError(err).WithField("name", ifaceName).Error("Failed to set secondary ENI MTU.")
			return err
		}
	}

	if iface.Attrs().OperState != netlink.OperUp {
		err := netlink.LinkSetUp(iface)
		if err != nil {
			logrus.WithError(err).WithField("name", ifaceName).Error("Failed to set secondary ENI MTU 'up'")
			return err
		}
	}

	// Make sure the interface has its primary IP.  This is needed for ARP to work.
	addrs, err := netlink.AddrList(iface, netlink.FAMILY_V4)
	if err != nil {
		logrus.WithError(err).WithField("name", ifaceName).Error("Failed to query interface addrs.")
		return err
	}

	foundPrimaryIP := false
	newAddr, err := netlink.ParseAddr(primaryIPStr + "/" + fmt.Sprint(a.awsState.SubnetCIDR.Prefix()))
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"name": ifaceName,
			"addr": primaryIPStr,
		}).Error("Failed to parse address.")
	}
	newAddr.Scope = int(netlink.SCOPE_LINK)

	for _, a := range addrs {
		if a.Equal(*newAddr) {
			foundPrimaryIP = true
			continue
		}

		// Unexpected address.
		err := netlink.AddrDel(iface, &a)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"name": ifaceName,
				"addr": a,
			}).Error("Failed to clean up old address.")
		}
	}

	if foundPrimaryIP {
		return nil
	}

	err = netlink.AddrAdd(iface, newAddr)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"name": ifaceName,
			"addr": newAddr,
		}).Error("Failed to add new primary IP to secondary interface.")
		return err
	} else {
		logrus.WithError(err).WithFields(logrus.Fields{
			"name": ifaceName,
			"addr": newAddr,
		}).Info("Added primary address to secondary ENI.")
	}

	return nil
}

// addIfaceActiveRules awsRuleKey values to activeRules according to the secondary IPs of the AWS NIC.
func (a *awsIPManager) addIfaceActiveRules(activeRules set.Set, awsNIC aws.Iface, routingTableID int) {
	for _, privateIP := range awsNIC.SecondaryIPv4Addrs {
		logrus.WithFields(logrus.Fields{"addr": privateIP, "rtID": routingTableID}).Debug("Adding routing rule.")
		activeRules.Add(awsRuleKey{
			addr:           privateIP,
			routingTableID: routingTableID,
		})
	}
}

// programIfaceRoutes updates the routing table for the given interface with the correct routes.
func (a *awsIPManager) programIfaceRoutes(rt routeTable, ifaceName string) {
	// Add a default route via the AWS subnet's gateway.  This is how traffic to the outside world gets
	// routed properly.
	routes := []routetable.Target{
		{
			Type: routetable.TargetTypeGlobalUnicast,
			CIDR: a.awsState.GatewayAddr.AsCIDR(),
		},
		{
			Type: routetable.TargetTypeGlobalUnicast,
			CIDR: ip.MustParseCIDROrIP("0.0.0.0/0"),
			GW:   a.awsState.GatewayAddr,
		},
	}
	rt.SetRoutes(ifaceName, routes)

	// Add narrower routes for Calico IP pools that throw the packet back to the main routing tables.
	// this is required to make RPF checks pass when traffic arrives from a Calico tunnel going to an
	// AWS-networked pod.
	var noIFRoutes []routetable.Target
	for _, pool := range a.poolsByID {
		if !pool.Masquerade {
			// Assuming that non-masquerade pools are reachable over the main network.
			// Disabled non-Masquerade rules are often used to mark an external CIDR as "reachable without
			// SNAT".
			continue
		}
		noIFRoutes = append(noIFRoutes, routetable.Target{
			Type: routetable.TargetTypeThrow,
			CIDR: ip.MustParseCIDROrIP(pool.Cidr),
		})
	}
	rt.SetRoutes(routetable.InterfaceNone, noIFRoutes)

}

// cleanUpRoutingTables scans routeTableIndexByIfaceName for routing tables that are no longer needed (i.e. no
// longer appear in activeIfaceNames and releases them.
func (a *awsIPManager) cleanUpRoutingTables(activeIfaceNames set.Set) {
	for ifaceName, idx := range a.routeTableIndexByIfaceName {
		if activeIfaceNames.Contains(ifaceName) {
			continue // NIC is known to AWS and the local dataplane.  All good.
		}
		if _, ok := a.routeTablesByIfaceName[ifaceName]; !ok {
			continue // RouteTable has already been flushed.
		}

		// NIC must have existed before but it no longer does.  Flush any routes from its routing table.
		rt := a.routeTablesByTable[idx]
		rt.SetRoutes(ifaceName, nil)
		rt.SetRoutes(routetable.InterfaceNone, nil)

		// Only delete from the a.routeTablesByIfaceName map.  This means that the routing table will live
		// on in a.routeTableIndexByIfaceName until we reuse its index.  We want the table to live on so that
		// it has a chance to actually apply the flush.  We use a LIFO queue when allocating table indexes so
		// the routeing table will be overwritten as soon as a new interface is added.
		delete(a.routeTablesByIfaceName, ifaceName)
		// Free the index so it can be reused.
		a.releaseRoutingTableID(ifaceName)
	}
}

// updateRouteRules calculates route rule deltas between the active rules and the set of rules that we've
// previously programmed.  It sends those to the RouteRules instance.
func (a *awsIPManager) updateRouteRules(activeRuleKeys set.Set /* awsRulesKey */) {
	for k, r := range a.routeRulesInDataplane {
		if activeRuleKeys.Contains(k) {
			continue // Route was present and still wanted; nothing to do.
		}
		// Route no longer wanted, clean it up.
		a.routeRules.RemoveRule(r)
		delete(a.routeRulesInDataplane, k)
	}
	activeRuleKeys.Iter(func(item interface{}) error {
		k := item.(awsRuleKey)
		if _, ok := a.routeRulesInDataplane[k]; ok {
			return nil // Route already present.  Nothing to do.
		}
		rule := routerule.
			NewRule(4, 101).
			MatchSrcAddress(k.addr.AsCIDR().ToIPNet()).
			GoToTable(k.routingTableID)
		a.routeRules.SetRule(rule)
		a.routeRulesInDataplane[k] = rule
		return nil
	})
}

func (a *awsIPManager) getOrAllocRoutingTable(tableIndex int, ifaceName string) routeTable {
	if rt, ok := a.routeTablesByIfaceName[ifaceName]; !ok {
		logrus.WithField("ifaceName", ifaceName).Info("Making routing table for AWS interface.")
		rt = routetable.New(
			[]string{"^" + regexp.QuoteMeta(ifaceName) + "$", routetable.InterfaceNone},
			4,
			false,
			a.dpConfig.NetlinkTimeout,
			nil,
			a.dpConfig.DeviceRouteProtocol,
			true,
			tableIndex,
			a.opRecorder,
		)
		a.routeTablesByIfaceName[ifaceName] = rt
		a.routeTablesByTable[tableIndex] = rt
	}
	return a.routeTablesByIfaceName[ifaceName]
}

func (a *awsIPManager) getOrAllocRoutingTableID(ifaceName string) int {
	if _, ok := a.routeTableIndexByIfaceName[ifaceName]; !ok {
		lastIdx := len(a.freeRouteTableIndexes) - 1
		a.routeTableIndexByIfaceName[ifaceName] = a.freeRouteTableIndexes[lastIdx]
		a.freeRouteTableIndexes = a.freeRouteTableIndexes[:lastIdx]
	}
	return a.routeTableIndexByIfaceName[ifaceName]
}

func (a *awsIPManager) releaseRoutingTableID(ifaceName string) {
	id := a.routeTableIndexByIfaceName[ifaceName]
	delete(a.routeTableIndexByIfaceName, ifaceName)
	a.freeRouteTableIndexes = append(a.freeRouteTableIndexes, id)
}

func (a *awsIPManager) GetRouteTableSyncers() []routeTableSyncer {
	var rts []routeTableSyncer
	for _, t := range a.routeTablesByTable {
		rts = append(rts, t)
	}
	return rts
}

func (a *awsIPManager) GetRouteRules() []routeRules {
	return []routeRules{a.routeRules}
}

var _ Manager = (*awsIPManager)(nil)
var _ ManagerWithRouteRules = (*awsIPManager)(nil)
var _ ManagerWithRouteTables = (*awsIPManager)(nil)
