// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/projectcalico/felix/aws"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type awsSubnetManager struct {
	poolsByID                 map[string]*proto.IPAMPool
	poolIDsBySubnetID         map[string]set.Set
	localAWSRoutesByDst       map[string]*proto.RouteUpdate
	localRouteDestsBySubnetID map[string]set.Set

	resyncNeeded bool
}

func newAWSSubnetManager() *awsSubnetManager {
	return &awsSubnetManager{
		resyncNeeded: true,
	}
}

func (a awsSubnetManager) OnUpdate(msg interface{}) {
	switch msg := msg.(type) {
	case *proto.IPAMPoolUpdate:
		a.onPoolUpdate(msg.Id, msg.Pool)
	case *proto.IPAMPoolRemove:
		a.onPoolUpdate(msg.Id, nil)
	case *proto.RouteUpdate:
		a.onRouteUpdate(msg.Dst, msg)
	case *proto.RouteRemove:
		a.onRouteUpdate(msg.Dst, nil)
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

func (a awsSubnetManager) onRouteUpdate(dst string, route *proto.RouteUpdate) {
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

func (a awsSubnetManager) CompleteDeferredWork() error {
	if !a.resyncNeeded {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ec2Client, err := aws.NewEC2Client(ctx)
	if err != nil {
		return err // FIXME
	}

	ec2Client.EC2Svc.DescribeNetworkInterfaces()
	ec2Client.EC2Svc.CreateNetworkInterface(nil)

	// Collect information about where this node is in AWS

	return nil
}

func NewAWSSubnetManager() *awsSubnetManager {
	return &awsSubnetManager{}
}

var _ Manager = &awsSubnetManager{}

func (c *ec2Client) foo() {
	// Get my instance, find my AZ.
	// Maintain set of awsSubnets referenced by IP pools.
	// When set changes:
	// - Figure out which subnets are in my AZ.
	// - Deterministically choose one subnet (there should only be one).
	// - When subnet changes (or start of day), reconcile Node resource allocation and local ENI(s).

	// Start of day:
	// - Get my instance type
	// - Used c.EC2Svc.DescribeInstanceTypes() to get the number of interfaces and IPs-per-interface

	// Reconcile node resource:
	// Get my Calico Node resource
	// Find name of k8s node from it (OrchRef)
	// Get my k8s node using k8s client
	// - Scan its extended resources.
	// - Patch out extended resources availability for subnets that we no longer have.
	// - Patch in extended resources availability for the subnet we now have.

	// Find the interfaces attached ot this instance already.

	// Check if we've already got the interfaces that we _want_.

	// Remove any interfaces that we no longer want.

	// For each interface that we're missing:
	// - Figure out <next available device ID> on this instance.
	//   - If there are no available slots, error.
	// - Search for any unattached interfaces that match our tags.
	// - If not found, create a new interface.
	// - Attach the found/created interface to <next available device ID>
	// - Delete any interfaces that turned up in the search that we no longer want.

	// Free any secondary IPs that no longer apply to local pods.
	// Claim any IPs that now apply to local pods.

	descIfacesOut, _ := c.EC2Svc.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters:             nil,
		MaxResults:          nil,
		NetworkInterfaceIds: nil,
		NextToken:           nil,
	})

	primaryIP := "10.0.0.1"
	nodeName := "my-node"
	subnetID := "sn-12345"
	c.EC2Svc.CreateNetworkInterface(
		&ec2.CreateNetworkInterfaceInput{
			Description:      stringPtr("Calico NIC for instance abcd1234"),
			Groups:           []*string{stringPtr("sg-12345")},
			PrivateIpAddress: stringPtr(primaryIP),
			SubnetId:         stringPtr(subnetID),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: stringPtr("network-interface"),
					Tags: []*ec2.Tag{{
						Key:   stringPtr("projectcalico.org/node"),
						Value: stringPtr(nodeName),
					}},
				},
			},
		},
	)

	c.EC2Svc.AttachNetworkInterface(&ec2.AttachNetworkInterfaceInput{
		DeviceIndex:        nil,
		DryRun:             nil,
		InstanceId:         nil,
		NetworkCardIndex:   nil,
		NetworkInterfaceId: nil,
	})
}

func stringPtr(s string) *string {
	return &s
}
