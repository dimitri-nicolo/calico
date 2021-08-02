// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

type awsSubnetManager struct {
}

func (a awsSubnetManager) OnUpdate(protoBufMsg interface{}) {

}

func (a awsSubnetManager) CompleteDeferredWork() error {
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
