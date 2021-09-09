// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package aws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/util/clock"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/proto"
	calierrors "github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	calinet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type SecondaryIfaceProvisioner struct {
	nodeName string
	timeout  time.Duration
	clock    clock.Clock

	healthAgg  *health.HealthAggregator
	opRecorder logutils.OpRecorder
	ipamClient ipam.Interface
	k8sClient  *kubernetes.Clientset

	// resyncNeeded is set to true if we need to do any kind of resync.
	resyncNeeded bool
	// orphanNICResyncNeeded is set to true if the next resync should also check for orphaned NICs.  I.e. ones
	// that this node previously created but which never got attached.  (For example, because felix restarted.)
	orphanNICResyncNeeded bool
	// hostIPAMResyncNeeded is set to true if the next resync should also check for IPs that are assigned ot this
	// node but not in use for one of our NICs.
	hostIPAMResyncNeeded bool

	cachedEC2Client     *EC2Client
	networkCapabilities *NetworkCapabilities
	awsGatewayAddr      ip.Addr
	awsSubnetCIDR       ip.CIDR

	// datastoreUpdateC carries updates from Felix's main dataplane loop to our loop.
	datastoreUpdateC chan DatastoreState
	// ds is the most recent datastore state we've received.
	ds DatastoreState

	// ResponseC is our channel back to the main dataplane loop.
	responseC chan *SecondaryIfaceState
	// response, if non-nil, is the response we want to send back to the main dataplane loop.  It carries the
	// current state of the AWS fabric.
	response *SecondaryIfaceState
}

type DatastoreState struct {
	LocalAWSRoutesByDst       map[ip.CIDR]*proto.RouteUpdate
	LocalRouteDestsBySubnetID map[string]set.Set /*ip.CIDR*/
	PoolIDsBySubnetID         map[string]set.Set
}

const (
	healthNameSubnetCapacity = "have-at-most-one-aws-subnet"
	healthNameAWSInSync      = "aws-enis-in-sync"
)

func NewSecondaryIfaceProvisioner(
	healthAgg *health.HealthAggregator,
	ipamClient ipam.Interface,
	k8sClient *kubernetes.Clientset,
	nodeName string,
	awsTimeout time.Duration,
) *SecondaryIfaceProvisioner {
	// TODO actually report health
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

	return &SecondaryIfaceProvisioner{
		healthAgg:  healthAgg,
		ipamClient: ipamClient,
		k8sClient:  k8sClient,
		nodeName:   nodeName,
		timeout:    awsTimeout,
		opRecorder: logutils.NewSummarizer("AWS secondary IP reconciliation loop"),

		datastoreUpdateC: make(chan DatastoreState, 1),
		clock: clock.RealClock{},
	}
}

func (m *SecondaryIfaceProvisioner) Start(ctx context.Context) (done chan struct{}) {
	done = make(chan struct{})
	go m.loopKeepingAWSInSync(ctx, done)
	return
}

type SecondaryIfaceState struct {
	SecondaryNICsByMAC map[string]Iface
	SubnetCIDR         ip.CIDR
	GatewayAddr        ip.Addr
}

type Iface struct {
	ID                 string
	MAC                net.HardwareAddr
	PrimaryIPv4Addr    ip.Addr
	SecondaryIPv4Addrs []ip.Addr
}

