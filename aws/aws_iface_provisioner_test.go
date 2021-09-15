// Copyright (c) 2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright (c) 2021  All rights reserved.

package aws

import (
	"context"
	nethttp "net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/transport/http"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/clock"

	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	"github.com/projectcalico/libcalico-go/lib/net"
)

const (
	nodeName   = "test-node"
	instanceID = "i-ca1ic000000000001"
	testVPC    = "vpc-01234567890123456"

	primaryNICID       = "eni-00000000000000001"
	primaryNICAttachID = "attach-00000000000000001"

	azWest1 = "us-west-1"
	azWest2 = "us-west-2"

	subnetIDWest1Calico  = "subnet-ca100000000000001"
	subnetIDWest2Calico  = "subnet-ca100000000000002"
	subnetIDWest1Default = "subnet-def00000000000001"
	subnetIDWest2Default = "subnet-def00000000000002"
)

func TestIfaceProvMainline(t *testing.T) {
	RegisterTestingT(t)
	fakeIPAM := newMockIPAM()
	fakeClock := clock.NewFakeClock(time.Now())
	capacityC := make(chan SecondaryIfaceCapacities, 10)
	ec2Client, fakeEC2 := newEC2Client()

	fakeEC2.InstancesByID[instanceID] = types.Instance{
		InstanceId:   stringPtr(instanceID),
		InstanceType: types.InstanceTypeT3Large,
		Placement: &types.Placement{
			AvailabilityZone: stringPtr(azWest1),
		},
		VpcId: stringPtr(testVPC),
	}
	fakeEC2.addSubnet(subnetIDWest1Default, azWest1, "192.164.1.0/24")
	fakeEC2.addSubnet(subnetIDWest2Default, azWest2, "192.164.2.0/24")
	fakeEC2.addSubnet(subnetIDWest1Calico, azWest1, "100.64.1.0/24")
	fakeEC2.addSubnet(subnetIDWest2Calico, azWest2, "100.64.2.0/24")

	fakeEC2.NICsByID[primaryNICID] = types.NetworkInterface{
		NetworkInterfaceId: stringPtr(primaryNICID),
		Attachment: &types.NetworkInterfaceAttachment{
			DeviceIndex:      int32Ptr(0),
			NetworkCardIndex: int32Ptr(0),
			AttachmentId:     stringPtr(primaryNICAttachID),
			InstanceId:       stringPtr(instanceID),
		},
		SubnetId: stringPtr(subnetIDWest1Default),
		PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
			{
				Primary:          boolPtr(true),
				PrivateIpAddress: stringPtr("192.164.1.5"),
			},
		},
		PrivateIpAddress: stringPtr("192.164.1.5"),
	}

	sip := NewSecondaryIfaceProvisioner(
		nodeName,
		health.NewHealthAggregator(),
		fakeIPAM,
		OptClockOverride(fakeClock),
		OptCapacityCallback(func(capacities SecondaryIfaceCapacities) {
			capacityC <- capacities
		}),
		OptNewEC2ClientkOverride(func(ctx context.Context) (*EC2Client, error) {
			return ec2Client, nil
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	doneC := sip.Start(ctx)
	defer func() {
		cancel()
		Eventually(doneC).Should(BeClosed())
	}()

	// Send an empty snapshot.
	sip.OnDatastoreUpdate(DatastoreState{
		LocalAWSRoutesByDst:       nil,
		LocalRouteDestsBySubnetID: nil,
		PoolIDsBySubnetID:         nil,
	})

	var responseState *IfaceState
	Eventually(sip.responseC).Should(Receive(&responseState))

	Expect(responseState).To(Equal(&IfaceState{
		PrimaryNIC:         nil,
		SecondaryNICsByMAC: nil,
		SubnetCIDR:         nil,
		GatewayAddr:        nil,
	}))
}

type fakeEC2 struct {
	InstancesByID      map[string]types.Instance
	NICAttachmentsByID map[string]types.InstanceNetworkInterfaceAttachment
	NICsByID           map[string]types.NetworkInterface
	SubnetsByID        map[string]types.Subnet
}

func newEC2Client() (*EC2Client, *fakeEC2) {
	mockEC2 := &fakeEC2{
		InstancesByID:      map[string]types.Instance{},
		NICAttachmentsByID: map[string]types.InstanceNetworkInterfaceAttachment{},
		NICsByID:           map[string]types.NetworkInterface{},
		SubnetsByID:        map[string]types.Subnet{},
	}
	return &EC2Client{
		EC2Svc:     mockEC2,
		InstanceID: instanceID,
	}, mockEC2
}

func (f *fakeEC2) addSubnet(id string, az string, cidr string) {
	f.SubnetsByID[id] = types.Subnet{
		AvailabilityZone: stringPtr(az),
		VpcId:            stringPtr(testVPC),
		SubnetId:         stringPtr(id),
		CidrBlock:        stringPtr(cidr),
	}
}

func (f *fakeEC2) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	if len(params.InstanceIds) != 1 {
		panic("fakeEC2 can't handle !=1 instance ID")
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}
	if inst, ok := f.InstancesByID[params.InstanceIds[0]]; !ok {
		return nil, errNotFound("DescribeInstances", "InvalidInstanceID.NotFound")
	} else {
		return &ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{
				{
					Instances: []types.Instance{
						inst,
					},
				},
			},
		}, nil
	}
}

