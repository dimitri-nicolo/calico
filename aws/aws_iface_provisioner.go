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
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/proto"
	calierrors "github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	calinet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/set"
)

const (
	// MaxInterfacesPerInstance is the current maximum total number of ENIs supported by any AWS instance type.
	// We only support the first network card on an instance right now so limiting to the maximum that one network
	// card can support.
	MaxInterfacesPerInstance = 15

	// SecondaryInterfaceCap is the maximum number of Calico secondary ENIs that we support.  The only reason to
	// cap this right now is so that we can pre-allocate one routing table per possible secondary ENI.
	SecondaryInterfaceCap = MaxInterfacesPerInstance - 1
)

// ipamInterface is just the parts of the IPAM interface that we need.
type ipamInterface interface {
	AutoAssign(ctx context.Context, args ipam.AutoAssignArgs) (*ipam.IPAMAssignments, *ipam.IPAMAssignments, error)
	ReleaseIPs(ctx context.Context, ips []calinet.IP) ([]calinet.IP, error)
	IPsByHandle(ctx context.Context, handleID string) ([]calinet.IP, error)
}

// Compile-time assert: ipamInterface should match the real interface.
var _ ipamInterface = ipam.Interface(nil)

type SecondaryIfaceProvisioner struct {
	nodeName     string
	timeout      time.Duration
	clock        clock.Clock
	newEC2Client func(ctx context.Context) (*EC2Client, error)

	healthAgg  *health.HealthAggregator
	opRecorder logutils.OpRecorder
	ipamClient ipamInterface

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
	responseC        chan *IfaceState
	capacityCallback func(SecondaryIfaceCapacities)
}

type DatastoreState struct {
	LocalAWSRoutesByDst       map[ip.CIDR]*proto.RouteUpdate
	LocalRouteDestsBySubnetID map[string]set.Set /*ip.CIDR*/
	PoolIDsBySubnetID         map[string]set.Set /*string*/
}

const (
	healthNameENICapacity = "aws-eni-capacity"
	healthNameAWSInSync   = "aws-eni-addresses-in-sync"
	defaultTimeout        = 30 * time.Second
)

type IfaceProvOpt func(provisioner *SecondaryIfaceProvisioner)

func OptTimeout(to time.Duration) IfaceProvOpt {
	return func(provisioner *SecondaryIfaceProvisioner) {
		provisioner.timeout = to
	}
}

func OptCapacityCallback(cb func(SecondaryIfaceCapacities)) IfaceProvOpt {

	return func(provisioner *SecondaryIfaceProvisioner) {
		provisioner.capacityCallback = cb
	}
}

func OptClockOverride(c clock.Clock) IfaceProvOpt {
	return func(provisioner *SecondaryIfaceProvisioner) {
		provisioner.clock = c
	}
}

func OptNewEC2ClientkOverride(f func(ctx context.Context) (*EC2Client, error)) IfaceProvOpt {
	return func(provisioner *SecondaryIfaceProvisioner) {
		provisioner.newEC2Client = f
	}
}

type SecondaryIfaceCapacities struct {
	MaxCalicoSecondaryIPs int
}

func (c SecondaryIfaceCapacities) Equals(caps SecondaryIfaceCapacities) bool {
	return c == caps
}

