// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
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

	resyncNeeded bool

	healthAgg  *health.HealthAggregator
	ipamClient ipam.Interface
	k8sClient  *kubernetes.Clientset
	nodeName   string
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

	return &awsSubnetManager{
		poolsByID:                 map[string]*proto.IPAMPool{},
		poolIDsBySubnetID:         map[string]set.Set{},
		localAWSRoutesByDst:       map[ip.CIDR]*proto.RouteUpdate{},
		localRouteDestsBySubnetID: map[string]set.Set{},
		resyncNeeded:              true,
		healthAgg:                 healthAgg,
		ipamClient:                ipamClient,
		k8sClient:                 k8sClient,
		nodeName:                  nodeName,
	}
}

func (a awsSubnetManager) OnUpdate(msg interface{}) {
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

func (a awsSubnetManager) onPoolUpdate(id string, pool *proto.IPAMPool) {
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
			"newSubnet":newSubnetID,
			"pool":id,
		}).Info("IP pool no longer associated with AWS subnet.")
		a.poolIDsBySubnetID[oldSubnetID].Discard(id)
		if a.poolIDsBySubnetID[oldSubnetID].Len() == 0 {
			delete(a.poolIDsBySubnetID, oldSubnetID)
		}
	}
	if newSubnetID != "" && oldSubnetID != newSubnetID {
		logrus.WithFields(logrus.Fields{
			"oldSubnet": oldSubnetID,
			"newSubnet":newSubnetID,
			"pool":id,
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

func (a awsSubnetManager) onRouteUpdate(dst ip.CIDR, route *proto.RouteUpdate) {
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

func (a awsSubnetManager) queueResync(reason string) {
	if a.resyncNeeded {
		return
	}
	logrus.WithField("reason", reason).Info("Resync needed")
	a.resyncNeeded = true
}

func (a awsSubnetManager) CompleteDeferredWork() error {
	if !a.resyncNeeded {
		return nil
	}

	err := a.resync()
	if err != nil {
		logrus.WithError(err).Warn("Failed to resync AWS subnet state.")
		return err
	}
	logrus.Info("Resync completed successfully.")
	a.resyncNeeded = false

	return nil
}

func (a awsSubnetManager) resync() error {
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
	secondaryNICsByID := map[string]ec2types.NetworkInterface{}
	nicIDsBySubnet := map[string][]string{}
	nicIDByIP := map[ip.CIDR]string{}
	nicIDByPrimaryIP := map[ip.CIDR]string{}
	inUseDeviceIndexes := map[int32]bool{}
	freeIPv4CapacityByNICID := map[string]int{}
	var primaryNIC *ec2types.NetworkInterface
	for _, n := range myNICs {
		if n.Attachment != nil && n.Attachment.DeviceIndex != nil {
			inUseDeviceIndexes[*n.Attachment.DeviceIndex] = true
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
		secondaryNICsByID[*n.NetworkInterfaceId] = n
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
	for nicID, nic := range secondaryNICsByID {
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

	// TODO AWS update phase:

	// TODO Free IPs that are no longer needed.
	// TODO Update nicIDByIP et al after deletions.

	// TODO Free NICs that are no longer needed.
	// TODO Update secondaryNICsByID et al after deletions.

	// Given the selected subnet, filter down the routes to only those that we can support.
	filteredRoutes := filterRoutesByAWSSubnet(missingRoutes, bestSubnet)
	if len(filteredRoutes) == 0 {
		logrus.Debug("No new AWS IPs to program")
		return nil
	}
	logrus.WithField("numNewRoutes", len(filteredRoutes)).Info("Need to program new AWS IPs")

	// TODO Look up NICs that we created but failed to attach and re-use them or clean them up (if they're from the wrong subnet)
	// TODO Look up any existing IPs we have in IPAM and use one of those rather than allocating new.

	// Allocate IPs for the new NICs
	totalIPs := a.localRouteDestsBySubnetID[bestSubnet].Len()
	if netCaps.MaxIPv4PerInterface <= 1 {
		logrus.Error("Instance type doesn't support secondary IPs")
		return fmt.Errorf("instance type doesn't support secondary IPs")
	}
	secondaryIPsPerIface := netCaps.MaxIPv4PerInterface - 1
	totalNICsNeeded := (totalIPs + secondaryIPsPerIface-1) / secondaryIPsPerIface
	nicsAlreadyAllocated := len(nicIDsBySubnet[bestSubnet])
	numNICsNeeded := totalNICsNeeded - nicsAlreadyAllocated

	if numNICsNeeded > 0 {
		ipamCtx, ipamCancel := context.WithTimeout(context.Background(), 90*time.Second)

		// TODO Using the node name here for consistency with tunnel IPs but I'm not sure if nodeName can change on AWS?
		handle := fmt.Sprintf("aws-secondary-ifaces-%s", a.nodeName)
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

func (a awsSubnetManager) calculateBestSubnet(localIPPoolSubnetIDs set.Set, nicIDsBySubnet map[string][]string) string {
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

var _ Manager = &awsSubnetManager{}
