// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"fmt"
	"regexp"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"k8s.io/client-go/kubernetes"

	aws2 "github.com/projectcalico/felix/dataplane/aws"

	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/ipam"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type awsSubnetManager struct {
	// Indexes of data we've learned from the datastore.

	poolsByID                 map[string]*proto.IPAMPool
	poolIDsBySubnetID         map[string]set.Set
	localAWSRoutesByDst       map[ip.CIDR]*proto.RouteUpdate
	localRouteDestsBySubnetID map[string]set.Set /*ip.CIDR*/
	awsResyncNeeded           bool

	// secondaryIfaceMgr manages the AWS fabric resources.  We send it datstore snapshots and it sends back
	// AWSState objects telling us what state the AWS fabric is in.
	secondaryIfaceMgr *aws2.SecondaryIfaceManager

	// awsState is the most recent update we've got from the background thread telling us what state it thinks
	// the AWS fabric should be in.
	// TODO Deal with this being "behind"" without logging lots of errors.
	awsState *aws2.AWSState

	// Dataplane state.

	routeTableIndexByIfaceName map[string]int
	freeRouteTableIndexes      []int
	routeTables                map[int]routeTable
	routeRules                 *routerule.RouteRules
	lastRules                  []*routerule.Rule
	dataplaneResyncNeeded      bool
	dpConfig                   Config

	opRecorder logutils.OpRecorder
}

func NewAWSSubnetManager(
	healthAgg *health.HealthAggregator,
	ipamClient ipam.Interface,
	k8sClient *kubernetes.Clientset,
	nodeName string,
	routeTableIndexes []int,
	dpConfig Config,
	opRecorder logutils.OpRecorder,
) *awsSubnetManager {

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

	sm := &awsSubnetManager{
		poolsByID:                 map[string]*proto.IPAMPool{},
		poolIDsBySubnetID:         map[string]set.Set{},
		localAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
		localRouteDestsBySubnetID: map[string]set.Set{},

		routeTableIndexByIfaceName: map[string]int{},
		freeRouteTableIndexes:      routeTableIndexes,
		routeTables:                map[int]routeTable{},

		routeRules: rules,
		dpConfig:   dpConfig,
		opRecorder: opRecorder,

		secondaryIfaceMgr: aws2.NewSecondaryIfaceManager(
			healthAgg,
			ipamClient,
			k8sClient,
			nodeName,
			dpConfig.AWSTimeout,
		),
	}
	sm.queueAWSResync("first run")
	sm.queueDataplaneResync("first run")
	return sm
}

func (a *awsSubnetManager) ResponseC() chan *aws2.AWSState {
	return a.secondaryIfaceMgr.ResponseC
}

func (a *awsSubnetManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.IPAMPoolUpdate:
		a.onPoolUpdate(msg.Id, msg.Pool)
	case *proto.IPAMPoolRemove:
		a.onPoolUpdate(msg.Id, nil)
	case *proto.RouteUpdate:
		a.onRouteUpdate(ip.MustParseCIDROrIP(msg.Dst), msg)
	case *proto.RouteRemove:
		a.onRouteUpdate(ip.MustParseCIDROrIP(msg.Dst), nil)
	}
}

func (a *awsSubnetManager) OnAWSStateUpdate(msg *aws2.AWSState) {
	a.queueDataplaneResync("AWS fabric updated")
	a.awsState = msg
}

