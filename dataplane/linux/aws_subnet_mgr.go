// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/vishvananda/netlink"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
	calierrors "github.com/projectcalico/libcalico-go/lib/errors"
	calinet "github.com/projectcalico/libcalico-go/lib/net"

	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/ipam"

	"github.com/projectcalico/felix/aws"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type awsSubnetManager struct {
	poolsByID                 map[string]*proto.IPAMPool
	poolIDsBySubnetID         map[string]set.Set
	localAWSRoutesByDst       map[ip.CIDR]*proto.RouteUpdate
	localRouteDestsBySubnetID map[string]set.Set /*ip.CIDR*/

	networkCapabilities        *aws.NetworkCapabilities
	awsNICsByID                map[string]ec2types.NetworkInterface
	awsGatewayAddr             ip.Addr
	awsSubnetCIDR              ip.CIDR
	routeTableIndexByIfaceName map[string]int
	freeRouteTableIndexes      []int
	routeTables                map[int]routeTable
	routeRules                 *routerule.RouteRules
	lastRules                  []*routerule.Rule

	resyncNeeded          bool
	orphanNICResyncNeeded bool

	nodeName string
	dpConfig Config

	healthAgg            *health.HealthAggregator
	opRecorder           logutils.OpRecorder
	ipamClient           ipam.Interface
	k8sClient            *kubernetes.Clientset
	cachedEC2Client      *aws.EC2Client
	timeout              time.Duration
	hostIPAMResyncNeeded bool
}

const (
	healthNameSubnetCapacity = "have-at-most-one-aws-subnet"
	healthNameAWSInSync      = "aws-enis-in-sync"
)

func NewAWSSubnetManager(
	healthAgg *health.HealthAggregator,
	ipamClient ipam.Interface,
	k8sClient *kubernetes.Clientset,
	nodeName string,
	routeTableIndexes []int,
	dpConfig Config,
	opRecorder logutils.OpRecorder,
) *awsSubnetManager {
	healthAgg.RegisterReporter(healthNameSubnetCapacity, &health.HealthReport{
		Ready: true,
		Live:  false,
	}, 0)
	healthAgg.Report(healthNameSubnetCapacity, &health.HealthReport{
		Ready: true,
		Live:  true,
	})
	healthAgg.RegisterReporter(healthNameAWSInSync, &health.HealthReport{
		Ready: true,
		Live:  false,
	}, 0)
	healthAgg.Report(healthNameAWSInSync, &health.HealthReport{
		Ready: true,
		Live:  true,
	})

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
		healthAgg:                 healthAgg,
		ipamClient:                ipamClient,
		k8sClient:                 k8sClient,
		nodeName:                  nodeName,

		routeTableIndexByIfaceName: map[string]int{},
		freeRouteTableIndexes:      routeTableIndexes,
		routeTables:                map[int]routeTable{},

		orphanNICResyncNeeded: true,
		hostIPAMResyncNeeded:  true,

		routeRules: rules,
		dpConfig:   dpConfig,
		opRecorder: opRecorder,
		timeout:    dpConfig.AWSTimeout,
	}
	sm.queueResync("first run")
	return sm
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
	a.queueResync("IP pool updated")
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
		a.queueResync("route subnet changed")
	}
	if newSubnetID != "" && oldSubnetID != newSubnetID {
		if _, ok := a.localRouteDestsBySubnetID[newSubnetID]; !ok {
			a.localRouteDestsBySubnetID[newSubnetID] = set.New()
		}
		a.localRouteDestsBySubnetID[newSubnetID].Add(dst)
		a.queueResync("route subnet added")
	}

	// Save off the route itself.
	if route == nil {
		if _, ok := a.localAWSRoutesByDst[dst]; !ok {
			return // Not a route we were tracking.
		}
		a.queueResync("route deleted")
		delete(a.localAWSRoutesByDst, dst)
	} else {
		a.localAWSRoutesByDst[dst] = route
		a.queueResync("route updated")
	}
}

func (a *awsSubnetManager) queueResync(reason string) {
	if a.resyncNeeded {
		return
	}
	logrus.WithField("reason", reason).Info("Resync needed")
	a.resyncNeeded = true
}

func (a *awsSubnetManager) CompleteDeferredWork() error {
	if !a.resyncNeeded {
		return nil
	}

	var awsResyncErr error
	for attempt := 0; attempt < 3; attempt++ {
		awsResyncErr = a.resyncWithAWS()
		if errors.Is(awsResyncErr, errResyncNeeded) {
			// Expected retry needed for some more complex cases...
			logrus.Info("Restarting resync after modifying AWS state.")
			continue
		} else if awsResyncErr != nil {
			logrus.WithError(awsResyncErr).Warn("Failed to resync AWS subnet state.")
			continue
		}
		a.resyncNeeded = false
		logrus.Info("Resync completed successfully.")
		break
	}
	if awsResyncErr != nil {
		return awsResyncErr
	}

	return a.resyncWithDataplane()
}

var errResyncNeeded = errors.New("resync needed")

