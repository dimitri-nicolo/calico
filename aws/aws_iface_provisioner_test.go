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
	encoding_binary "encoding/binary"
	"errors"
	"fmt"
	"net"
	nethttp "net/http"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/transport/http"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/util/clock"

	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/ipam"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
)

const (
	nodeName   = "test-node"
	instanceID = "i-ca1ic000000000001"
	testVPC    = "vpc-01234567890123456"

	primaryNICID       = "eni-00000000000000001"
	primaryNICAttachID = "attach-00000000000000001"
	primaryNICMAC      = "00:00:00:00:00:01"

	azWest1 = "us-west-1"
	azWest2 = "us-west-2"

	subnetIDWest1Calico  = "subnet-ca100000000000001"
	subnetIDWest2Calico  = "subnet-ca100000000000002"
	subnetIDWest1Default = "subnet-def00000000000001"
	subnetIDWest2Default = "subnet-def00000000000002"

	subnetWest1CIDRCalico    = "100.64.1.0/24"
	subnetWest1GatewayCalico = "100.64.1.1"
	subnetWest2CIDRCalico    = "100.64.2.0/24"

	calicoHostIP1 = "100.64.1.5"
	calicoHostIP2 = "100.64.1.6"

	wl1Addr = "100.64.1.64/32"

	ipPoolIDWest1Hosts    = "pool-west-1-hosts"
	ipPoolIDWest2Hosts    = "pool-west-2-hosts"
	ipPoolIDWest1Gateways = "pool-west-1-gateways"
	ipPoolIDWest2Gateways = "pool-west-2-gateways"
)

type sipTestFakes struct {
	IPAM  *fakeIPAM
	EC2   *fakeEC2
	Clock *clock.FakeClock
}

func setup(t *testing.T) (*SecondaryIfaceProvisioner, *sipTestFakes) {
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
	fakeEC2.addSubnet(subnetIDWest1Calico, azWest1, subnetWest1CIDRCalico)
	fakeEC2.addSubnet(subnetIDWest2Calico, azWest2, subnetWest2CIDRCalico)

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
		MacAddress:       stringPtr(primaryNICMAC),
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

	return sip, &sipTestFakes{
		IPAM:  fakeIPAM,
		EC2:   fakeEC2,
		Clock: fakeClock,
	}
}

func setupAndStart(t *testing.T) (*SecondaryIfaceProvisioner, *sipTestFakes, func()) {
	sip, fake := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	doneC := sip.Start(ctx)
	return sip, fake, func() {
		cancel()
		Eventually(doneC).Should(BeClosed())
	}
}

func TestSecondaryIfaceProvisioner_NoPoolsOrWorkloadsMainline(t *testing.T) {
	sip, _, cancel := setupAndStart(t)
	defer cancel()

	// Send an empty snapshot.
	sip.OnDatastoreUpdate(DatastoreState{
		LocalAWSRoutesByDst:       nil,
		LocalRouteDestsBySubnetID: nil,
		PoolIDsBySubnetID:         nil,
	})

	// Should get an empty response.
	Eventually(sip.responseC).Should(Receive(Equal(&IfaceState{})))
}

func TestSecondaryIfaceProvisioner_AWSPoolsButNoWorkloadsMainline(t *testing.T) {
	sip, _, cancel := setupAndStart(t)
	defer cancel()

	sip.OnDatastoreUpdate(DatastoreState{
		LocalAWSRoutesByDst:       nil,
		LocalRouteDestsBySubnetID: nil,
		PoolIDsBySubnetID: map[string]set.Set{
			subnetIDWest1Calico: set.FromArray([]string{ipPoolIDWest1Hosts, ipPoolIDWest1Gateways}),
			subnetIDWest2Calico: set.FromArray([]string{ipPoolIDWest2Hosts, ipPoolIDWest2Gateways}),
		},
	})

	// Should respond with the Calico subnet details for the node's AZ..
	Eventually(sip.responseC).Should(Receive(Equal(&IfaceState{
		PrimaryNICMAC:      primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{},
		SubnetCIDR:         ip.MustParseCIDROrIP(subnetWest1CIDRCalico),
		GatewayAddr:        ip.FromString(subnetWest1GatewayCalico),
	})))
}

