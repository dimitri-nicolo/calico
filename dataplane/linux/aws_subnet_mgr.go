// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"context"
	"fmt"
	"time"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

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
}

const (
	healthNameSubnetCapacity = "have-at-most-one-aws-subnet"
	healthNameAWSInSync      = "aws-enis-in-sync"
)

func newAWSSubnetManager(
	healthAgg *health.HealthAggregator,
	ipamClient ipam.Interface,
	k8sClient *kubernetes.Clientset,
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
		resyncNeeded: true,
		healthAgg:    healthAgg,
		ipamClient:   ipamClient,
		k8sClient:    k8sClient,
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
		a.poolIDsBySubnetID[oldSubnetID].Discard(id)
		if a.poolIDsBySubnetID[oldSubnetID].Len() == 0 {
			delete(a.poolIDsBySubnetID, oldSubnetID)
		}
	}
	if newSubnetID != "" && oldSubnetID != newSubnetID {
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
	a.resyncNeeded = true
}

func (a awsSubnetManager) onRouteUpdate(dst ip.CIDR, route *proto.RouteUpdate) {
	if route != nil && !route.LocalWorkload {
		return // We only care about local workload routes.
	}
	if route != nil && route.AwsSubnetId == "" {
		return // We only care about AWS routes.
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
	}
	if newSubnetID != "" && oldSubnetID != newSubnetID {
		if _, ok := a.localRouteDestsBySubnetID[newSubnetID]; !ok {
			a.localRouteDestsBySubnetID[newSubnetID] = set.New()
		}
		a.localRouteDestsBySubnetID[newSubnetID].Add(dst)
	}

	// Save off the route itself.
	if route == nil {
		if _, ok := a.localAWSRoutesByDst[dst]; !ok {
			return // Not a route we were tracking.
		}
		delete(a.localAWSRoutesByDst, dst)
	} else {
		a.localAWSRoutesByDst[dst] = route
	}
	a.resyncNeeded = true
}

type awsNICInfo struct {
	awsNIC *ec2.NetworkInterface
}

func (a awsSubnetManager) CompleteDeferredWork() error {
	if !a.resyncNeeded {
		return nil
	}

	err := a.resync()
	if err != nil {
		return err
	}
	a.resyncNeeded = false

	return nil
}

func (a awsSubnetManager) resync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
	for _, n := range myNICs {
		if !aws.NetworkInterfaceIsCalicoSecondary(n) {
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
	}

	// Scan for IPs thatare present on our AWS NICs but no longer required by Calico.
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
	localIPPoolSubnetIDs := set.New()
	for subnetID := range a.poolIDsBySubnetID {
		if localSubnetIDs.Contains(subnetID) {
			localIPPoolSubnetIDs.Add(subnetID)
		}
	}

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
	if len(filteredRoutes) == 0 {
		logrus.Debug("No new AWS IPs to program")
		return nil
	}
	logrus.WithField("numNewRoutes", len(filteredRoutes)).Info("Need to program new AWS IPs")

	// Allocate new NICs where needed.
	totalIPs := a.localRouteDestsBySubnetID[bestSubnet].Len()
	if netCaps.MaxIPv4PerInterface <= 1 {
		logrus.Error("Instance type doesn't support secondary IPs")
		return fmt.Errorf("instance type doesn't support secondary IPs")
	}
	totalNICsNeeded := totalIPs / (netCaps.MaxIPv4PerInterface - 1)
	nicsAlreadyAllocated := len(nicIDsBySubnet[bestSubnet])
	newNICsNeeded := totalNICsNeeded - nicsAlreadyAllocated
	for i := 0; i < newNICsNeeded; i++ {
		logrus.WithField("subnet", bestSubnet).Info("Allocating new AWS NIC.")

	}

	// Assign IPs to NICs.

	// TODO update k8s Node with capacities
	// Report health

	return nil
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

func NewAWSSubnetManager() *awsSubnetManager {
	return &awsSubnetManager{}
}

var _ Manager = &awsSubnetManager{}

//
// func foo() {
// 	// Get my instance, find my AZ.
// 	// Maintain set of awsSubnets referenced by IP pools.
// 	// When set changes:
// 	// - Figure out which subnets are in my AZ.
// 	// - Deterministically choose one subnet (there should only be one).
// 	// - When subnet changes (or start of day), reconcile Node resource allocation and local ENI(s).
//
// 	// Start of day:
// 	// - Get my instance type
// 	// - Used c.EC2Svc.DescribeInstanceTypes() to get the number of interfaces and IPs-per-interface
//
// 	// Reconcile node resource:
// 	// Get my Calico Node resource
// 	// Find name of k8s node from it (OrchRef)
// 	// Get my k8s node using k8s client
// 	// - Scan its extended resources.
// 	// - Patch out extended resources availability for subnets that we no longer have.
// 	// - Patch in extended resources availability for the subnet we now have.
//
// 	// Find the interfaces attached ot this instance already.
//
// 	// Check if we've already got the interfaces that we _want_.
//
// 	// Remove any interfaces that we no longer want.
//
// 	// For each interface that we're missing:
// 	// - Figure out <next available device ID> on this instance.
// 	//   - If there are no available slots, error.
// 	// - Search for any unattached interfaces that match our tags.
// 	// - If not found, create a new interface.
// 	// - Attach the found/created interface to <next available device ID>
// 	// - Delete any interfaces that turned up in the search that we no longer want.
//
// 	// Free any secondary IPs that no longer apply to local pods.
// 	// Claim any IPs that now apply to local pods.
//
// 	descIfacesOut, _ := c.EC2Svc.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
// 		Filters:             nil,
// 		MaxResults:          nil,
// 		NetworkInterfaceIds: nil,
// 		NextToken:           nil,
// 	})
//
// 	primaryIP := "10.0.0.1"
// 	nodeName := "my-node"
// 	subnetID := "sn-12345"
// 	c.EC2Svc.CreateNetworkInterface(
// 		&ec2.CreateNetworkInterfaceInput{
// 			Description:      stringPtr("Calico NIC for instance abcd1234"),
// 			Groups:           []*string{stringPtr("sg-12345")},
// 			PrivateIpAddress: stringPtr(primaryIP),
// 			SubnetId:         stringPtr(subnetID),
// 			TagSpecifications: []*ec2.TagSpecification{
// 				{
// 					ResourceType: stringPtr("network-interface"),
// 					Tags: []*ec2.Tag{{
// 						Key:   stringPtr("projectcalico.org/node"),
// 						Value: stringPtr(nodeName),
// 					}},
// 				},
// 			},
// 		},
// 	)
//
// 	c.EC2Svc.AttachNetworkInterface(&ec2.AttachNetworkInterfaceInput{
// 		DeviceIndex:        nil,
// 		DryRun:             nil,
// 		InstanceId:         nil,
// 		NetworkCardIndex:   nil,
// 		NetworkInterfaceId: nil,
// 	})
// }