func (a *awsSubnetManager) resyncWithAWS() error {
	// TODO Decouple the AWS resync from the main goroutine.  AWS could be slow or throttle us.  don't want to hold others up.

	if a.networkCapabilities == nil {
		// Figure out what kind of instance we are and how many NICs and IPs we can support.
		netCaps, err := a.getMyNetworkCapabilities()
		if err != nil {
			return err
		}
		logrus.WithField("netCaps", netCaps).Info("Retrieved my instance's network capabilities")
		// Cache off the network capabilities since this shouldn't change during the lifetime of an instance.
		a.networkCapabilities = netCaps
	}

	// Collect the current state of this instance and our NICs according to AWS.
	awsNICState, err := a.loadAWSNICsState()
	a.awsNICsByID = awsNICState.awsNICsByID

	// Scan for IPs that are present on our AWS NICs but no longer required by Calico.
	awsIPsToRelease := a.findUnusedAWSIPs(awsNICState)

	// Figure out the AWS subnets that live in our AZ.  We can only create NICs within these subnets.
	localSubnetsByID, err := a.loadLocalAWSSubnets()
	if err != nil {
		return err
	}

	// Scan for NICs that are in a subnet that no longer matches an IP pool.
	nicsToRelease := a.findNICsWithNoPool(awsNICState)

	// Figure out which Calico IPs are not present in on our AWS NICs.
	allCalicoRoutesNotInAWS := a.findRoutesWithNoAWSAddr(awsNICState, localSubnetsByID)

	// Release any AWS IPs that are no longer required.
	err = a.unassignAWSIPs(awsIPsToRelease, awsNICState)
	var needRefresh bool
	if errors.Is(err, errResyncNeeded) {
		// Released some IPs so awsNICState will be out of sync; defer the return until we've done more clean up.
		needRefresh = true
	} else {
		return err
	}

	// Release any AWS NICs that are no longer needed.
	err = a.releaseAWSNICs(nicsToRelease, awsNICState)
	if err != nil {
		// errResyncNeeded if there were any NICs released.  We return now since the awsNICState will be too
		// out of sync to continue.
		return err
	}

	// We only support a single local subnet, choose one based on some heuristics.
	bestSubnetID := a.calculateBestSubnet(awsNICState, localSubnetsByID)
	if bestSubnetID == "" {
		logrus.Debug("No AWS subnets needed.")
		if needRefresh {
			return errResyncNeeded
		}
		return nil
	}

	// Record the gateway address of the best subnet.
	bestSubnet := localSubnetsByID[bestSubnetID]
	subnetCIDR, gatewayAddr, err := a.subnetCIDRAndGW(bestSubnet)
	if err != nil {
		return err
	}
	if a.awsGatewayAddr != gatewayAddr || a.awsSubnetCIDR != subnetCIDR {
		a.awsGatewayAddr = gatewayAddr
		a.awsSubnetCIDR = subnetCIDR
		logrus.WithFields(logrus.Fields{
			"addr":   a.awsGatewayAddr,
			"subnet": subnetCIDR,
		}).Info("Calculated new AWS subnet CIDR/gateway.")
	}

	// Given the selected subnet, filter down the routes to only those that we can support.
	subnetCalicoRoutesNotInAWS := filterRoutesByAWSSubnet(allCalicoRoutesNotInAWS, bestSubnetID)
	if len(subnetCalicoRoutesNotInAWS) == 0 {
		logrus.Debug("No new AWS IPs to program")
		if needRefresh {
			return errResyncNeeded
		}
		return nil
	}
	logrus.WithField("numNewRoutes", len(subnetCalicoRoutesNotInAWS)).Info("Need to program new AWS IPs")

	if a.orphanNICResyncNeeded {
		// Look for any AWS interfaces that belong to this node (as recorded in a tag that we attach to the node)
		// but are not actually attached to this node.
		err = a.attachOrphanNICs(awsNICState, bestSubnetID)
		if err != nil {
			return err
		}
		// We won't need to do this again unless we fail to attach a NIC in the future.
		a.orphanNICResyncNeeded = false
	}

	if a.hostIPAMResyncNeeded {
		// Now we've cleaned up any unneeded NICs. Free any IPs that are assigned to us in IPAM but not in use for
		// one of our NICs.
		err = a.freeUnusedHostCalicoIPs(awsNICState)
		if err != nil {
			return err
		}
		// Won't need to do this again unless we hit an issue.
		a.hostIPAMResyncNeeded = false
	}

	// TODO clean up any NICs that are missing from IPAM?  Shouldn't be possible but would be good to do.

	// Figure out if we need to add any new NICs to the host.
	numNICsNeeded, err := a.calculateNumNewNICsNeeded(awsNICState, bestSubnetID)
	if err != nil {
		return err
	}

	if numNICsNeeded > 0 {
		logrus.WithField("num", numNICsNeeded).Info("Allocating IPs for new AWS NICs.")
		v4addrs, err := a.allocateCalicoHostIPs(numNICsNeeded)
		if err != nil {
			// Queue up a clean up of any IPs we may have leaked.
			a.hostIPAMResyncNeeded = true
			return err
		}
		logrus.WithField("addrs", v4addrs.IPs).Info("Allocated IPs; creating AWS NICs...")
		err = a.createAWSNICs(awsNICState, bestSubnetID, v4addrs.IPs)
		if err != nil {
			logrus.WithError(err).Error("Some AWS NIC operations failed; may retry.")
			// Queue up a clean up of any IPs we may have leaked.
			a.hostIPAMResyncNeeded = true
			needRefresh = true
		}
	}

	// Tell AWS to assign the needed Calico IPs to the secondary NICs.
	err = a.assignSecondaryIPsToNICs(awsNICState, subnetCalicoRoutesNotInAWS)
	if err != nil {
		return err
	}

	if needRefresh {
		return errResyncNeeded
	}

	// TODO update k8s Node with capacities
	// Report health

	return nil
}