func TestSecondaryIfaceProvisioner_AWSPoolsSingleWorkloadMainline(t *testing.T) {
	sip, _, cancel := setupAndStart(t)
	defer cancel()

	wl1CIDR := ip.MustParseCIDROrIP(wl1Addr)
	sip.OnDatastoreUpdate(DatastoreState{
		LocalAWSRoutesByDst: map[ip.CIDR]*proto.RouteUpdate{
			wl1CIDR: &proto.RouteUpdate{
				Dst:           wl1Addr,
				LocalWorkload: true,
				AwsSubnetId:   subnetIDWest1Calico,
			},
		},
		LocalRouteDestsBySubnetID: map[string]set.Set{
			subnetIDWest1Calico: set.FromArray([]ip.CIDR{wl1CIDR}),
		},
		PoolIDsBySubnetID: map[string]set.Set{
			subnetIDWest1Calico: set.FromArray([]string{ipPoolIDWest1Hosts, ipPoolIDWest1Gateways}),
			subnetIDWest2Calico: set.FromArray([]string{ipPoolIDWest2Hosts, ipPoolIDWest2Gateways}),
		},
	})

	// Should respond with the Calico subnet details for the node's AZ..
	mac, err := net.ParseMAC("00:00:e8:03:00:00")
	if err != nil {
		panic(err)
	}
	Eventually(sip.responseC).Should(Receive(Equal(&IfaceState{
		PrimaryNICMAC:      primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{
			"00:00:e8:03:00:00": {
				ID: "eni-000000000000003e8",
				MAC: mac,
				PrimaryIPv4Addr: ip.FromString(calicoHostIP1),
				SecondaryIPv4Addrs: []ip.Addr{ip.MustParseCIDROrIP(wl1Addr).Addr()},
			},
		},
		SubnetCIDR:         ip.MustParseCIDROrIP(subnetWest1CIDRCalico),
		GatewayAddr:        ip.FromString(subnetWest1GatewayCalico),
	})))
}

type fakeEC2 struct {
	lock sync.Mutex

	InstancesByID map[string]types.Instance
	NICsByID      map[string]types.NetworkInterface
	SubnetsByID   map[string]types.Subnet

	nextNICNum    int
	nextAttachNum int
}

func newEC2Client() (*EC2Client, *fakeEC2) {
	mockEC2 := &fakeEC2{
		InstancesByID: map[string]types.Instance{},
		NICsByID:      map[string]types.NetworkInterface{},
		SubnetsByID:   map[string]types.Subnet{},

		nextNICNum:    1000,
		nextAttachNum: 1000,
	}
	return &EC2Client{
		EC2Svc:     mockEC2,
		InstanceID: instanceID,
	}, mockEC2
}

func (f *fakeEC2) nextAttachID() string {
	id := fmt.Sprintf("attach-%017x", f.nextAttachNum)
	f.nextAttachNum++
	return id
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
	f.lock.Lock()
	defer f.lock.Unlock()

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
	f.lock.Lock()
	defer f.lock.Unlock()

	panic("fakeEC2 doesn't support requested feature")
}

func (f *fakeEC2) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

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
	f.lock.Lock()
	defer f.lock.Unlock()

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
	f.lock.Lock()
	defer f.lock.Unlock()

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
	f.lock.Lock()
	defer f.lock.Unlock()

	if params.DryRun != nil || len(params.PrivateIpAddresses) > 0 {
		panic("fakeEC2 doesn't support requested feature")
	}
	if *params.SubnetId != subnetIDWest1Calico {
		panic("wrong subnet ID" + *params.SubnetId)
	}
	if params.PrivateIpAddress == nil {
		panic("expected specific IP address")
	}

	nicID := fmt.Sprintf("eni-%017x", f.nextNICNum)
	mac := make(net.HardwareAddr, 6)
	encoding_binary.LittleEndian.PutUint32(mac[2:], uint32(f.nextNICNum))
	f.nextNICNum++

	var tags []types.Tag
	for _, tagSpec := range params.TagSpecifications {
		if tagSpec.ResourceType != "network-interface" {
			panic("tag spec missing incorrect resource type")
		}
		for _, t := range tagSpec.Tags {
			tags = append(tags, t)
		}
	}

	nic := types.NetworkInterface{
		NetworkInterfaceId: stringPtr(nicID),
		SubnetId:           params.SubnetId,
		Description:        params.Description,
		Attachment: &types.NetworkInterfaceAttachment{
			Status: types.AttachmentStatusDetached,
		},
		AvailabilityZone: stringPtr(azWest1),
		Groups:           nil, // FIXME
		InterfaceType:    "eni",
		MacAddress:       stringPtr(mac.String()),
		PrivateIpAddress: params.PrivateIpAddress,
		PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
			{
				Primary:          boolPtr(true),
				PrivateIpAddress: params.PrivateIpAddress,
			},
		},
		SourceDestCheck: boolPtr(true),
		Status:          types.NetworkInterfaceStatusAvailable,
		TagSet:          tags,
		VpcId:           stringPtr(testVPC),
	}
	f.NICsByID[nicID] = nic

	return &ec2.CreateNetworkInterfaceOutput{
		NetworkInterface: &nic,
	}, nil
}