func (m *SecondaryIfaceProvisioner) loopKeepingAWSInSync(ctx context.Context, doneC chan struct{}) {
	defer close(doneC)

	// Response channel is masked (nil) until we're ready to send something.
	var responseC chan *SecondaryIfaceState

	backoffMgr := m.newBackoffManager()
	defer backoffMgr.Backoff().Stop()

	var backoffTimer clock.Timer
	var backoffC <-chan time.Time

	for {
		// Thread safety: we receive messages _from_, and, send messages _to_ the dataplane main loop.
		// To avoid deadlock,
		// - Sends on datastoreUpdateC never block the main loop.  We ensure this by draining the capacity one
		//   channel before sending in OnDatastoreUpdate.
		// - We do our receives and sends in the same select block so that we never block a send op on a receive op
		//   or vice versa.
		select {
		case <-ctx.Done():
			logrus.Info("SecondaryIfaceManager stopping, context canceled.")
			return
		case snapshot := <-m.datastoreUpdateC:
			logrus.Debug("New datastore snapshot received")
			m.resyncNeeded = true
			m.ds = snapshot
		case responseC <- m.response:
			// Mask the response channel so we don't resend again and again.
			responseC = nil
			continue // Don't want sending a response to trigger an early resync.
		case <-backoffC:
			// Nil out the timer so we don't try to stop it again below.
			logrus.Warn("Retrying AWS resync after backoff.")
			m.opRecorder.RecordOperation("aws-retry")
			backoffC = nil
			backoffTimer = nil
		}

		if backoffTimer != nil {
			// New snapshot arrived, ignore the backoff since the new snapshot might resolve whatever issue
			// caused us to fail to resync.  We also must reset the timer before calling Backoff() again for
			// correct behaviour. This is the standard time.Timer.Stop() dance...
			if !backoffTimer.Stop() {
				<-backoffTimer.C()
			}
			backoffTimer = nil
			backoffC = nil
		}

		if m.resyncNeeded {
			err := m.resync(ctx)
			if err != nil {
				logrus.WithError(err).Error("Failed to resync with AWS. Will retry after backoff.")
				backoffTimer = backoffMgr.Backoff()
				backoffC = backoffTimer.C()
			}
			if m.response == nil {
				responseC = nil
			} else {
				responseC = m.responseC
			}
		}
	}
}

func (m *SecondaryIfaceProvisioner) newBackoffManager() wait.BackoffManager {
	const (
		initBackoff   = 1 * time.Second
		maxBackoff    = 1 * time.Minute
		resetDuration = 10 * time.Minute
		backoffFactor = 2.0
		jitter        = 0.1
	)
	backoffMgr := wait.NewExponentialBackoffManager(initBackoff, maxBackoff, resetDuration, backoffFactor, jitter, m.clock)
	return backoffMgr
}

func (m *SecondaryIfaceProvisioner) ResponseC() <-chan *SecondaryIfaceState {
	return m.responseC
}

func (m *SecondaryIfaceProvisioner) OnDatastoreUpdate(snapshot DatastoreState) {
	// To make sure we don't block, drain any pending update from the channel.
	select {
	case <-m.datastoreUpdateC:
		// Discarded previous snapshot, channel now has capacity for new one.
	default:
		// No pending update.  We're ready to send a new one.
	}
	// Should have capacity in the channel now to send without blocking.
	m.datastoreUpdateC <- snapshot
}

var errResyncNeeded = errors.New("resync needed")

func (m *SecondaryIfaceProvisioner) resync(ctx context.Context) error {
	var awsResyncErr error
	m.opRecorder.RecordOperation("aws-fabric-resync")
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		awsResyncErr = m.attemptResync()
		if errors.Is(awsResyncErr, errResyncNeeded) {
			// Expected retry needed for some more complex cases...
			logrus.Info("Restarting resync after modifying AWS state.")
			continue
		} else if awsResyncErr != nil {
			logrus.WithError(awsResyncErr).Warn("Failed to resync AWS subnet state.")
			m.cachedEC2Client = nil // Maybe something wrong with client?
			return awsResyncErr
		}
		m.resyncNeeded = false
		break
	}
	if awsResyncErr != nil {
		return awsResyncErr
	}
	return nil
}