// getMyNetworkCapabilities looks up the network capabilities of this host; this includes the number of NICs
// and IPs per NIC.
func (a *awsSubnetManager) getMyNetworkCapabilities() (*aws.NetworkCapabilities, error) {
	ctx, cancel := a.newContext()
	defer cancel()
	ec2Client, err := a.ec2Client()
	if err != nil {
		return nil, err
	}
	netCaps, err := ec2Client.GetMyNetworkCapabilities(ctx)
	if err != nil {
		return nil, err
	}
	return &netCaps, nil
}

// awsNICState captures the current state of the AWS NICs attached to this host, indexed in various ways.
// It is populated from scratch at the start of each resync.  This is because some operations (such as assigning
// an IP or attaching a new NIC) invalidate the data.
type awsNICState struct {
	awsNICsByID             map[string]ec2types.NetworkInterface
	nicIDsBySubnet          map[string][]string
	nicIDByIP               map[ip.CIDR]string
	nicIDByPrimaryIP        map[ip.CIDR]string
	inUseDeviceIndexes      map[int32]bool
	freeIPv4CapacityByNICID map[string]int
	attachmentIDByNICID     map[string]string
	primaryNIC              *ec2types.NetworkInterface
}

func (s *awsNICState) PrimaryNICSecurityGroups() []string {
	var securityGroups []string
	for _, sg := range s.primaryNIC.Groups {
		if sg.GroupId == nil {
			continue
		}
		securityGroups = append(securityGroups, *sg.GroupId)
	}
	return securityGroups
}

func (s *awsNICState) FindFreeDeviceIdx() int32 {
	devIdx := int32(0)
	for s.inUseDeviceIndexes[devIdx] {
		devIdx++
	}
	return devIdx
}

func (s *awsNICState) ClaimDeviceIdx(devIdx int32) {
	s.inUseDeviceIndexes[devIdx] = true
}

func (s *awsNICState) OnIPUnassigned(nicID string, addr ip.CIDR) {
	delete(s.nicIDByIP, addr)
	s.freeIPv4CapacityByNICID[nicID]++
}

// loadAWSNICsState looks up all the NICs attached ot this host and creates an awsNICState to index them.
func (a *awsSubnetManager) loadAWSNICsState() (s *awsNICState, err error) {
	ctx, cancel := a.newContext()
	defer cancel()
	ec2Client, err := a.ec2Client()
	if err != nil {
		return nil, err
	}

	myNICs, err := ec2Client.GetMyEC2NetworkInterfaces(ctx)
	if err != nil {
		return
	}

	s = &awsNICState{
		awsNICsByID:             map[string]ec2types.NetworkInterface{},
		nicIDsBySubnet:          map[string][]string{},
		nicIDByIP:               map[ip.CIDR]string{},
		nicIDByPrimaryIP:        map[ip.CIDR]string{},
		inUseDeviceIndexes:      map[int32]bool{},
		freeIPv4CapacityByNICID: map[string]int{},
		attachmentIDByNICID:     map[string]string{},
	}

	for _, n := range myNICs {
		if n.NetworkInterfaceId == nil {
			logrus.Debug("AWS NIC had no NetworkInterfaceId.")
			continue
		}
		if n.Attachment != nil {
			if n.Attachment.DeviceIndex != nil {
				s.inUseDeviceIndexes[*n.Attachment.DeviceIndex] = true
			}
			if n.Attachment.AttachmentId != nil {
				s.attachmentIDByNICID[*n.NetworkInterfaceId] = *n.Attachment.AttachmentId
			}
		}
		if !aws.NetworkInterfaceIsCalicoSecondary(n) {
			if s.primaryNIC == nil || n.Attachment != nil && n.Attachment.DeviceIndex != nil && *n.Attachment.DeviceIndex == 0 {
				s.primaryNIC = &n
			}
			continue
		}
		// Found one of our managed interfaces; collect its IPs.
		logCtx := logrus.WithField("id", *n.NetworkInterfaceId)
		logCtx.Debug("Found Calico NIC")
		s.awsNICsByID[*n.NetworkInterfaceId] = n
		s.nicIDsBySubnet[*n.SubnetId] = append(s.nicIDsBySubnet[*n.SubnetId], *n.NetworkInterfaceId)
		for _, addr := range n.PrivateIpAddresses {
			if addr.PrivateIpAddress == nil {
				continue
			}
			cidr := ip.MustParseCIDROrIP(*addr.PrivateIpAddress)
			if addr.Primary != nil && *addr.Primary {
				logCtx.WithField("ip", *addr.PrivateIpAddress).Debug("Found primary IP on Calico NIC")
				s.nicIDByPrimaryIP[cidr] = *n.NetworkInterfaceId
			} else {
				logCtx.WithField("ip", *addr.PrivateIpAddress).Debug("Found secondary IP on Calico NIC")
				s.nicIDByIP[cidr] = *n.NetworkInterfaceId
			}
		}
		s.freeIPv4CapacityByNICID[*n.NetworkInterfaceId] = a.networkCapabilities.MaxIPv4PerInterface - len(n.PrivateIpAddresses)
		logCtx.WithField("availableIPs", s.freeIPv4CapacityByNICID[*n.NetworkInterfaceId]).Debug("Calculated available IPs")
		if s.freeIPv4CapacityByNICID[*n.NetworkInterfaceId] < 0 {
			logCtx.Errorf("NIC appears to have more IPs (%v) that it should (%v)", len(n.PrivateIpAddresses), a.networkCapabilities.MaxIPv4PerInterface)
			s.freeIPv4CapacityByNICID[*n.NetworkInterfaceId] = 0
		}
	}

	return
}