func (f *fakeEC2) ModifyNetworkInterfaceAttribute(ctx context.Context, params *ec2.ModifyNetworkInterfaceAttributeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyNetworkInterfaceAttributeOutput, error) {
	panic("fakeEC2 doesn't support requested feature")
}

func (f *fakeEC2) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	if params.DryRun != nil || params.MaxResults != nil || params.NextToken != nil {
		panic("fakeEC2 doesn't support requested feature")
	}

	var subnets []types.Subnet
	for _, subnet := range f.SubnetsByID {
		allFiltersMatch := true
		for _, f := range params.Filters {
			filterMatches := false
			switch *f.Name {
			case "availability-zone":
				for _, v := range f.Values {
					if *subnet.AvailabilityZone == v {
						filterMatches = true
						break
					}
				}
			case "vpc-id":
				for _, v := range f.Values {
					if *subnet.VpcId == v {
						filterMatches = true
						break
					}
				}
			default:
				panic("fakeEC2 doesn't understand filter " + *f.Name)
			}
			allFiltersMatch = allFiltersMatch && filterMatches
		}
		if !allFiltersMatch {
			continue
		}

		// NIC matches
		subnets = append(subnets, subnet)
	}

	return &ec2.DescribeSubnetsOutput{
		Subnets: subnets,
	}, nil
}

func (f *fakeEC2) DescribeInstanceTypes(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	if params.DryRun != nil || params.MaxResults != nil || params.NextToken != nil {
		panic("fakeEC2 doesn't support requested feature")
	}
	if len(params.InstanceTypes) != 1 {
		panic("fakeEC2 can't handle !=1 instance type")
	}
	switch params.InstanceTypes[0] {
	case types.InstanceTypeT3Large:
		return &ec2.DescribeInstanceTypesOutput{
			InstanceTypes: []types.InstanceTypeInfo{
				{
					InstanceType: types.InstanceTypeT3Large,
					NetworkInfo: &types.NetworkInfo{
						Ipv4AddressesPerInterface: int32Ptr(12),
						Ipv6AddressesPerInterface: int32Ptr(12),
						Ipv6Supported:             boolPtr(true),
						MaximumNetworkCards:       int32Ptr(1),
						MaximumNetworkInterfaces:  int32Ptr(3),
						NetworkCards: []types.NetworkCardInfo{
							{
								MaximumNetworkInterfaces: int32Ptr(3),
								NetworkCardIndex:         int32Ptr(0),
							},
						},
					},
				},
			},
		}, nil
	default:
		panic("unknown instance type")
	}
}