func (f *fakeEC2) AttachNetworkInterface(ctx context.Context, params *ec2.AttachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.AttachNetworkInterfaceOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if params.DryRun != nil || params.NetworkCardIndex != nil && *params.NetworkCardIndex != 0 {
		panic("fakeEC2 doesn't support requested feature")
	}

	if params.InstanceId == nil || params.NetworkInterfaceId == nil {
		panic("missing instance ID or NIC ID on attach call")
	}

	inst, ok := f.InstancesByID[*params.InstanceId]
	if !ok {
		return nil, errNotFound("AttachNetworkInterface", "InstanceId.NotFound")
	}
	nic, ok := f.NICsByID[*params.NetworkInterfaceId]
	if !ok {
		return nil, errNotFound("AttachNetworkInterface", "NetworkInterfaceId.NotFound")
	}

	if nic.Attachment != nil && nic.Attachment.InstanceId != nil {
		return nil, errBadParam("AttachNetworkInterface", "NetworkInterface.AlreadyAttached")
	}

	for _, ni := range inst.NetworkInterfaces {
		if *ni.Attachment.DeviceIndex == *params.DeviceIndex {
			return nil, errBadParam("AttachNetworkInterface", "DeviceIndex.Conflict")
		}
	}

	nic.Attachment = &types.NetworkInterfaceAttachment{
		AttachmentId:        stringPtr(f.nextAttachID()),
		DeleteOnTermination: boolPtr(false),
		DeviceIndex:         params.DeviceIndex,
		InstanceId:          params.InstanceId,
		NetworkCardIndex:    int32Ptr(0),
		Status:              types.AttachmentStatusAttached,
	}

	var privIPs []types.InstancePrivateIpAddress
	for _, ip := range nic.PrivateIpAddresses {
		privIPs = append(privIPs, types.InstancePrivateIpAddress{
			Primary:          ip.Primary,
			PrivateIpAddress: ip.PrivateIpAddress,
		})
	}
	inst.NetworkInterfaces = append(inst.NetworkInterfaces, types.InstanceNetworkInterface{
		Association: nil,
		Attachment: &types.InstanceNetworkInterfaceAttachment{
			AttachmentId:        nic.Attachment.AttachmentId,
			DeleteOnTermination: boolPtr(false),
			DeviceIndex:         params.DeviceIndex,
			NetworkCardIndex:    int32Ptr(0),
			Status:              types.AttachmentStatusAttached,
		},
		Description:        nic.Description,
		Groups:             nic.Groups,
		InterfaceType:      stringPtr(string(nic.InterfaceType)),
		MacAddress:         nic.MacAddress,
		NetworkInterfaceId: params.NetworkInterfaceId,
		PrivateIpAddress:   nic.PrivateIpAddress,
		PrivateIpAddresses: privIPs,
		SourceDestCheck:    nic.SourceDestCheck,
		Status:             types.NetworkInterfaceStatusAssociated,
		SubnetId:           nic.SubnetId,
		VpcId:              nic.VpcId,
	})

	f.InstancesByID[*params.InstanceId] = inst
	f.NICsByID[*params.NetworkInterfaceId] = nic

	return &ec2.AttachNetworkInterfaceOutput{
		AttachmentId:     nic.Attachment.AttachmentId,
		NetworkCardIndex: int32Ptr(0),
	}, nil
}