// findUnusedAWSIPs scans the AWS state for secondary IPs that are not assigned in Calico IPAM.
func (a *awsSubnetManager) findUnusedAWSIPs(awsState *awsNICState) set.Set /* ip.Addr */ {
	awsIPsToRelease := set.New()
	for addr, nicID := range awsState.nicIDByIP {
		if _, ok := a.localAWSRoutesByDst[addr]; !ok {
			logrus.WithFields(logrus.Fields{
				"addr":  addr,
				"nidID": nicID,
			}).Info("AWS Secondary IP no longer needed")
			awsIPsToRelease.Add(addr)
		}
	}
	return awsIPsToRelease
}

// loadLocalAWSSubnets looks up all the AWS Subnets that are in this host's VPC and availability zone.
func (a *awsSubnetManager) loadLocalAWSSubnets() (map[string]ec2types.Subnet, error) {
	ctx, cancel := a.newContext()
	defer cancel()
	ec2Client, err := a.ec2Client()
	if err != nil {
		return nil, err
	}

	localSubnets, err := ec2Client.GetAZLocalSubnets(ctx)
	if err != nil {
		return nil, err
	}
	localSubnetsByID := map[string]ec2types.Subnet{}
	for _, s := range localSubnets {
		if s.SubnetId == nil {
			continue
		}
		localSubnetsByID[*s.SubnetId] = s
	}
	return localSubnetsByID, nil
}

// findNICsWithNoPool scans the awsNICState for secondary AWS NICs that were created by Calico but no longer
// have an associated IP pool.
func (a *awsSubnetManager) findNICsWithNoPool(awsNICState *awsNICState) set.Set {
	nicsToRelease := set.New()
	for nicID, nic := range awsNICState.awsNICsByID {
		if _, ok := a.poolIDsBySubnetID[*nic.SubnetId]; ok {
			continue
		}
		// No longer have an IP pool for this NIC.
		logrus.WithFields(logrus.Fields{
			"nicID":  nicID,
			"subnet": *nic.SubnetId,
		}).Info("AWS NIC belongs to subnet with no matching Calico IP pool, NIC should be released")
		nicsToRelease.Add(nicID)
	}
	return nicsToRelease
}

// findRoutesWithNoAWSAddr Scans our local Calico workload routes for routes with no corresponding AWS IP.
func (a *awsSubnetManager) findRoutesWithNoAWSAddr(awsNICState *awsNICState, localSubnetsByID map[string]ec2types.Subnet) []*proto.RouteUpdate {
	var missingRoutes []*proto.RouteUpdate
	for addr, route := range a.localAWSRoutesByDst {
		if _, ok := localSubnetsByID[route.AwsSubnetId]; !ok {
			logrus.WithFields(logrus.Fields{
				"addr":           addr,
				"requiredSubnet": route.AwsSubnetId,
			}).Warn("Local workload needs an IP from an AWS subnet that is not accessible from this " +
				"availability zone. Unable to allocate an AWS IP for it.")
			continue
		}
		if nicID, ok := awsNICState.nicIDByPrimaryIP[addr]; ok {
			logrus.WithFields(logrus.Fields{
				"addr": addr,
				"nic":  nicID,
			}).Warn("Local workload IP clashes with host's primary IP on one of its secondary interfaces.")
			continue
		}
		if nicID, ok := awsNICState.nicIDByIP[addr]; ok {
			logrus.WithFields(logrus.Fields{
				"addr": addr,
				"nic":  nicID,
			}).Debug("Local workload IP is already present on one of our AWS NICs.")
			continue
		}
		logrus.WithFields(logrus.Fields{
			"addr":      addr,
			"awsSubnet": route.AwsSubnetId,
		}).Info("Local workload IP needs to be added to AWS NIC.")
		missingRoutes = append(missingRoutes, route)
	}
	return missingRoutes
}

// unassignAWSIPs unassigns (releases) the given IPs in the AWS fabric.  It updates the free IP counters
// in the awsNICState (but it does not refresh the AWS NIC data itself).
func (a *awsSubnetManager) unassignAWSIPs(awsIPsToRelease set.Set, awsNICState *awsNICState) error {
	ctx, cancel := a.newContext()
	defer cancel()
	ec2Client, err := a.ec2Client()
	if err != nil {
		return err
	}

	needRefresh := false
	awsIPsToRelease.Iter(func(item interface{}) error {
		addr := item.(ip.CIDR)
		nicID := awsNICState.nicIDByIP[addr]
		_, err := ec2Client.EC2Svc.UnassignPrivateIpAddresses(ctx, &ec2.UnassignPrivateIpAddressesInput{
			NetworkInterfaceId: &nicID,
			// TODO batch up all updates for same NIC?
			PrivateIpAddresses: []string{addr.Addr().String()},
		})
		if err != nil {
			logrus.WithError(err).Error("Failed to release AWS IP.")
			return nil
		}
		// TODO Modifying awsNICState but also signalling a refresh
		awsNICState.OnIPUnassigned(nicID, addr)
		needRefresh = true
		return nil
	})

	if needRefresh {
		return errResyncNeeded
	}
	return nil
}