func (m *SecondaryIfaceProvisioner) attemptResync() error {
	m.response = nil

	if m.networkCapabilities == nil {
		// Figure out what kind of instance we are and how many NICs and IPs we can support.
		netCaps, err := m.getMyNetworkCapabilities()
		if err != nil {
			logrus.WithError(err).Error("Failed to get this node's network capabilities from the AWS API; " +
				"are AWS API permissions properly configured?")
			return err
		}
		logrus.WithField("netCaps", netCaps).Info("Retrieved my instance's network capabilities")
		// Cache off the network capabilities since this shouldn't change during the lifetime of an instance.
		m.networkCapabilities = netCaps
	}

	// Collect the current state of this instance and our NICs according to AWS.
	awsNICState, err := m.loadAWSNICsState()

	// Scan for IPs that are present on our AWS NICs but no longer required by Calico.
	awsIPsToRelease := m.findUnusedAWSIPs(awsNICState)

	// Figure out the AWS subnets that live in our AZ.  We can only create NICs within these subnets.
	localSubnetsByID, err := m.loadLocalAWSSubnets()
	if err != nil {
		return err
	}

	// Scan for NICs that are in a subnet that no longer matches an IP pool.
	nicsToRelease := m.findNICsWithNoPool(awsNICState)

	// Figure out which Calico IPs are not present in on our AWS NICs.
	allCalicoRoutesNotInAWS := m.findRoutesWithNoAWSAddr(awsNICState, localSubnetsByID)

	// Release any AWS IPs that are no longer required.
	err = m.unassignAWSIPs(awsIPsToRelease, awsNICState)
	var needRefresh bool
	if errors.Is(err, errResyncNeeded) {
		// Released some IPs so awsNICState will be out of sync; defer the return until we've done more clean up.
		needRefresh = true
	} else {
		return err
	}

	// Release any AWS NICs that are no longer needed.
	err = m.releaseAWSNICs(nicsToRelease, awsNICState)
	if err != nil {
		// errResyncNeeded if there were any NICs released.  We return now since the awsNICState will be too
		// out of sync to continue.
		return err
	}

	// We only support a single local subnet, choose one based on some heuristics.
	bestSubnetID := m.calculateBestSubnet(awsNICState, localSubnetsByID)
	if bestSubnetID == "" {
		logrus.Debug("No AWS subnets needed.")
		m.response = &SecondaryIfaceState{}
		if needRefresh {
			return errResyncNeeded
		}
		return nil
	}

	// Record the gateway address of the best subnet.
	bestSubnet := localSubnetsByID[bestSubnetID]
	subnetCIDR, gatewayAddr, err := m.subnetCIDRAndGW(bestSubnet)
	if err != nil {
		return err
	}
	if m.awsGatewayAddr != gatewayAddr || m.awsSubnetCIDR != subnetCIDR {
		m.awsGatewayAddr = gatewayAddr
		m.awsSubnetCIDR = subnetCIDR
		logrus.WithFields(logrus.Fields{
			"addr":   m.awsGatewayAddr,
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

	if m.orphanNICResyncNeeded {
		// Look for any AWS interfaces that belong to this node (as recorded in a tag that we attach to the node)
		// but are not actually attached to this node.
		err = m.attachOrphanNICs(awsNICState, bestSubnetID)
		if err != nil {
			return err
		}
		// We won't need to do this again unless we fail to attach a NIC in the future.
		m.orphanNICResyncNeeded = false
	}

	if m.hostIPAMResyncNeeded {
		// Now we've cleaned up any unneeded NICs. Free any IPs that are assigned to us in IPAM but not in use for
		// one of our NICs.
		err = m.freeUnusedHostCalicoIPs(awsNICState)
		if err != nil {
			return err
		}
		// Won't need to do this again unless we hit an issue.
		m.hostIPAMResyncNeeded = false
	}

	// TODO clean up any NICs that are missing from IPAM?  Shouldn't be possible but would be good to do.

	// Figure out if we need to add any new NICs to the host.
	numNICsNeeded, err := m.calculateNumNewNICsNeeded(awsNICState, bestSubnetID)
	if err != nil {
		return err
	}

	if numNICsNeeded > 0 {
		logrus.WithField("num", numNICsNeeded).Info("Allocating IPs for new AWS NICs.")
		v4addrs, err := m.allocateCalicoHostIPs(numNICsNeeded)
		if err != nil {
			// Queue up a clean up of any IPs we may have leaked.
			m.hostIPAMResyncNeeded = true
			return err
		}
		logrus.WithField("addrs", v4addrs.IPs).Info("Allocated IPs; creating AWS NICs...")
		err = m.createAWSNICs(awsNICState, bestSubnetID, v4addrs.IPs)
		if err != nil {
			logrus.WithError(err).Error("Some AWS NIC operations failed; may retry.")
			// Queue up a clean up of any IPs we may have leaked.
			m.hostIPAMResyncNeeded = true
			needRefresh = true
		}
	}

	// Tell AWS to assign the needed Calico IPs to the secondary NICs as best we can.  (It's possible we weren't able
	// to allocate enough IPs or NICs above.)
	err = m.assignSecondaryIPsToNICs(awsNICState, subnetCalicoRoutesNotInAWS)
	if err != nil {
		return err
	}

	if needRefresh {
		return errResyncNeeded
	}

	// TODO update k8s Node with capacities
	// TODO Report health

	m.response = m.calculateResponse(awsNICState)

	return nil
}

func (m *SecondaryIfaceProvisioner) calculateResponse(awsNICState *awsNICState) *SecondaryIfaceState {
	// Index the AWS NICs on MAC.
	ifacesByMAC := map[string]Iface{}
	for nicID, awsNIC := range awsNICState.awsNICsByID {
		if awsNIC.MacAddress == nil {
			continue
		}
		hwAddr, err := net.ParseMAC(*awsNIC.MacAddress)
		if err != nil {
			logrus.WithError(err).Error("Failed to parse MAC address of AWS NIC.")
		}
		var primary ip.Addr
		var secondaryAddrs []ip.Addr
		for _, pa := range awsNIC.PrivateIpAddresses {
			if pa.PrivateIpAddress == nil {
				continue
			}
			addr := ip.FromString(*pa.PrivateIpAddress)
			if pa.Primary != nil && *pa.Primary || primary == nil /* primary should be first */ {
				primary = addr
			} else {
				secondaryAddrs = append(secondaryAddrs, addr)
			}
		}
		ifacesByMAC[hwAddr.String()] = Iface{
			ID:                 nicID,
			MAC:                hwAddr,
			PrimaryIPv4Addr:    primary,
			SecondaryIPv4Addrs: secondaryAddrs,
		}
	}

	return &SecondaryIfaceState{
		SecondaryNICsByMAC: ifacesByMAC,
		SubnetCIDR:         m.awsSubnetCIDR,
		GatewayAddr:        m.awsGatewayAddr,
	}
}

// getMyNetworkCapabilities looks up the network capabilities of this host; this includes the number of NICs
// and IPs per NIC.
func (m *SecondaryIfaceProvisioner) getMyNetworkCapabilities() (*NetworkCapabilities, error) {
	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
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
func (m *SecondaryIfaceProvisioner) loadAWSNICsState() (s *awsNICState, err error) {
	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
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
		if !NetworkInterfaceIsCalicoSecondary(n) {
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
		s.freeIPv4CapacityByNICID[*n.NetworkInterfaceId] = m.networkCapabilities.MaxIPv4PerInterface - len(n.PrivateIpAddresses)
		logCtx.WithField("availableIPs", s.freeIPv4CapacityByNICID[*n.NetworkInterfaceId]).Debug("Calculated available IPs")
		if s.freeIPv4CapacityByNICID[*n.NetworkInterfaceId] < 0 {
			logCtx.Errorf("NIC appears to have more IPs (%v) that it should (%v)", len(n.PrivateIpAddresses), m.networkCapabilities.MaxIPv4PerInterface)
			s.freeIPv4CapacityByNICID[*n.NetworkInterfaceId] = 0
		}
	}

	return
}

// findUnusedAWSIPs scans the AWS state for secondary IPs that are not assigned in Calico IPAM.
func (m *SecondaryIfaceProvisioner) findUnusedAWSIPs(awsState *awsNICState) set.Set /* ip.Addr */ {
	awsIPsToRelease := set.New()
	for addr, nicID := range awsState.nicIDByIP {
		if _, ok := m.ds.LocalAWSRoutesByDst[addr]; !ok {
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
func (m *SecondaryIfaceProvisioner) loadLocalAWSSubnets() (map[string]ec2types.Subnet, error) {
	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
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
func (m *SecondaryIfaceProvisioner) findNICsWithNoPool(awsNICState *awsNICState) set.Set {
	nicsToRelease := set.New()
	for nicID, nic := range awsNICState.awsNICsByID {
		if _, ok := m.ds.PoolIDsBySubnetID[*nic.SubnetId]; ok {
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
func (m *SecondaryIfaceProvisioner) findRoutesWithNoAWSAddr(awsNICState *awsNICState, localSubnetsByID map[string]ec2types.Subnet) []*proto.RouteUpdate {
	var missingRoutes []*proto.RouteUpdate
	for addr, route := range m.ds.LocalAWSRoutesByDst {
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
func (m *SecondaryIfaceProvisioner) unassignAWSIPs(awsIPsToRelease set.Set, awsNICState *awsNICState) error {
	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
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
func (m *SecondaryIfaceProvisioner) releaseAWSNICs(nicsToRelease set.Set, awsNICState *awsNICState) error {
	if nicsToRelease.Len() == 0 {
		return nil
	}

	// About to release some NICs, queue up a check of our IPAM handle.
	m.hostIPAMResyncNeeded = true

	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
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
func (m *SecondaryIfaceProvisioner) calculateBestSubnet(awsNICState *awsNICState, localSubnetsByID map[string]ec2types.Subnet) string {
	// Match AWS subnets against our IP pools.
	localIPPoolSubnetIDs := set.New()
	for subnetID := range m.ds.PoolIDsBySubnetID {
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
	for _, r := range m.ds.LocalAWSRoutesByDst {
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
func (m *SecondaryIfaceProvisioner) subnetCIDRAndGW(subnet ec2types.Subnet) (ip.CIDR, ip.Addr, error) {
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
func (m *SecondaryIfaceProvisioner) attachOrphanNICs(awsNICState *awsNICState, bestSubnetID string) error {
	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
	if err != nil {
		return err
	}

	dio, err := ec2Client.EC2Svc.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
		Filters: []ec2types.Filter{
			{
				// We label all our NICs at creation time with the instance they belong to.
				Name:   stringPtr("tag:" + NetworkInterfaceTagOwningInstance),
				Values: []string{ec2Client.InstanceID},
			},
			{
				Name:   stringPtr("status"),
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
		if subnetID != bestSubnetID || int(devIdx) >= m.networkCapabilities.MaxNetworkInterfaces {
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
func (m *SecondaryIfaceProvisioner) freeUnusedHostCalicoIPs(awsNICState *awsNICState) error {
	ctx, cancel := m.newContext()
	defer cancel()
	ourIPs, err := m.ipamClient.IPsByHandle(ctx, m.ipamHandle())
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
			_, err := m.ipamClient.ReleaseIPs(ctx, []calinet.IP{addr})
			if err != nil {
				logrus.WithError(err).WithField("ip", addr).Error(
					"Failed to free host IP that we no longer need.")
			}
		}
	}

	return nil
}

// calculateNumNewNICsNeeded does the maths to figure out how many NICs we need to add given the number of
// IPs we need and the spare capacity of existing NICs.
func (m *SecondaryIfaceProvisioner) calculateNumNewNICsNeeded(awsNICState *awsNICState, bestSubnetID string) (int, error) {
	totalIPs := m.ds.LocalRouteDestsBySubnetID[bestSubnetID].Len()
	if m.networkCapabilities.MaxIPv4PerInterface <= 1 {
		logrus.Error("Instance type doesn't support secondary IPs")
		return 0, fmt.Errorf("instance type doesn't support secondary IPs")
	}
	secondaryIPsPerIface := m.networkCapabilities.MaxIPv4PerInterface - 1
	totalNICsNeeded := (totalIPs + secondaryIPsPerIface - 1) / secondaryIPsPerIface
	nicsAlreadyAllocated := len(awsNICState.nicIDsBySubnet[bestSubnetID])
	numNICsNeeded := totalNICsNeeded - nicsAlreadyAllocated

	return numNICsNeeded, nil
}

// allocateCalicoHostIPs allocates the given number of IPPoolAllowedUseHostSecondary IPs to this host in Calico IPAM.
func (m *SecondaryIfaceProvisioner) allocateCalicoHostIPs(numNICsNeeded int) (*ipam.IPAMAssignments, error) {
	ipamCtx, ipamCancel := m.newContext()

	handle := m.ipamHandle()
	v4addrs, _, err := m.ipamClient.AutoAssign(ipamCtx, ipam.AutoAssignArgs{
		Num4:     numNICsNeeded,
		HandleID: &handle,
		Attrs: map[string]string{
			ipam.AttributeType: "aws-secondary-iface",
			ipam.AttributeNode: m.nodeName,
		},
		Hostname:    m.nodeName,
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

// createAWSNICs creates one AWS secondary ENI in the given subnet for each given IP address and attempts to
// attach the newly created NIC to this host.
func (m *SecondaryIfaceProvisioner) createAWSNICs(awsNICState *awsNICState, subnetID string, v4addrs []calinet.IPNet) error {
	if len(v4addrs) == 0 {
		return nil
	}

	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
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
			Description:      stringPtr(fmt.Sprintf("Calico secondary NIC for instance %s", ec2Client.InstanceID)),
			Groups:           secGroups,
			Ipv6AddressCount: int32Ptr(0),
			PrivateIpAddress: stringPtr(ipStr),
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeNetworkInterface,
					Tags: []ec2types.Tag{
						{
							Key:   stringPtr(NetworkInterfaceTagUse),
							Value: stringPtr(NetworkInterfaceUseSecondary),
						},
						{
							Key:   stringPtr(NetworkInterfaceTagOwningInstance),
							Value: stringPtr(ec2Client.InstanceID),
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
			m.networkCapabilities.MaxIPv4PerInterface - len(cno.NetworkInterface.PrivateIpAddresses)

		// TODO disable source/dest check?
	}

	if finalErr != nil {
		logrus.Info("Some AWS NIC operations failed; queueing a scan for orphaned NICs.")
		m.orphanNICResyncNeeded = true
	}

	return finalErr
}

func (m *SecondaryIfaceProvisioner) assignSecondaryIPsToNICs(awsNICState *awsNICState, filteredRoutes []*proto.RouteUpdate) error {
	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
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

func (m *SecondaryIfaceProvisioner) ipamHandle() string {
	// Using the node name here for consistency with tunnel IPs.
	return fmt.Sprintf("aws-secondary-ifaces-%s", m.nodeName)
}

func (m *SecondaryIfaceProvisioner) newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), m.timeout)
}

func (m *SecondaryIfaceProvisioner) ec2Client() (*EC2Client, error) {
	if m.cachedEC2Client != nil {
		return m.cachedEC2Client, nil
	}

	ctx, cancel := m.newContext()
	defer cancel()
	c, err := NewEC2Client(ctx)
	if err != nil {
		return nil, err
	}
	m.cachedEC2Client = c
	return m.cachedEC2Client, nil
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

func stringPtr(s string) *string {
	return &s
}