func (f *fakeEC2) AssignPrivateIpAddresses(ctx context.Context, params *ec2.AssignPrivateIpAddressesInput, optFns ...func(*ec2.Options)) (*ec2.AssignPrivateIpAddressesOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if params.NetworkInterfaceId == nil {
		return nil, errBadParam("AssignPrivateIpAddresses", "NetworkInterfaceId.Missing")
	}

	if params.AllowReassignment == nil || !*params.AllowReassignment {
		panic("BUG: expecting AllowReassignment to be set")
	}

	if len(params.PrivateIpAddresses) == 0 {
		panic("BUG: assigning 0 IPs?")
	}
	if params.SecondaryPrivateIpAddressCount != nil {
		panic("fakeEC2 doesn't support AWS IPAM")
	}

	// Find the NIC.
	nic := f.NICsByID[*params.NetworkInterfaceId]
	for _, newAddr := range params.PrivateIpAddresses {
		for _, addr := range nic.PrivateIpAddresses {
			if *addr.PrivateIpAddress == newAddr {
				return nil, errBadParam("AssignPrivateIpAddresses", "Address.AlreadyAssigned")
			}
		}
	}

	for _, newAddr := range params.PrivateIpAddresses {
		nic.PrivateIpAddresses = append(nic.PrivateIpAddresses, types.NetworkInterfacePrivateIpAddress{
			Primary:          boolPtr(false),
			PrivateIpAddress: stringPtr(newAddr),
		})
	}
	f.NICsByID[*params.NetworkInterfaceId] = nic

	for nicID, nic := range f.NICsByID {
		if nicID == *params.NetworkInterfaceId {
			continue
		}
		for _, newAddr := range params.PrivateIpAddresses {
			for i, addr := range nic.PrivateIpAddresses {
				if *addr.PrivateIpAddress == newAddr {
					// Other NIC has this IP, delete it.
					nic.PrivateIpAddresses[i] = nic.PrivateIpAddresses[len(nic.PrivateIpAddresses)-1]
					nic.PrivateIpAddresses = nic.PrivateIpAddresses[:len(nic.PrivateIpAddresses)-1]
				}
			}
		}
		f.NICsByID[nicID] = nic
	}

	return &ec2.AssignPrivateIpAddressesOutput{
		// Not currently used so not bothering to fill in
	}, nil
}

func (f *fakeEC2) UnassignPrivateIpAddresses(ctx context.Context, params *ec2.UnassignPrivateIpAddressesInput, optFns ...func(*ec2.Options)) (*ec2.UnassignPrivateIpAddressesOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	panic("implement me")
}

func (f *fakeEC2) DetachNetworkInterface(ctx context.Context, params *ec2.DetachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DetachNetworkInterfaceOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	panic("implement me")
}

func (f *fakeEC2) DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	panic("implement me")
}

type ipamAlloc struct {
	Addr string
	Args ipam.AutoAssignArgs
}

type fakeIPAM struct {
	lock sync.Mutex

	freeIPs     []string
	requests    []ipam.AutoAssignArgs
	allocations []ipamAlloc
}

func (m *fakeIPAM) Allocations() []ipamAlloc {
	m.lock.Lock()
	defer m.lock.Unlock()
	out := make([]ipamAlloc, len(m.allocations))
	copy(out, m.allocations)
	return out
}

func (m *fakeIPAM) AutoAssign(ctx context.Context, args ipam.AutoAssignArgs) (*ipam.IPAMAssignments, *ipam.IPAMAssignments, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.requests = append(m.requests, args)
	if args.Num6 > 0 {
		panic("IPV6 not supported")
	}
	if args.Num4 <= 0 {
		panic("expected some v4 addresses")
	}
	if args.HandleID == nil {
		return nil, nil, errors.New("missing handle")
	}
	if args.Hostname == "" {
		return nil, nil, errors.New("missing hostname")
	}
	if args.IntendedUse != v3.IPPoolAllowedUseHostSecondary {
		return nil, nil, errors.New("expected AllowedUseHostSecondary")
	}

	v4Allocs := &ipam.IPAMAssignments{
		IPs:          nil,
		IPVersion:    4,
		NumRequested: args.Num4,
	}
	for i := 0; i < args.Num4; i++ {
		if len(m.freeIPs) == 0 {
			return v4Allocs, nil, errors.New("couldn't alloc all IPs")
		}
		chosenIP := m.freeIPs[0]
		m.allocations = append(m.allocations, ipamAlloc{
			Addr: chosenIP,
			Args: args,
		})
		m.freeIPs = m.freeIPs[1:]
		_, addr, err := cnet.ParseCIDROrIP(chosenIP)
		if err != nil {
			panic("failed to parse test IP")
		}
		v4Allocs.IPs = append(v4Allocs.IPs, *addr)
	}

	return v4Allocs, nil, nil
}

func (m *fakeIPAM) ReleaseIPs(ctx context.Context, ips []cnet.IP) ([]cnet.IP, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	panic("implement me")
}

func (m *fakeIPAM) IPsByHandle(ctx context.Context, handleID string) ([]cnet.IP, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	panic("implement me")
}

func newMockIPAM() *fakeIPAM {
	return &fakeIPAM{
		freeIPs: []string{
			calicoHostIP1,
			calicoHostIP2,
		},
	}
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

func errBadParam(op string, code string) error {
	return &smithy.OperationError{
		ServiceID:     "EC2",
		OperationName: op,
		Err: &http.ResponseError{
			Response: &http.Response{
				Response: &nethttp.Response{
					StatusCode: 400,
				},
			},
			Err: &smithy.GenericAPIError{
				Code:    code,
				Message: "Bad paremeter",
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