// releaseAWSNICs tries to unattach and release the given NICs.  Returns errResyncNeeded if the awsNICState now needs
// to be refreshed.
func (a *awsSubnetManager) releaseAWSNICs(nicsToRelease set.Set, awsNICState *awsNICState) error {
	if nicsToRelease.Len() == 0 {
		return nil
	}

	// About to release some NICs, queue up a check of our IPAM handle.
	a.hostIPAMResyncNeeded = true

	ctx, cancel := a.newContext()
	defer cancel()
	ec2Client, err := a.ec2Client()
	if err != nil {
		return err
	}

	// Release any NICs we no longer want.
	nicsToRelease.Iter(func(item interface{}) error {
		nicID := item.(string)
		attachID := awsNICState.attachmentIDByNICID[nicID]
		_, err := ec2Client.EC2Svc.(*ec2.Client).DetachNetworkInterface(ctx, &ec2.DetachNetworkInterfaceInput{
			AttachmentId: &attachID,
			Force:        boolPtr(true),
		})
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"nicID":    nicID,
				"attachID": attachID,
			}).Error("Failed to detach unneeded NIC")
		}
		// Worth trying this even if detach fails.  Possible the failure was caused by it already
		// being detached.
		_, err = ec2Client.EC2Svc.(*ec2.Client).DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: &nicID,
		})
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"nicID":    nicID,
				"attachID": attachID,
			}).Error("Failed to delete unneeded NIC")
		}
		return nil
	})
	return errResyncNeeded
}

// calculateBestSubnet Tries to calculate a single "best" AWS subnet for this host.  When we're configured correctly
// there should only be one subnet in use on this host but we try to pick a sensible one if the IP pools have conflicting
// information.
func (a *awsSubnetManager) calculateBestSubnet(awsNICState *awsNICState, localSubnetsByID map[string]ec2types.Subnet) string {
	// Match AWS subnets against our IP pools.
	localIPPoolSubnetIDs := set.New()
	for subnetID := range a.poolIDsBySubnetID {
		if _, ok := localSubnetsByID[subnetID]; ok {
			localIPPoolSubnetIDs.Add(subnetID)
		}
	}
	logrus.WithField("subnets", localIPPoolSubnetIDs).Debug("AWS Subnets with associated Calico IP pool.")

	// If the IP pools only name one then that is preferred.  If there's more than one in the IP pools but we've
	// already got a local NIC, that one is preferred.  If there's a tie, pick the one with the most routes.
	subnetScores := map[string]int{}
	localIPPoolSubnetIDs.Iter(func(item interface{}) error {
		subnetID := item.(string)
		subnetScores[subnetID] += 1000000
		return nil
	})
	for subnet, nicIDs := range awsNICState.nicIDsBySubnet {
		subnetScores[subnet] += 10000 * len(nicIDs)
	}
	for _, r := range a.localAWSRoutesByDst {
		subnetScores[r.AwsSubnetId] += 1
	}
	var bestSubnet string
	var bestScore int
	for subnet, score := range subnetScores {
		if score > bestScore ||
			score == bestScore && subnet > bestSubnet {
			bestSubnet = subnet
			bestScore = score
		}
	}
	return bestSubnet
}

// subnetCIDRAndGW extracts the subnet's CIDR and gateway address from the given AWS subnet.
func (a *awsSubnetManager) subnetCIDRAndGW(subnet ec2types.Subnet) (ip.CIDR, ip.Addr, error) {
	subnetID := safeReadString(subnet.SubnetId)
	if subnet.CidrBlock == nil {
		return nil, nil, fmt.Errorf("our subnet missing its CIDR id=%s", subnetID) // AWS bug?
	}
	ourCIDR, err := ip.ParseCIDROrIP(*subnet.CidrBlock)
	if err != nil {
		return nil, nil, fmt.Errorf("our subnet had malformed CIDR %q: %w", *subnet.CidrBlock, err)
	}
	// The AWS Subnet gateway is always the ".1" address in the subnet.
	addr := ourCIDR.Addr().Add(1)
	return ourCIDR, addr, nil
}

// filterRoutesByAWSSubnet returns the subset of the given routes that belong to the given AWS subnet.
func filterRoutesByAWSSubnet(missingRoutes []*proto.RouteUpdate, bestSubnet string) []*proto.RouteUpdate {
	var filteredRoutes []*proto.RouteUpdate
	for _, r := range missingRoutes {
		if r.AwsSubnetId != bestSubnet {
			logrus.WithFields(logrus.Fields{
				"route":        r,
				"activeSubnet": bestSubnet,
			}).Warn("Cannot program route into AWS fabric; only one AWS subnet is supported per node.")
			continue
		}
		filteredRoutes = append(filteredRoutes, r)
	}
	return filteredRoutes
}