func NewSecondaryIfaceProvisioner(
	nodeName string,
	healthAgg *health.HealthAggregator,
	ipamClient ipamInterface,
	options ...IfaceProvOpt,
) *SecondaryIfaceProvisioner {
	healthAgg.RegisterReporter(healthNameENICapacity, &health.HealthReport{
		Ready: true,
		Live:  false,
	}, 0)
	healthAgg.Report(healthNameENICapacity, &health.HealthReport{
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

	sip := &SecondaryIfaceProvisioner{
		healthAgg:  healthAgg,
		ipamClient: ipamClient,
		nodeName:   nodeName,
		timeout:    defaultTimeout,
		opRecorder: logutils.NewSummarizer("AWS secondary IP reconciliation loop"),

		// Do the extra scans on first run.
		orphanNICResyncNeeded: true,
		hostIPAMResyncNeeded:  true,

		datastoreUpdateC: make(chan DatastoreState, 1),
		responseC:        make(chan *IfaceState, 1),
		clock:            clock.RealClock{},
		capacityCallback: func(c SecondaryIfaceCapacities) {
			logrus.WithField("cap", c).Debug("Capacity updated but no callback configured.")
		},
		newEC2Client: NewEC2Client,
	}

	for _, o := range options {
		o(sip)
	}

	return sip
}

func (m *SecondaryIfaceProvisioner) Start(ctx context.Context) (done chan struct{}) {
	logrus.Info("Starting AWS secondary interface provisioner.")
	done = make(chan struct{})
	go m.loopKeepingAWSInSync(ctx, done)
	return
}

type IfaceState struct {
	PrimaryNICMAC      string
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
	logrus.Info("AWS secondary interface provisioner running in background.")

	// Response channel is masked (nil) until we're ready to send something.
	var responseC chan *IfaceState
	var response *IfaceState

	// Set ourselves up for exponential backoff after a failure.  backoffMgr.Backoff() returns the same Timer
	// on each call so we need to stop it properly when cancelling it.
	var backoffTimer clock.Timer
	var backoffC <-chan time.Time
	backoffMgr := m.newBackoffManager()
	stopBackoffTimer := func() {
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
	}
	defer stopBackoffTimer()

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
		case responseC <- response:
			// Mask the response channel so we don't resend again and again.
			logrus.WithField("repsonse", response).Debug("Sent AWS state back to main goroutine")
			responseC = nil
			continue // Don't want sending a response to trigger an early resync.
		case <-backoffC:
			// Important: nil out the timer so that stopBackoffTimer() won't try to stop it again (and deadlock).
			backoffC = nil
			backoffTimer = nil
			logrus.Warn("Retrying AWS resync after backoff.")
			m.opRecorder.RecordOperation("aws-retry")
		}

		stopBackoffTimer()

		if m.resyncNeeded {
			var err error
			response, err = m.resync(ctx)
			m.healthAgg.Report(healthNameAWSInSync, &health.HealthReport{Ready: err == nil, Live: true})
			if err != nil {
				logrus.WithError(err).Error("Failed to resync with AWS. Will retry after backoff.")
				backoffTimer = backoffMgr.Backoff()
				backoffC = backoffTimer.C()
			}
			if response == nil {
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

func (m *SecondaryIfaceProvisioner) ResponseC() <-chan *IfaceState {
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

func (m *SecondaryIfaceProvisioner) resync(ctx context.Context) (*IfaceState, error) {
	var awsResyncErr error
	m.opRecorder.RecordOperation("aws-fabric-resync")
	var response *IfaceState
	for attempt := 0; attempt < 5; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		response, awsResyncErr = m.attemptResync()
		if errors.Is(awsResyncErr, errResyncNeeded) {
			// Expected retry needed for some more complex cases...
			logrus.Debug("Restarting resync after modifying AWS state.")
			continue
		} else if awsResyncErr != nil {
			logrus.WithError(awsResyncErr).Warn("Failed to resync AWS subnet state.")
			m.cachedEC2Client = nil // Maybe something wrong with client?
			break
		}
		m.resyncNeeded = false
		break
	}
	if awsResyncErr != nil {
		return nil, awsResyncErr // Will trigger backoff.
	}
	return response, nil
}

func (m *SecondaryIfaceProvisioner) attemptResync() (*IfaceState, error) {
	if m.networkCapabilities == nil {
		// Figure out what kind of instance we are and how many NICs and IPs we can support.
		netCaps, err := m.getMyNetworkCapabilities()
		if err != nil {
			logrus.WithError(err).Error("Failed to get this node's network capabilities from the AWS API; " +
				"are AWS API permissions properly configured?")
			return nil, err
		}
		logrus.WithField("netCaps", netCaps).Info("Retrieved my instance's network capabilities")
		// Cache off the network capabilities since this shouldn't change during the lifetime of an instance.
		m.networkCapabilities = netCaps
	}

	// Collect the current state of this instance and our NICs according to AWS.
	awsSnapshot, resyncState, err := m.loadAWSNICsState()
	if err != nil {
		return nil, err
	}

	// Let the kubernetes Node updater know our capacity.
	m.capacityCallback(SecondaryIfaceCapacities{
		MaxCalicoSecondaryIPs: m.calculateMaxCalicoSecondaryIPs(awsSnapshot),
	})

	// Scan for IPs that are present on our AWS NICs but no longer required by Calico.
	awsIPsToRelease := m.findUnusedAWSIPs(awsSnapshot)

	// Figure out the AWS subnets that live in our AZ.  We can only create NICs within these subnets.
	localSubnetsByID, err := m.loadLocalAWSSubnets()
	if err != nil {
		return nil, err
	}

	// Scan for NICs that are in a subnet that no longer matches an IP pool.
	nicsToRelease := m.findNICsWithNoPool(awsSnapshot)

	// Figure out which Calico IPs are not present in on our AWS NICs.
	allCalicoRoutesNotInAWS := m.findRoutesWithNoAWSAddr(awsSnapshot, localSubnetsByID)

	// Release any AWS IPs that are no longer required.
	err = m.unassignAWSIPs(awsIPsToRelease, awsSnapshot)
	if err != nil {
		return nil, err
	}

	// Release any AWS NICs that are no longer needed.
	err = m.releaseAWSNICs(nicsToRelease, awsSnapshot)
	if err != nil {
		// errResyncNeeded if there were any NICs released.  We return now since the awsSnapshot will be too
		// out of sync to continue.
		return nil, err
	}

	// We only support a single local subnet, choose one based on some heuristics.
	bestSubnetID := m.calculateBestSubnet(awsSnapshot, localSubnetsByID)
	if bestSubnetID == "" {
		logrus.Debug("No AWS subnets needed.")
		return &IfaceState{}, nil
	}

	// Record the gateway address of the best subnet.
	bestSubnet := localSubnetsByID[bestSubnetID]
	subnetCIDR, gatewayAddr, err := m.subnetCIDRAndGW(bestSubnet)
	if err != nil {
		return nil, err
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
		return m.calculateResponse(awsSnapshot)
	}
	logrus.WithField("numNewRoutes", len(subnetCalicoRoutesNotInAWS)).Info("Need to program new AWS IPs")

	if m.orphanNICResyncNeeded {
		// Look for any AWS interfaces that belong to this node (as recorded in a tag that we attach to the node)
		// but are not actually attached to this node.
		err = m.attachOrphanNICs(resyncState, bestSubnetID)
		if err != nil {
			return nil, err
		}
		// We won't need to do this again unless we fail to attach a NIC in the future.
		m.orphanNICResyncNeeded = false
	}

	if m.hostIPAMResyncNeeded {
		// Now we've cleaned up any unneeded NICs. Free any IPs that are assigned to us in IPAM but not in use for
		// one of our NICs.
		err = m.freeUnusedHostCalicoIPs(awsSnapshot)
		if err != nil {
			return nil, err
		}
		// Won't need to do this again unless we hit an IPAM error.
		m.hostIPAMResyncNeeded = false
	}

	// Figure out if we need to add any new NICs to the host.
	numNICsNeeded, err := m.calculateNumNewNICsNeeded(awsSnapshot, bestSubnetID)
	if err != nil {
		return nil, err
	}

	numNICsToCreate := numNICsNeeded
	if numNICsNeeded > 0 {
		// Check if we _can_ create that many NICs.
		numNICsPossible := resyncState.calculateUnusedNICCapacity(m.networkCapabilities)
		haveNICCapacity := numNICsToCreate <= numNICsPossible
		m.healthAgg.Report(healthNameENICapacity, &health.HealthReport{
			Live:  true,
			Ready: haveNICCapacity,
		})
		if !haveNICCapacity {
			logrus.Warnf("Need %d more AWS secondary ENIs to support local workloads but only %d are "+
				"available.  Some local workloads will not have requested AWS connectivity.",
				numNICsToCreate, numNICsPossible)
			numNICsToCreate = numNICsPossible // Avoid trying to create NICs that we know will fail.
		}
	}

	if numNICsToCreate > 0 {
		logrus.WithField("num", numNICsToCreate).Info("Allocating IPs for new AWS NICs.")
		v4addrs, err := m.allocateCalicoHostIPs(numNICsToCreate)
		if err != nil {
			// Queue up a clean up of any IPs we may have leaked.
			m.hostIPAMResyncNeeded = true
			return nil, err
		}
		logrus.WithField("addrs", v4addrs.IPs).Info("Allocated IPs; creating AWS NICs...")
		err = m.createAWSNICs(awsSnapshot, resyncState, bestSubnetID, v4addrs.IPs)
		if err != nil {
			logrus.WithError(err).Error("Some AWS NIC operations failed; may retry.")
			// Queue up a cleanup of any IPs we may have leaked.
			m.hostIPAMResyncNeeded = true
			return nil, err
		}
	}

	// Tell AWS to assign the needed Calico IPs to the secondary NICs as best we can.  (It's possible we weren't able
	// to allocate enough IPs or NICs above.)
	err = m.assignSecondaryIPsToNICs(resyncState, subnetCalicoRoutesNotInAWS)
	if errors.Is(err, errResyncNeeded) {
		// This is the mainline after we assign an IP, avoid doing a full resync and just reload the snapshot.
		awsSnapshot, _, err = m.loadAWSNICsState()
	}
	if err != nil {
		return nil, err
	}
	return m.calculateResponse(awsSnapshot)
}

func (m *SecondaryIfaceProvisioner) calculateResponse(awsNICState *nicSnapshot) (*IfaceState, error) {
	// Index the AWS NICs on MAC.
	ifacesByMAC := map[string]Iface{}
	for _, awsNIC := range awsNICState.calicoOwnedNICsByID {
		iface, err := m.ec2NICToIface(&awsNIC)
		if err != nil {
			logrus.WithError(err).Warn("Failed to convert AWS NIC.")
			continue
		}
		ifacesByMAC[iface.MAC.String()] = *iface
	}
	primaryNIC, err := m.ec2NICToIface(awsNICState.primaryNIC)
	if err != nil {
		logrus.WithError(err).Error("Failed to convert primary NIC.")
		return nil, err
	}
	return &IfaceState{
		PrimaryNICMAC:      primaryNIC.MAC.String(),
		SecondaryNICsByMAC: ifacesByMAC,
		SubnetCIDR:         m.awsSubnetCIDR,
		GatewayAddr:        m.awsGatewayAddr,
	}, nil
}

var errNoMAC = errors.New("AWS NIC missing MAC")

func (m *SecondaryIfaceProvisioner) ec2NICToIface(awsNIC *ec2types.NetworkInterface) (*Iface, error) {
	if awsNIC.MacAddress == nil {
		return nil, errNoMAC
	}
	hwAddr, err := net.ParseMAC(*awsNIC.MacAddress)
	if err != nil {
		logrus.WithError(err).Error("Failed to parse MAC address of AWS NIC.")
		return nil, err
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
	iface := &Iface{
		ID:                 safeReadString(awsNIC.NetworkInterfaceId),
		MAC:                hwAddr,
		PrimaryIPv4Addr:    primary,
		SecondaryIPv4Addrs: secondaryAddrs,
	}
	return iface, nil
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

	if netCaps.MaxNetworkInterfaces > MaxInterfacesPerInstance {
		logrus.Infof("Instance type supports %v interfaces, limiting to our interface cap (%v)",
			netCaps.MaxNetworkInterfaces, MaxInterfacesPerInstance)
		netCaps.MaxNetworkInterfaces = MaxInterfacesPerInstance
	}
	return &netCaps, nil
}

// nicSnapshot captures the current state of the AWS NICs attached to this host, indexed in various ways.
type nicSnapshot struct {
	primaryNIC             *ec2types.NetworkInterface
	calicoOwnedNICsByID    map[string]ec2types.NetworkInterface
	nonCalicoOwnedNICsByID map[string]ec2types.NetworkInterface
	nicIDsBySubnet         map[string][]string
	nicIDByIP              map[ip.CIDR]string
	nicIDByPrimaryIP       map[ip.CIDR]string
	attachmentIDByNICID    map[string]string
}

func (s *nicSnapshot) PrimaryNICSecurityGroups() []string {
	var securityGroups []string
	for _, sg := range s.primaryNIC.Groups {
		if sg.GroupId == nil {
			continue
		}
		securityGroups = append(securityGroups, *sg.GroupId)
	}
	return securityGroups
}

// nicResyncState is the working state of the in-progress resync.  We update this as the resync progresses.
type nicResyncState struct {
	inUseDeviceIndexes      map[int32]bool
	freeIPv4CapacityByNICID map[string]int
}

func (r *nicResyncState) calculateUnusedNICCapacity(netCaps *NetworkCapabilities) int {
	// For now, only supporting the first network card.
	numPossibleNICs := netCaps.MaxNICsForCard(0)
	numExistingNICS := len(r.inUseDeviceIndexes)
	return numPossibleNICs - numExistingNICS
}

func (r *nicResyncState) FindFreeDeviceIdx() int32 {
	devIdx := int32(0)
	for r.inUseDeviceIndexes[devIdx] {
		devIdx++
	}
	return devIdx
}

func (r *nicResyncState) ClaimDeviceIdx(devIdx int32) {
	r.inUseDeviceIndexes[devIdx] = true
}

// loadAWSNICsState looks up all the NICs attached ot this host and creates an nicSnapshot to index them.
func (m *SecondaryIfaceProvisioner) loadAWSNICsState() (s *nicSnapshot, r *nicResyncState, err error) {
	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
	if err != nil {
		return nil, nil, err
	}

	myNICs, err := ec2Client.GetMyEC2NetworkInterfaces(ctx)
	if err != nil {
		return
	}

	s = &nicSnapshot{
		calicoOwnedNICsByID:    map[string]ec2types.NetworkInterface{},
		nonCalicoOwnedNICsByID: map[string]ec2types.NetworkInterface{},
		nicIDsBySubnet:         map[string][]string{},
		nicIDByIP:              map[ip.CIDR]string{},
		nicIDByPrimaryIP:       map[ip.CIDR]string{},
		attachmentIDByNICID:    map[string]string{},
	}

	r = &nicResyncState{
		inUseDeviceIndexes:      map[int32]bool{},
		freeIPv4CapacityByNICID: map[string]int{},
	}

	for _, nic := range myNICs {
		nic := nic
		if nic.NetworkInterfaceId == nil {
			logrus.Debug("AWS NIC had no NetworkInterfaceId.")
			continue
		}
		if nic.Attachment != nil {
			if nic.Attachment.DeviceIndex != nil {
				r.inUseDeviceIndexes[*nic.Attachment.DeviceIndex] = true
			}
			if nic.Attachment.NetworkCardIndex != nil && *nic.Attachment.NetworkCardIndex != 0 {
				// Ignore NICs that aren't on the primary network card.  We only support one network card for now.
				logrus.Debugf("Ignoring NIC on non-primary network card: %d.", *nic.Attachment.NetworkCardIndex)
				continue
			}
			if nic.Attachment.AttachmentId != nil {
				s.attachmentIDByNICID[*nic.NetworkInterfaceId] = *nic.Attachment.AttachmentId
			}
		}
		if !NetworkInterfaceIsCalicoSecondary(nic) {
			if s.primaryNIC == nil || nic.Attachment != nil && nic.Attachment.DeviceIndex != nil && *nic.Attachment.DeviceIndex == 0 {
				s.primaryNIC = &nic
			}
			s.nonCalicoOwnedNICsByID[*nic.NetworkInterfaceId] = nic
			continue
		}
		// Found one of our managed interfaces; collect its IPs.
		logCtx := logrus.WithField("id", *nic.NetworkInterfaceId)
		logCtx.Debug("Found Calico NIC")
		s.calicoOwnedNICsByID[*nic.NetworkInterfaceId] = nic
		s.nicIDsBySubnet[*nic.SubnetId] = append(s.nicIDsBySubnet[*nic.SubnetId], *nic.NetworkInterfaceId)
		for _, addr := range nic.PrivateIpAddresses {
			if addr.PrivateIpAddress == nil {
				continue
			}
			cidr := ip.MustParseCIDROrIP(*addr.PrivateIpAddress)
			if addr.Primary != nil && *addr.Primary {
				logCtx.WithField("ip", *addr.PrivateIpAddress).Debug("Found primary IP on Calico NIC")
				s.nicIDByPrimaryIP[cidr] = *nic.NetworkInterfaceId
			} else {
				logCtx.WithField("ip", *addr.PrivateIpAddress).Debug("Found secondary IP on Calico NIC")
				s.nicIDByIP[cidr] = *nic.NetworkInterfaceId
			}
		}

		r.freeIPv4CapacityByNICID[*nic.NetworkInterfaceId] = m.networkCapabilities.MaxIPv4PerInterface - len(nic.PrivateIpAddresses)
		logCtx.WithField("availableIPs", r.freeIPv4CapacityByNICID[*nic.NetworkInterfaceId]).Debug("Calculated available IPs")
		if r.freeIPv4CapacityByNICID[*nic.NetworkInterfaceId] < 0 {
			logCtx.Errorf("NIC appears to have more IPs (%v) that it should (%v)", len(nic.PrivateIpAddresses), m.networkCapabilities.MaxIPv4PerInterface)
			r.freeIPv4CapacityByNICID[*nic.NetworkInterfaceId] = 0
		}
	}

	return
}

// findUnusedAWSIPs scans the AWS state for secondary IPs that are not assigned in Calico IPAM.
func (m *SecondaryIfaceProvisioner) findUnusedAWSIPs(awsState *nicSnapshot) set.Set /* ip.Addr */ {
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

// findNICsWithNoPool scans the nicSnapshot for secondary AWS NICs that were created by Calico but no longer
// have an associated IP pool.
func (m *SecondaryIfaceProvisioner) findNICsWithNoPool(awsNICState *nicSnapshot) set.Set {
	nicsToRelease := set.New()
	for nicID, nic := range awsNICState.calicoOwnedNICsByID {
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
func (m *SecondaryIfaceProvisioner) findRoutesWithNoAWSAddr(awsNICState *nicSnapshot, localSubnetsByID map[string]ec2types.Subnet) []*proto.RouteUpdate {
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
// in the nicSnapshot (but it does not refresh the AWS NIC data itself).
func (m *SecondaryIfaceProvisioner) unassignAWSIPs(awsIPsToRelease set.Set, awsNICState *nicSnapshot) error {
	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
	if err != nil {
		return err
	}

	// Batch up the IPs by NIC; the AWS API lets us release multiple IPs from the same NIC in one shot.
	ipsToReleaseByNICID := map[string][]string{}
	awsIPsToRelease.Iter(func(item interface{}) error {
		addr := item.(ip.CIDR)
		nicID := awsNICState.nicIDByIP[addr]
		ipsToReleaseByNICID[nicID] = append(ipsToReleaseByNICID[nicID], addr.Addr().String())
		return nil
	})

	needRefresh := false
	for nicID, ipsToRelease := range ipsToReleaseByNICID {
		nicID := nicID
		_, err := ec2Client.EC2Svc.UnassignPrivateIpAddresses(ctx, &ec2.UnassignPrivateIpAddressesInput{
			NetworkInterfaceId: &nicID,
			PrivateIpAddresses: ipsToRelease,
		})
		if err != nil {
			logrus.WithError(err).WithField("nicID", nicID).Error("Failed to release AWS IPs.")
			return nil
		}
		needRefresh = true
	}

	if needRefresh {
		return errResyncNeeded
	}
	return nil
}

// releaseAWSNICs tries to unattach and release the given NICs.  Returns errResyncNeeded if the nicSnapshot now needs
// to be refreshed.
func (m *SecondaryIfaceProvisioner) releaseAWSNICs(nicsToRelease set.Set, awsNICState *nicSnapshot) error {
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
		_, err := ec2Client.EC2Svc.DetachNetworkInterface(ctx, &ec2.DetachNetworkInterfaceInput{
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
		_, err = ec2Client.EC2Svc.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
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
func (m *SecondaryIfaceProvisioner) calculateBestSubnet(awsNICState *nicSnapshot, localSubnetsByID map[string]ec2types.Subnet) string {
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
func (m *SecondaryIfaceProvisioner) attachOrphanNICs(resyncState *nicResyncState, bestSubnetID string) error {
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
		devIdx := resyncState.FindFreeDeviceIdx()

		subnetID := safeReadString(nic.SubnetId)
		if subnetID != bestSubnetID || int(devIdx) >= m.networkCapabilities.MaxNICsForCard(0) {
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
		resyncState.ClaimDeviceIdx(devIdx)
		attOut, err := ec2Client.EC2Svc.AttachNetworkInterface(ctx, &ec2.AttachNetworkInterfaceInput{
			DeviceIndex:        &devIdx,
			InstanceId:         &ec2Client.InstanceID,
			NetworkInterfaceId: nic.NetworkInterfaceId,
			// For now, only support the first network card.  There's only one type of AWS instance with >1
			// NetworkCard.
			NetworkCardIndex: int32Ptr(0),
		})
		if err != nil {
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
func (m *SecondaryIfaceProvisioner) freeUnusedHostCalicoIPs(awsNICState *nicSnapshot) error {
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
func (m *SecondaryIfaceProvisioner) calculateNumNewNICsNeeded(awsNICState *nicSnapshot, bestSubnetID string) (int, error) {
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
func (m *SecondaryIfaceProvisioner) createAWSNICs(awsNICState *nicSnapshot, resyncState *nicResyncState, subnetID string, v4addrs []calinet.IPNet) error {
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
		cno, err := ec2Client.EC2Svc.CreateNetworkInterface(ctx, &ec2.CreateNetworkInterfaceInput{
			SubnetId:         &subnetID,
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
			logrus.WithError(err).Error("Failed to create interface.")
			finalErr = errors.New("failed to create interface")
			continue // Carry on and try the other interfaces before we give up.
		}

		// Find a free device index.
		devIdx := int32(0)
		for resyncState.inUseDeviceIndexes[devIdx] {
			devIdx++
		}
		resyncState.inUseDeviceIndexes[devIdx] = true
		attOut, err := ec2Client.EC2Svc.AttachNetworkInterface(ctx, &ec2.AttachNetworkInterfaceInput{
			DeviceIndex:        &devIdx,
			InstanceId:         &ec2Client.InstanceID,
			NetworkInterfaceId: cno.NetworkInterface.NetworkInterfaceId,
			// For now, only support the first network card.  There's only one type of AWS instance with >1
			// NetworkCard.
			NetworkCardIndex: int32Ptr(0),
		})
		if err != nil {
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
		resyncState.freeIPv4CapacityByNICID[*cno.NetworkInterface.NetworkInterfaceId] =
			m.networkCapabilities.MaxIPv4PerInterface - len(cno.NetworkInterface.PrivateIpAddresses)
	}

	if finalErr != nil {
		logrus.Info("Some AWS NIC operations failed; queueing a scan for orphaned NICs/IPAM resources.")
		m.hostIPAMResyncNeeded = true
		m.orphanNICResyncNeeded = true
	}

	return finalErr
}

func (m *SecondaryIfaceProvisioner) assignSecondaryIPsToNICs(resyncState *nicResyncState, filteredRoutes []*proto.RouteUpdate) error {
	if len(filteredRoutes) == 0 {
		return nil
	}

	ctx, cancel := m.newContext()
	defer cancel()
	ec2Client, err := m.ec2Client()
	if err != nil {
		return err
	}

	attemptedSomeAssignments := false
	remainingRoutes := filteredRoutes
	var fatalErr error
	for nicID, freeIPs := range resyncState.freeIPv4CapacityByNICID {
		if freeIPs == 0 {
			continue
		}
		routesToAdd := remainingRoutes
		if len(routesToAdd) > freeIPs {
			routesToAdd = routesToAdd[:freeIPs]
		}
		remainingRoutes = remainingRoutes[len(routesToAdd):]

		var ipAddrs []string
		for _, r := range routesToAdd {
			ipAddrs = append(ipAddrs, trimPrefixLen(r.Dst))
		}

		logrus.WithFields(logrus.Fields{"nic": nicID, "addrs": ipAddrs})
		attemptedSomeAssignments = true
		_, err := ec2Client.EC2Svc.AssignPrivateIpAddresses(ctx, &ec2.AssignPrivateIpAddressesInput{
			NetworkInterfaceId: &nicID,
			AllowReassignment:  boolPtr(true),
			PrivateIpAddresses: ipAddrs,
		})
		if err != nil {
			logrus.WithError(err).WithField("nidID", nicID).Error("Failed to assign IPs to my NIC.")
			fatalErr = fmt.Errorf("failed to assign workload IPs to secondary ENI: %w", err)
			continue // Carry on trying to assign more IPs.
		}
		logrus.WithFields(logrus.Fields{"nicID": nicID, "addrs": ipAddrs}).Info("Assigned IPs to secondary NIC.")
	}

	if len(remainingRoutes) > 0 {
		logrus.Warn("Failed to assign all Calico IPs to local ENIs.  Insufficient secondary IP capacity on the available ENIs.")
	}

	if fatalErr != nil {
		return fatalErr
	}

	if attemptedSomeAssignments {
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
	c, err := m.newEC2Client(ctx)
	if err != nil {
		return nil, err
	}
	m.cachedEC2Client = c
	return m.cachedEC2Client, nil
}

func (m *SecondaryIfaceProvisioner) calculateMaxCalicoSecondaryIPs(snapshot *nicSnapshot) int {
	caps := m.networkCapabilities
	maxSecondaryIPsPerNIC := caps.MaxIPv4PerInterface - 1
	maxCalicoNICs := caps.MaxNetworkInterfaces - len(snapshot.nonCalicoOwnedNICsByID)
	maxCapacity := maxCalicoNICs * maxSecondaryIPsPerNIC
	return maxCapacity
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
