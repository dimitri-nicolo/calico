// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/projectcalico/felix/logutils"
	"github.com/projectcalico/felix/routerule"
	"github.com/projectcalico/felix/routetable"
	calierrors "github.com/projectcalico/libcalico-go/lib/errors"
	calinet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/vishvananda/netlink"
	"k8s.io/client-go/kubernetes"

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

	awsNICsByID                map[string]ec2types.NetworkInterface
	awsGatewayAddr             ip.Addr
	routeTableIndexByIfaceName map[string]int
	freeRouteTableIndexes      []int
	routeTables map[int]routeTable
	routeRules *routerule.RouteRules
	lastRules   []*routerule.Rule

	resyncNeeded bool

	nodeName   string
	dpConfig    Config

	healthAgg  *health.HealthAggregator
	opRecorder  logutils.OpRecorder
	ipamClient ipam.Interface
	k8sClient  *kubernetes.Clientset
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
		awsNICsByID: nil , // Set on first resync
		routeTables: map[int]routeTable{},

		routeRules: rules,
		dpConfig:   dpConfig,
		opRecorder: opRecorder,
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
	// TODO Arbitrary timeout for whole operation.
	// TODO Split this up into methods.
	// TODO Decouple the AWS resync from the main goroutine.  AWS could be slow or throttle us.  don't want to hold others up.

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	ec2Client, err := aws.NewEC2Client(ctx)
	if err != nil {
		return err
	}

	// Figure out what kind of instance we are and how many NICs and IPs we can support.
	netCaps, err := ec2Client.GetMyNetworkCapabilities(ctx)
	if err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		"netCaps": netCaps,
	}).Info("Retrieved my instance's network capabilities")

	// Collect the current state of this instance and our NICs according to AWS.
	myNICs, err := ec2Client.GetMyEC2NetworkInterfaces(ctx)
	a.awsNICsByID = map[string]ec2types.NetworkInterface{}
	nicIDsBySubnet := map[string][]string{}
	nicIDByIP := map[ip.CIDR]string{}
	nicIDByPrimaryIP := map[ip.CIDR]string{}
	inUseDeviceIndexes := map[int32]bool{}
	freeIPv4CapacityByNICID := map[string]int{}
	attachmentIDByNICID := map[string]string{}
	var primaryNIC *ec2types.NetworkInterface
	for _, n := range myNICs {
		if n.NetworkInterfaceId == nil {
			logrus.Debug("AWS NIC had no NetworkInterfaceId.")
			continue
		}
		if n.Attachment != nil {
			if n.Attachment.DeviceIndex != nil {
				inUseDeviceIndexes[*n.Attachment.DeviceIndex] = true
			}
			if n.Attachment.AttachmentId != nil {
				attachmentIDByNICID[*n.NetworkInterfaceId] = *n.Attachment.AttachmentId
			}
		}
		if !aws.NetworkInterfaceIsCalicoSecondary(n) {
			if primaryNIC == nil || n.Attachment != nil && n.Attachment.DeviceIndex != nil && *n.Attachment.DeviceIndex == 0 {
				primaryNIC = &n
			}
			continue
		}
		// Found one of our managed interfaces; collect its IPs.
		logCtx := logrus.WithField("id", *n.NetworkInterfaceId)
		logCtx.Debug("Found Calico NIC")
		a.awsNICsByID[*n.NetworkInterfaceId] = n
		nicIDsBySubnet[*n.SubnetId] = append(nicIDsBySubnet[*n.SubnetId], *n.NetworkInterfaceId)
		for _, addr := range n.PrivateIpAddresses {
			if addr.PrivateIpAddress == nil {
				continue
			}
			cidr := ip.MustParseCIDROrIP(*addr.PrivateIpAddress)
			if addr.Primary != nil && *addr.Primary {
				logCtx.WithField("ip", *addr.PrivateIpAddress).Debug("Found primary IP on Calico NIC")
				nicIDByPrimaryIP[cidr] = *n.NetworkInterfaceId
			} else {
				logCtx.WithField("ip", *addr.PrivateIpAddress).Debug("Found secondary IP on Calico NIC")
				nicIDByIP[cidr] = *n.NetworkInterfaceId
			}
		}
		freeIPv4CapacityByNICID[*n.NetworkInterfaceId] = netCaps.MaxIPv4PerInterface - len(n.PrivateIpAddresses)
		logCtx.WithField("availableIPs", freeIPv4CapacityByNICID[*n.NetworkInterfaceId]).Debug("Calculated available IPs")
		if freeIPv4CapacityByNICID[*n.NetworkInterfaceId] < 0 {
			logCtx.Errorf("NIC appears to have more IPs (%v) that it should (%v)", len(n.PrivateIpAddresses), netCaps.MaxIPv4PerInterface)
			freeIPv4CapacityByNICID[*n.NetworkInterfaceId] = 0
		}
	}

	// Scan for IPs that are present on our AWS NICs but no longer required by Calico.
	ipsToRelease := set.New()
	for addr, nicID := range nicIDByIP {
		if _, ok := a.localAWSRoutesByDst[addr]; !ok {
			logrus.WithFields(logrus.Fields{
				"addr":  addr,
				"nidID": nicID,
			}).Info("AWS Secondary IP no longer needed")
			ipsToRelease.Add(addr)
		}
	}

	// Figure out the subnets that live in our AZ.  We can only create NICs within these subnets.
	localSubnets, err := ec2Client.GetAZLocalSubnets(ctx)
	if err != nil {
		return err
	}
	localSubnetIDs := set.New()
	for _, s := range localSubnets {
		if s.SubnetId == nil {
			continue
		}
		localSubnetIDs.Add(*s.SubnetId)
	}
	logrus.WithField("subnets", localSubnetIDs).Info("Looked up local AWS Subnets.")
	localIPPoolSubnetIDs := set.New()
	for subnetID := range a.poolIDsBySubnetID {
		if localSubnetIDs.Contains(subnetID) {
			localIPPoolSubnetIDs.Add(subnetID)
		}
	}
	logrus.WithField("subnets", localIPPoolSubnetIDs).Info("AWS Subnets with associated Calico IP pool.")

	// Scan for NICs that are in a subnet that no longer matches an IP pool.
	nicsToRelease := set.New()
	for nicID, nic := range a.awsNICsByID {
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

	// Figure out what IPs we're missing.
	var missingRoutes []*proto.RouteUpdate
	for addr, route := range a.localAWSRoutesByDst {
		if !localSubnetIDs.Contains(route.AwsSubnetId) {
			logrus.WithFields(logrus.Fields{
				"addr":           addr,
				"requiredSubnet": route.AwsSubnetId,
			}).Warn("Local workload needs an IP from an AWS subnet that is not accessible from this " +
				"availability zone. Unable to allocate an AWS IP for it.")
			continue
		}
		if nicID, ok := nicIDByPrimaryIP[addr]; ok {
			logrus.WithFields(logrus.Fields{
				"addr": addr,
				"nic":  nicID,
			}).Warn("Local workload IP clashes with host's primary IP on one of its secondary interfaces.")
			continue
		}
		if nicID, ok := nicIDByIP[addr]; ok {
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

	// We only support a single local subnet, choose one based on some heuristics.
	bestSubnet := a.calculateBestSubnet(localIPPoolSubnetIDs, nicIDsBySubnet)
	if bestSubnet == "" {
		logrus.Debug("No AWS subnets needed.")
		return nil
	}

	// Record the gateway address of the best subnet.
	{
		for _, subnet := range localSubnets {
			if subnet.SubnetId != nil && *subnet.SubnetId == bestSubnet {
				ourSubnet := subnet
				if ourSubnet.CidrBlock == nil {
					return fmt.Errorf("our subnet missing its CIDR id=%s", bestSubnet) // AWS bug?
				}
				ourCIDR, err := ip.ParseCIDROrIP(*ourSubnet.CidrBlock)
				if err != nil {
					return fmt.Errorf("our subnet had malformed CIDR %q: %w", *ourSubnet.CidrBlock, err)
				}
				addr := ourCIDR.Addr().Add(1)
				if addr != a.awsGatewayAddr {
					a.awsGatewayAddr = addr
					logrus.WithField("addr", a.awsGatewayAddr).Info("Calculated new AWS gateway.")
				}
			}
		}
	}

	// Release any IPs that are no longer required.
	ipsToRelease.Iter(func(item interface{}) error {
		addr := item.(ip.CIDR)
		nicID := nicIDByIP[addr]
		_, err := ec2Client.EC2Svc.UnassignPrivateIpAddresses(ctx, &ec2.UnassignPrivateIpAddressesInput{
			NetworkInterfaceId: &nicID,
			// TODO batch up all updates for same NIC?
			PrivateIpAddresses: []string{addr.Addr().String()},
		})
		if err != nil {
			logrus.WithError(err).Error("Failed to release AWS IP.")
			return nil
		}
		delete(nicIDByIP, addr)
		freeIPv4CapacityByNICID[nicID]++
		return nil
	})

	if nicsToRelease.Len() > 0 {
		// Release any NICs we no longer want.
		nicsToRelease.Iter(func(item interface{}) error {
			nicID := item.(string)
			attachID := attachmentIDByNICID[nicID]
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
			// Worth trying this even if detach fails.  Possible someone the failure was caused by it already
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

		// Go again from the top so we don't need to try fixing up all the maps.
		return errResyncNeeded
	}

	// Given the selected subnet, filter down the routes to only those that we can support.
	filteredRoutes := filterRoutesByAWSSubnet(missingRoutes, bestSubnet)
	if len(filteredRoutes) == 0 {
		logrus.Debug("No new AWS IPs to program")
		return nil
	}
	logrus.WithField("numNewRoutes", len(filteredRoutes)).Info("Need to program new AWS IPs")

	// Look for any AWS interfaces that belong to this node (as recorded in a tag that we attach to the node)
	// but are not actually attached to this node.
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
		devIdx := int32(0)
		for inUseDeviceIndexes[devIdx] {
			devIdx++
		}

		subnetID := safeReadString(nic.SubnetId)
		if subnetID != bestSubnet || int(devIdx) >= netCaps.MaxNetworkInterfaces {
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
		inUseDeviceIndexes[devIdx] = true
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
		}).Info("Attached the loose NIC.")
		attachedOrphan = true
	}
	if attachedOrphan {
		// Laziness!  Avoid recalculating indexes after attaching the orphaned NICs.
		logrus.Info("Restarting resync after cleaning up loose interfaces.")
		return errResyncNeeded
	}

	// TODO Using the node name here for consistency with tunnel IPs but I'm not sure if nodeName can change on AWS?
	handle := fmt.Sprintf("aws-secondary-ifaces-%s", a.nodeName)

	// Now we've cleaned up any unneeded NICs. Free any IPs that are assigned to us in IPAM but not in use for
	// one of our NICs.
	{
		ourIPs, err := a.ipamClient.IPsByHandle(ctx, handle)
		if err != nil && !errors.Is(err, calierrors.ErrorResourceDoesNotExist{}) {
			return fmt.Errorf("failed to look up our existing IPs: %w", err)
		}
		for _, addr := range ourIPs {
			cidr := ip.CIDRFromNetIP(addr.IP)
			if _, ok := nicIDByPrimaryIP[cidr]; !ok {
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
	}

	// TODO clean up any NICs that are missing from IPAM?  Shouldn't be possible but would be good to do.

	// Allocate IPs for the new NICs
	totalIPs := a.localRouteDestsBySubnetID[bestSubnet].Len()
	if netCaps.MaxIPv4PerInterface <= 1 {
		logrus.Error("Instance type doesn't support secondary IPs")
		return fmt.Errorf("instance type doesn't support secondary IPs")
	}
	secondaryIPsPerIface := netCaps.MaxIPv4PerInterface - 1
	totalNICsNeeded := (totalIPs + secondaryIPsPerIface - 1) / secondaryIPsPerIface
	nicsAlreadyAllocated := len(nicIDsBySubnet[bestSubnet])
	numNICsNeeded := totalNICsNeeded - nicsAlreadyAllocated

	if numNICsNeeded > 0 {
		ipamCtx, ipamCancel := context.WithTimeout(context.Background(), 90*time.Second)

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
			return err
		}
		if v4addrs == nil || len(v4addrs.IPs) == 0 {
			return fmt.Errorf("failed to allocate IP for secondary interface: %v", v4addrs.Msgs)
		}
		logrus.WithField("ips", v4addrs.IPs).Info("Allocated primary IPs for secondary interfaces")
		if len(v4addrs.IPs) < numNICsNeeded {
			logrus.WithFields(logrus.Fields{
				"needed":    numNICsNeeded,
				"allocated": len(v4addrs.IPs),
			}).Warn("Wasn't able to allocate enough ENI primary IPs. IP pool may be full.")
		}

		// Figure out the security groups of our primary NIC, we'll copy these to the new interfaces that we create.
		var securityGroups []string
		for _, sg := range primaryNIC.Groups {
			if sg.GroupId == nil {
				continue
			}
			securityGroups = append(securityGroups, *sg.GroupId)
		}

		// Create the new NICs for the IPs we were able to get.
		for _, addr := range v4addrs.IPs {
			ipStr := addr.IP.String()
			token := fmt.Sprintf("calico-secondary-%s-%s", ec2Client.InstanceID, ipStr)
			cno, err := ec2Client.EC2Svc.CreateNetworkInterface(ctx, &ec2.CreateNetworkInterfaceInput{
				SubnetId:         &bestSubnet,
				ClientToken:      &token,
				Description:      stringPointer(fmt.Sprintf("Calico secondary NIC for instance %s", ec2Client.InstanceID)),
				Groups:           securityGroups,
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
				continue
			}

			// Find a free device index.
			devIdx := int32(0)
			for inUseDeviceIndexes[devIdx] {
				devIdx++
			}
			inUseDeviceIndexes[devIdx] = true
			attOut, err := ec2Client.EC2Svc.AttachNetworkInterface(ctx, &ec2.AttachNetworkInterfaceInput{
				DeviceIndex:        &devIdx,
				InstanceId:         &ec2Client.InstanceID,
				NetworkInterfaceId: cno.NetworkInterface.NetworkInterfaceId,
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
			}).Info("Attached NIC.")

			// Calculate the free IPs from the output. Once we add an idempotency token, it'll be possible to have
			// >1 IP in place already.
			freeIPv4CapacityByNICID[*cno.NetworkInterface.NetworkInterfaceId] = netCaps.MaxIPv4PerInterface -
				len(cno.NetworkInterface.PrivateIpAddresses)

			// TODO disable source/dest check?
		}
	}

	// Assign secondary IPs to NICs.
	for nicID, freeIPs := range freeIPv4CapacityByNICID {
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
			// TODO What now?
		}
		logrus.WithFields(logrus.Fields{"nicID": nicID, "addrs": ipAddrs}).Info("Assigned IPs to secondary NIC.")
	}

	// TODO update k8s Node with capacities
	// Report health

	return nil
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

func (a *awsSubnetManager) calculateBestSubnet(localIPPoolSubnetIDs set.Set, nicIDsBySubnet map[string][]string) string {
	// If the IP pools only name one then that is preferred.  If there's more than one in the IP pools but we've already
	// got a local NIC, that one is preferred.  If there's a tie, pick the one with the most routes.
	subnetScores := map[string]int{}
	localIPPoolSubnetIDs.Iter(func(item interface{}) error {
		subnetID := item.(string)
		subnetScores[subnetID] += 1000000
		return nil
	})
	for subnet, nicIDs := range nicIDsBySubnet {
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

func (a *awsSubnetManager) resyncWithDataplane() error {
	// TODO Listen for interface updates
	if a.awsGatewayAddr == nil {
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
		if awsNIC.NetworkInterfaceId == nil {
			continue // Very unlikely.
		}
		logrus.WithFields(logrus.Fields{
			"mac":      mac,
			"name":     ifaceName,
			"awsNICID": awsNIC.NetworkInterfaceId,
		}).Debug("Matched local NIC with AWs NIC.")

		// Enable the NIC.
		err := netlink.LinkSetUp(iface)
		if err != nil {
			ifaceName := iface.Attrs().Name
			logrus.WithError(err).WithField("name", ifaceName).Error("Failed to set link up")
		}

		// For each IP assigned to the NIC, we'll add a routing rule that sends traffic _from_ that IP to
		// a dedicated routing table for the NIC.
		routingTableID := a.getOrAllocRoutingTableID(ifaceName)

		for _, privateIP := range awsNIC.PrivateIpAddresses {
			if privateIP.PrivateIpAddress == nil {
				continue
			}
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
					Type: routetable.TargetTypeOnLink,
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
		a.routeTables[tableIndex] = routetable.New(
			[]string{"^" + ifaceName + "$", routetable.InterfaceNone},
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