// attachOrphanNICs looks for any unattached Calico-created NICs that should be attached to this host and tries
// to attach them.
func (a *awsSubnetManager) attachOrphanNICs(awsNICState *awsNICState, bestSubnetID string) error {
	ctx, cancel := a.newContext()
	defer cancel()
	ec2Client, err := a.ec2Client()
	if err != nil {
		return err
	}

	dio, err := ec2Client.EC2Svc.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
		Filters: []ec2types.Filter{
			{
				// We label all our NICs at creation time with the instance they belong to.
				Name:   stringPointer("tag:" + aws.NetworkInterfaceTagOwningInstance),
				Values: []string{ec2Client.InstanceID},
			},
			{
				Name:   stringPointer("status"),
				Values: []string{"available" /* Not attached to the instance */},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list unattached NICs that belong to this node: %w", err)
	}

	attachedOrphan := false
	for _, nic := range dio.NetworkInterfaces {
		// Find next free device index.
		devIdx := awsNICState.FindFreeDeviceIdx()

		subnetID := safeReadString(nic.SubnetId)
		if subnetID != bestSubnetID || int(devIdx) >= a.networkCapabilities.MaxNetworkInterfaces {
			nicID := safeReadString(nic.NetworkInterfaceId)
			logrus.WithField("nicID", nicID).Info(
				"Found unattached NIC that belongs to this node and is no longer needed, deleting.")
			_, err = ec2Client.EC2Svc.(*ec2.Client).DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
				NetworkInterfaceId: nic.NetworkInterfaceId,
			})
			if err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"nicID": nicID,
				}).Error("Failed to delete unattached NIC")
				// Could bail out here but having an orphaned NIC doesn't stop us from getting _our_ state right.
			}
			continue
		}

		logrus.WithFields(logrus.Fields{
			"nicID": nic.NetworkInterfaceId,
		}).Info("Found unattached NIC that belongs to this node; trying to attach it.")
		awsNICState.ClaimDeviceIdx(devIdx)
		attOut, err := ec2Client.EC2Svc.AttachNetworkInterface(ctx, &ec2.AttachNetworkInterfaceInput{
			DeviceIndex:        &devIdx,
			InstanceId:         &ec2Client.InstanceID,
			NetworkInterfaceId: nic.NetworkInterfaceId,
			NetworkCardIndex:   nil, // TODO Multi-network card handling
		})
		if err != nil {
			// TODO handle idempotency; make sure that we can't get a successful failure(!)
			logrus.WithError(err).Error("Failed to attach interface to host.")
			continue
		}
		logrus.WithFields(logrus.Fields{
			"attachmentID": safeReadString(attOut.AttachmentId),
			"networkCard":  safeReadInt32(attOut.NetworkCardIndex),
		}).Info("Attached orphaned AWS NIC to this host.")
		attachedOrphan = true
	}
	if attachedOrphan {
		return errResyncNeeded
	}
	return nil
}

// freeUnusedHostCalicoIPs finds any IPs assign to this host for a secondary ENI that are not actually in use
// and then frees those IPs.
func (a *awsSubnetManager) freeUnusedHostCalicoIPs(awsNICState *awsNICState) error {
	ctx, cancel := a.newContext()
	defer cancel()
	ourIPs, err := a.ipamClient.IPsByHandle(ctx, a.ipamHandle())
	if err != nil && !errors.Is(err, calierrors.ErrorResourceDoesNotExist{}) {
		return fmt.Errorf("failed to look up our existing IPs: %w", err)
	}
	for _, addr := range ourIPs {
		cidr := ip.CIDRFromNetIP(addr.IP)
		if _, ok := awsNICState.nicIDByPrimaryIP[cidr]; !ok {
			// IP is not assigned to any of our local NICs and, if we got this far, we've already attached
			// any orphaned NICs or deleted them.  Clean up the IP.
			logrus.WithField("addr", addr).Info(
				"Found IP assigned to this node in IPAM but not in use for an AWS NIC, freeing it.")
			_, err := a.ipamClient.ReleaseIPs(ctx, []calinet.IP{addr})
			if err != nil {
				logrus.WithError(err).WithField("ip", addr).Error(
					"Failed to free host IP that we no longer need.")
			}
		}
	}

	return nil
}

func (a *awsSubnetManager) calculateNumNewNICsNeeded(awsNICState *awsNICState, bestSubnetID string) (int, error) {
	totalIPs := a.localRouteDestsBySubnetID[bestSubnetID].Len()
	if a.networkCapabilities.MaxIPv4PerInterface <= 1 {
		logrus.Error("Instance type doesn't support secondary IPs")
		return 0, fmt.Errorf("instance type doesn't support secondary IPs")
	}
	secondaryIPsPerIface := a.networkCapabilities.MaxIPv4PerInterface - 1
	totalNICsNeeded := (totalIPs + secondaryIPsPerIface - 1) / secondaryIPsPerIface
	nicsAlreadyAllocated := len(awsNICState.nicIDsBySubnet[bestSubnetID])
	numNICsNeeded := totalNICsNeeded - nicsAlreadyAllocated

	return numNICsNeeded, nil
}