func (a *awsSubnetManager) onPoolUpdate(id string, pool *proto.IPAMPool) {
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

func (a *awsSubnetManager) onRouteUpdate(dst ip.CIDR, route *proto.RouteUpdate) {
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

func (a *awsSubnetManager) queueAWSResync(reason string) {
	if a.awsResyncNeeded {
		return
	}
	logrus.WithField("reason", reason).Debug("AWS resync needed")
	a.awsResyncNeeded = true
}

func (a *awsSubnetManager) queueDataplaneResync(reason string) {
	if a.dataplaneResyncNeeded {
		return
	}
	logrus.WithField("reason", reason).Debug("Dataplane resync needed")
	a.dataplaneResyncNeeded = true
}

func (a *awsSubnetManager) CompleteDeferredWork() error {
	if a.awsResyncNeeded {
		ds := aws2.DatastoreState{
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
		a.secondaryIfaceMgr.OnDatastoreUpdate(ds)
	}

	var err error
	if a.dataplaneResyncNeeded {
		err = a.resyncWithDataplane()
	}
	return err
}

func (a *awsSubnetManager) resyncWithDataplane() error {
	// TODO Listen for interface updates and resync after any relevant ones.

	if a.awsState == nil {
		logrus.Debug("No AWS information yet, not syncing dataplane.")
		return nil
	}

	// Find all the local NICs and match them up with AWS NICs.
	ifaces, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("failed to load local interfaces: %w", err)
	}
	var rules []*routerule.Rule
	for _, iface := range ifaces {
		ifaceName := iface.Attrs().Name
		mac := iface.Attrs().HardwareAddr.String()
		awsNIC, ok := a.awsState.SecondaryNICsByMAC[mac]
		if !ok {
			continue
		}
		logrus.WithFields(logrus.Fields{
			"mac":      mac,
			"name":     ifaceName,
			"awsNICID": awsNIC.ID,
		}).Info("Matched local NIC with AWS NIC.")

		// Enable the NIC.
		err := netlink.LinkSetUp(iface)
		if err != nil {
			ifaceName := iface.Attrs().Name
			logrus.WithError(err).WithField("name", ifaceName).Error("Failed to set link up")
		}

		{
			// Make sure the interface has its primary IP.  This is needed for ARP to work.
			addrs, err := netlink.AddrList(iface, netlink.FAMILY_V4)
			if err != nil {
				logrus.WithError(err).WithField("name", ifaceName).Error("Failed to query interface addrs.")
			} else {
				found := false
				addrStr := awsNIC.PrimaryIPv4Addr.String()
				newAddr, err := netlink.ParseAddr(addrStr + "/" + fmt.Sprint(a.awsState.SubnetCIDR.Prefix()))
				if err != nil {
					logrus.WithError(err).WithFields(logrus.Fields{
						"name": ifaceName,
						"addr": addrStr,
					}).Error("Failed to parse address.")
				}
				newAddr.Scope = int(netlink.SCOPE_LINK)

				for _, a := range addrs {
					if a.Equal(*newAddr) {
						found = true
						continue
					}
					err := netlink.AddrDel(iface, &a)
					if err != nil {
						logrus.WithError(err).WithFields(logrus.Fields{
							"name": ifaceName,
							"addr": a,
						}).Error("Failed to clean up old address.")
					}
				}
				if !found {
					err := netlink.AddrAdd(iface, newAddr)
					if err != nil {
						logrus.WithError(err).WithFields(logrus.Fields{
							"name": ifaceName,
							"addr": newAddr,
						}).Error("Failed to add new address.")
					} else {
						logrus.WithError(err).WithFields(logrus.Fields{
							"name": ifaceName,
							"addr": newAddr,
						}).Info("Added address to secondary ENI.")
					}
				}
			}
		}

		// For each IP assigned to the NIC, we'll add a routing rule that sends traffic _from_ that IP to
		// a dedicated routing table for the NIC.
		routingTableID := a.getOrAllocRoutingTableID(ifaceName)

		for _, privateIP := range awsNIC.SecondaryIPv4Addrs {
			logrus.WithFields(logrus.Fields{
				"addr": privateIP,
				"rtID": routingTableID,
			}).Info("Adding routing rule.")
			rule := routerule.NewRule(4, 101).MatchSrcAddress(privateIP.AsCIDR().ToIPNet()).GoToTable(routingTableID)
			rules = append(rules, rule)
		}

		// Program routes into the NIC-specific routing table.  These work as follows:
		// - Add a default route via the AWS subnet's gateway.  This is how traffic to the outside world gets
		//   routed properly.
		// - Add narrower routes for Calico IP pools that throw the packet back to the main routing tables.
		//   this is required to make RPF checks pass when traffic arrives from a Calico tunnel going to an
		//   AWS-networked pod.
		rt := a.getOrAllocRoutingTable(routingTableID, ifaceName)
		{
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
		}
		{
			var noIFRoutes []routetable.Target
			for _, pool := range a.poolsByID {
				if !pool.Masquerade {
					continue // Assuming that non-masquerade pools are reachable over the main network.
				}
				noIFRoutes = append(noIFRoutes, routetable.Target{
					Type: routetable.TargetTypeThrow,
					CIDR: ip.MustParseCIDROrIP(pool.Cidr),
				})
			}
			rt.SetRoutes(routetable.InterfaceNone, noIFRoutes)
		}
	}

	// TODO Release unused routing table IDs, being careful to clear the routing table and rule before disposing of them
	// TODO Avoid reprogramming all rules every time just to clean up.
	for _, r := range a.lastRules {
		a.routeRules.RemoveRule(r)
	}
	for _, r := range rules {
		a.routeRules.SetRule(r)
	}
	a.lastRules = rules

	return nil
}

func (a *awsSubnetManager) getOrAllocRoutingTable(tableIndex int, ifaceName string) routeTable {
	if _, ok := a.routeTables[tableIndex]; !ok {
		logrus.WithField("ifaceName", ifaceName).Info("Making routing table for AWS interface.")
		a.routeTables[tableIndex] = routetable.New(
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
	}
	return a.routeTables[tableIndex]
}

func (a *awsSubnetManager) getOrAllocRoutingTableID(ifaceName string) int {
	if _, ok := a.routeTableIndexByIfaceName[ifaceName]; !ok {
		a.routeTableIndexByIfaceName[ifaceName] = a.freeRouteTableIndexes[0]
		a.freeRouteTableIndexes = a.freeRouteTableIndexes[1:]
	}
	return a.routeTableIndexByIfaceName[ifaceName]
}

func (a *awsSubnetManager) releaseRoutingTableID(ifaceName string) {
	id := a.routeTableIndexByIfaceName[ifaceName]
	delete(a.routeTableIndexByIfaceName, ifaceName)
	a.freeRouteTableIndexes = append(a.freeRouteTableIndexes, id)
}

func (a *awsSubnetManager) GetRouteTableSyncers() []routeTableSyncer {
	var rts []routeTableSyncer
	for _, t := range a.routeTables {
		rts = append(rts, t)
	}
	return rts
}

func (a *awsSubnetManager) GetRouteRules() []routeRules {
	return []routeRules{a.routeRules}
}

var _ Manager = (*awsSubnetManager)(nil)
var _ ManagerWithRouteRules = (*awsSubnetManager)(nil)
var _ ManagerWithRouteTables = (*awsSubnetManager)(nil)