func (f *fakeEC2) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	if params.DryRun != nil || params.MaxResults != nil || params.NextToken != nil {
		panic("fakeEC2 doesn't support requested feature")
	}

	var nics []types.NetworkInterface
	for nicID, nic := range f.NICsByID {
		nic := nic
		if params.NetworkInterfaceIds != nil {
			found := false
			for _, id := range params.NetworkInterfaceIds {
				if nicID == id {
					found = true
				}
			}
			if !found {
				continue
			}
		}

		allFiltersMatch := true
		for _, f := range params.Filters {
			filterMatches := false
			switch *f.Name {
			case "attachment.instance-id":
				for _, v := range f.Values {
					if *nic.Attachment.InstanceId == v {
						filterMatches = true
						break
					}
				}
			default:
				panic("fakeEC2 doesn't understand filter " + *f.Name)
			}
			allFiltersMatch = allFiltersMatch && filterMatches
		}
		if !allFiltersMatch {
			continue
		}

		// NIC matches
		nics = append(nics, nic)
	}

	// DescribeNetworkInterfaces seems to return an empty list rather than a not-found error.
	return &ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: nics,
	}, nil
}

func (f *fakeEC2) CreateNetworkInterface(ctx context.Context, params *ec2.CreateNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.CreateNetworkInterfaceOutput, error) {
	panic("implement me")
}

func (f *fakeEC2) AttachNetworkInterface(ctx context.Context, params *ec2.AttachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.AttachNetworkInterfaceOutput, error) {
	panic("implement me")
}

func (f *fakeEC2) AssignPrivateIpAddresses(ctx context.Context, params *ec2.AssignPrivateIpAddressesInput, optFns ...func(*ec2.Options)) (*ec2.AssignPrivateIpAddressesOutput, error) {
	panic("implement me")
}

func (f *fakeEC2) UnassignPrivateIpAddresses(ctx context.Context, params *ec2.UnassignPrivateIpAddressesInput, optFns ...func(*ec2.Options)) (*ec2.UnassignPrivateIpAddressesOutput, error) {
	panic("implement me")
}

func (f *fakeEC2) DetachNetworkInterface(ctx context.Context, params *ec2.DetachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DetachNetworkInterfaceOutput, error) {
	panic("implement me")
}

func (f *fakeEC2) DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error) {
	panic("implement me")
}

type fakeIPAM struct {
}

func (m *fakeIPAM) AutoAssign(ctx context.Context, args ipam.AutoAssignArgs) (*ipam.IPAMAssignments, *ipam.IPAMAssignments, error) {
	panic("implement me")
}

func (m *fakeIPAM) ReleaseIPs(ctx context.Context, ips []net.IP) ([]net.IP, error) {
	panic("implement me")
}

func (m *fakeIPAM) IPsByHandle(ctx context.Context, handleID string) ([]net.IP, error) {
	panic("implement me")
}

func newMockIPAM() ipamInterface {
	return &fakeIPAM{}
}

// errNotFound returns an error with the same structure as the AWSv2 client returns.  The code under test
// unwraps errors with errors.As() so it's important that we return something that's the right shape.
func errNotFound(op string, code string) error {
	return &smithy.OperationError{
		ServiceID:     "EC2",
		OperationName: op,
		Err: &http.ResponseError{
			Response: &http.Response{
				Response: &nethttp.Response{
					StatusCode: 403,
				},
			},
			Err: &smithy.GenericAPIError{
				Code:    code,
				Message: "The XXX does not exist",
				Fault:   0,
			},
		},
	}
}

func errUnauthorized(op string) error {
	return &smithy.OperationError{
		ServiceID:     "EC2",
		OperationName: op,
		Err: &http.ResponseError{
			Response: &http.Response{
				Response: &nethttp.Response{
					StatusCode: 403,
				},
			},
			Err: &smithy.GenericAPIError{
				Code:    "UnauthorizedOperation",
				Message: "You are not authorized to perform this operation",
				Fault:   0,
			},
		},
	}
}