func (a *awsSubnetManager) allocateCalicoHostIPs(numNICsNeeded int) (*ipam.IPAMAssignments, error) {
	ipamCtx, ipamCancel := a.newContext()

	handle := a.ipamHandle()
	v4addrs, _, err := a.ipamClient.AutoAssign(ipamCtx, ipam.AutoAssignArgs{
		Num4:     numNICsNeeded,
		HandleID: &handle,
		Attrs: map[string]string{
			ipam.AttributeType: "aws-secondary-iface",
			ipam.AttributeNode: a.nodeName,
		},
		Hostname:    a.nodeName,
		IntendedUse: v3.IPPoolAllowedUseHostSecondary,
	})
	ipamCancel()
	if err != nil {
		return nil, err
	}
	if v4addrs == nil || len(v4addrs.IPs) == 0 {
		return nil, fmt.Errorf("failed to allocate IP for secondary interface: %v", v4addrs.Msgs)
	}
	logrus.WithField("ips", v4addrs.IPs).Info("Allocated primary IPs for secondary interfaces")
	if len(v4addrs.IPs) < numNICsNeeded {
		logrus.WithFields(logrus.Fields{
			"needed":    numNICsNeeded,
			"allocated": len(v4addrs.IPs),
		}).Warn("Wasn't able to allocate enough ENI primary IPs. IP pool may be full.")
	}
	return v4addrs, nil
}

func (a *awsSubnetManager) createAWSNICs(awsNICState *awsNICState, subnetID string, v4addrs []calinet.IPNet) error {
	ctx, cancel := a.newContext()
	defer cancel()
	ec2Client, err := a.ec2Client()
	if err != nil {
		return err
	}

	// Figure out the security groups of our primary NIC, we'll copy these to the new interfaces that we create.
	secGroups := awsNICState.PrimaryNICSecurityGroups()

	// Create the new NICs for the IPs we were able to get.
	var finalErr error
	for _, addr := range v4addrs {
		ipStr := addr.IP.String()
		token := fmt.Sprintf("calico-secondary-%s-%s", ec2Client.InstanceID, ipStr)
		cno, err := ec2Client.EC2Svc.CreateNetworkInterface(ctx, &ec2.CreateNetworkInterfaceInput{
			SubnetId:         &subnetID,
			ClientToken:      &token,
			Description:      stringPointer(fmt.Sprintf("Calico secondary NIC for instance %s", ec2Client.InstanceID)),
			Groups:           secGroups,
			Ipv6AddressCount: int32Ptr(0),
			PrivateIpAddress: stringPointer(ipStr),
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeNetworkInterface,
					Tags: []ec2types.Tag{
						{
							Key:   stringPointer(aws.NetworkInterfaceTagUse),
							Value: stringPointer(aws.NetworkInterfaceUseSecondary),
						},
						{
							Key:   stringPointer(aws.NetworkInterfaceTagOwningInstance),
							Value: stringPointer(ec2Client.InstanceID),
						},
					},
				},
			},
		})
		if err != nil {
			// TODO handle idempotency; make sure that we can't get a successful failure(!)
			logrus.WithError(err).Error("Failed to create interface.")
			finalErr = errors.New("failed to create interface")
			continue // Carry on and try the other interfaces before we give up.
		}

		// Find a free device index.
		devIdx := int32(0)
		for awsNICState.inUseDeviceIndexes[devIdx] {
			devIdx++
		}
		awsNICState.inUseDeviceIndexes[devIdx] = true
		attOut, err := ec2Client.EC2Svc.AttachNetworkInterface(ctx, &ec2.AttachNetworkInterfaceInput{
			DeviceIndex:        &devIdx,
			InstanceId:         &ec2Client.InstanceID,
			NetworkInterfaceId: cno.NetworkInterface.NetworkInterfaceId,
			NetworkCardIndex:   nil, // TODO Multi-network card handling
		})
		if err != nil {
			// TODO handle idempotency; make sure that we can't get a successful failure(!)
			logrus.WithError(err).Error("Failed to attach interface to host.")
			finalErr = errors.New("failed to attach interface")
			continue // Carry on and try the other interfaces before we give up.
		}
		logrus.WithFields(logrus.Fields{
			"attachmentID": safeReadString(attOut.AttachmentId),
			"networkCard":  safeReadInt32(attOut.NetworkCardIndex),
		}).Info("Attached NIC.")

		// Calculate the free IPs from the output. Once we add an idempotency token, it'll be possible to have
		// >1 IP in place already.
		awsNICState.freeIPv4CapacityByNICID[*cno.NetworkInterface.NetworkInterfaceId] =
			a.networkCapabilities.MaxIPv4PerInterface - len(cno.NetworkInterface.PrivateIpAddresses)

		// TODO disable source/dest check?
	}

	if finalErr != nil {
		log.Info("Some AWS NIC operations failed; queueing a scan for orphaned NICs.")
		a.orphanNICResyncNeeded = true
	}

	return finalErr
}

func (a *awsSubnetManager) assignSecondaryIPsToNICs(awsNICState *awsNICState, filteredRoutes []*proto.RouteUpdate) error {
	ctx, cancel := a.newContext()
	defer cancel()
	ec2Client, err := a.ec2Client()
	if err != nil {
		return err
	}

	var needRefresh bool
	for nicID, freeIPs := range awsNICState.freeIPv4CapacityByNICID {
		if freeIPs == 0 {
			continue
		}
		routesToAdd := filteredRoutes
		if len(routesToAdd) > freeIPs {
			routesToAdd = routesToAdd[:freeIPs]
		}
		filteredRoutes = filteredRoutes[len(routesToAdd):]

		var ipAddrs []string
		for _, r := range routesToAdd {
			ipAddrs = append(ipAddrs, trimPrefixLen(r.Dst))
		}

		logrus.WithFields(logrus.Fields{"nic": nicID, "addrs": ipAddrs})
		_, err := ec2Client.EC2Svc.AssignPrivateIpAddresses(ctx, &ec2.AssignPrivateIpAddressesInput{
			NetworkInterfaceId: &nicID,
			AllowReassignment:  boolPtr(true),
			PrivateIpAddresses: ipAddrs,
		})
		if err != nil {
			logrus.WithError(err).WithField("nidID", nicID).Error("Failed to assign IPs to my NIC.")
			needRefresh = true
		}
		logrus.WithFields(logrus.Fields{"nicID": nicID, "addrs": ipAddrs}).Info("Assigned IPs to secondary NIC.")
		needRefresh = true
	}
	if needRefresh {
		return errResyncNeeded
	}
	return nil
}

func (a *awsSubnetManager) ipamHandle() string {
	// Using the node name here for consistency with tunnel IPs.
	return fmt.Sprintf("aws-secondary-ifaces-%s", a.nodeName)
}

func (a *awsSubnetManager) newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), a.timeout)
}

func (a *awsSubnetManager) ec2Client() (*aws.EC2Client, error) {
	if a.cachedEC2Client != nil {
		return a.cachedEC2Client, nil
	}

	ctx, cancel := a.newContext()
	defer cancel()
	c, err := aws.NewEC2Client(ctx)
	if err != nil {
		return nil, err
	}
	a.cachedEC2Client = c
	return a.cachedEC2Client, nil
}

func (a *awsSubnetManager) resyncWithDataplane() error {
	// TODO Listen for interface updates
	if a.awsGatewayAddr == nil {
		logrus.Info("No AWS gateway address yet.")
		return nil
	}

	// Index the AWS NICs on MAC.
	awsIfacesByMAC := map[string]ec2types.NetworkInterface{}
	for _, awsNIC := range a.awsNICsByID {
		if awsNIC.MacAddress == nil {
			continue
		}
		hwAddr, err := net.ParseMAC(*awsNIC.MacAddress)
		if err != nil {
			logrus.WithError(err).Error("Failed to parse MAC address of AWS NIC.")
		}
		awsIfacesByMAC[hwAddr.String()] = awsNIC
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
		awsNIC, ok := awsIfacesByMAC[mac]
		if !ok {
			continue
		}
		if awsNIC.NetworkInterfaceId == nil || awsNIC.PrivateIpAddress == nil {
			logrus.WithField("nic", awsNIC).Warn("AWS NIC missing ID or address?")
			continue // Very unlikely.
		}
		logrus.WithFields(logrus.Fields{
			"mac":      mac,
			"name":     ifaceName,
			"awsNICID": awsNIC.NetworkInterfaceId,
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
				addrStr := *awsNIC.PrivateIpAddress
				newAddr, err := netlink.ParseAddr(addrStr + "/" + fmt.Sprint(a.awsSubnetCIDR.Prefix()))
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

		var secondaryIPs []ec2types.NetworkInterfacePrivateIpAddress
		if len(awsNIC.PrivateIpAddresses) >= 1 {
			// Ignore the primary IP (which is always listed first), it doesn't need a special routing table.
			secondaryIPs = awsNIC.PrivateIpAddresses[1:]
		}
		for _, privateIP := range secondaryIPs {
			if privateIP.PrivateIpAddress == nil {
				continue
			}
			logrus.WithFields(logrus.Fields{
				"addr": *privateIP.PrivateIpAddress,
				"rtID": routingTableID,
			}).Info("Adding routing rule.")
			cidr, err := ip.ParseCIDROrIP(*privateIP.PrivateIpAddress)
			if err != nil {
				logrus.WithField("ip", *privateIP.PrivateIpAddress).Warn("Bad IP from AWS NIC")
			}
			rule := routerule.NewRule(4, 101).MatchSrcAddress(cidr.ToIPNet()).GoToTable(routingTableID)
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
					CIDR: a.awsGatewayAddr.AsCIDR(),
				},
				{
					Type: routetable.TargetTypeGlobalUnicast,
					CIDR: ip.MustParseCIDROrIP("0.0.0.0/0"),
					GW:   a.awsGatewayAddr,
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

func trimPrefixLen(cidr string) string {
	parts := strings.Split(cidr, "/")
	return parts[0]
}

func safeReadInt32(iptr *int32) string {
	if iptr == nil {
		return "<nil>"
	}
	return fmt.Sprint(*iptr)
}
func safeReadString(sptr *string) string {
	if sptr == nil {
		return "<nil>"
	}
	return *sptr
}

func boolPtr(b bool) *bool {
	return &b
}
func int32Ptr(i int32) *int32 {
	return &i
}

func stringPointer(s string) *string {
	return &s
}

var _ Manager = (*awsSubnetManager)(nil)
var _ ManagerWithRouteRules = (*awsSubnetManager)(nil)
var _ ManagerWithRouteTables = (*awsSubnetManager)(nil)
