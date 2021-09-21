// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package aws

import (
	"context"
	encoding_binary "encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/testutils"
)

const (
	instanceTypeT0Pico types.InstanceType = "t0.pico"
)

type fakeEC2 struct {
	lock sync.Mutex

	Errors testutils.ErrorProducer

	InstancesByID map[string]types.Instance
	NICsByID      map[string]types.NetworkInterface
	SubnetsByID   map[string]types.Subnet

	nextNICNum    int
	nextAttachNum int
}

func newFakeEC2Client() (*EC2Client, *fakeEC2) {
	mockEC2 := &fakeEC2{
		InstancesByID: map[string]types.Instance{},
		NICsByID:      map[string]types.NetworkInterface{},
		SubnetsByID:   map[string]types.Subnet{},

		Errors: testutils.NewErrorProducer(testutils.WithErrFactory(func(queueName string) error {
			return errBadParam(queueName, "ErrorFactory.Error")
		})),

		nextNICNum:    0x1000,
		nextAttachNum: 0x1000,
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

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

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

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

	panic("fakeEC2 doesn't support requested feature")
}

func (f *fakeEC2) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

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

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

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
	case instanceTypeT0Pico:
		// Made up type without any secondary ENI capacity.
		return &ec2.DescribeInstanceTypesOutput{
			InstanceTypes: []types.InstanceTypeInfo{
				{
					InstanceType: instanceTypeT0Pico,
					NetworkInfo: &types.NetworkInfo{
						Ipv4AddressesPerInterface: int32Ptr(1),
						Ipv6AddressesPerInterface: int32Ptr(1),
						Ipv6Supported:             boolPtr(true),
						MaximumNetworkCards:       int32Ptr(1),
						MaximumNetworkInterfaces:  int32Ptr(2),
						NetworkCards: []types.NetworkCardInfo{
							{
								MaximumNetworkInterfaces: int32Ptr(2),
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

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

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
		for _, filter := range params.Filters {
			filterMatches := false
			switch *filter.Name {
			case "attachment.instance-id":
				for _, v := range filter.Values {
					if nic.Attachment != nil && nic.Attachment.InstanceId != nil && *nic.Attachment.InstanceId == v {
						filterMatches = true
						break
					}
				}
			case "status":
				for _, v := range filter.Values {
					if string(nic.Status) == v {
						filterMatches = true
						break
					}
				}
			case "tag:calico:instance":
				for _, v := range filter.Values {
					for _, tag := range nic.TagSet {
						if *tag.Key == "calico:instance" && *tag.Value == v {
							filterMatches = true
							break
						}
					}
				}
			default:
				panic("fakeEC2 doesn't understand filter " + *filter.Name)
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

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

	if params.DryRun != nil || len(params.PrivateIpAddresses) > 0 {
		panic("fakeEC2 doesn't support requested feature")
	}
	if *params.SubnetId != subnetIDWest1Calico && *params.SubnetId != subnetIDWest1CalicoAlt {
		panic("wrong subnet ID" + *params.SubnetId)
	}
	if params.PrivateIpAddress == nil {
		panic("expected specific IP address")
	}

	nicID := fmt.Sprintf("eni-%017x", f.nextNICNum)
	mac := make(net.HardwareAddr, 6)
	encoding_binary.BigEndian.PutUint32(mac[2:], uint32(f.nextNICNum))
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
	var sgs []types.GroupIdentifier

	for _, g := range params.Groups {
		sgs = append(sgs, types.GroupIdentifier{
			GroupId:   stringPtr(g),
			GroupName: stringPtr(g + " name"),
		})
	}

	nic := types.NetworkInterface{
		NetworkInterfaceId: stringPtr(nicID),
		SubnetId:           params.SubnetId,
		Description:        params.Description,
		Attachment: &types.NetworkInterfaceAttachment{
			Status: types.AttachmentStatusDetached,
		},
		AvailabilityZone: stringPtr(azWest1),
		Groups:           sgs,
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

	logrus.WithField("nic", spew.Sdump(nic)).Info("FakeEC2: Created NIC.")

	return &ec2.CreateNetworkInterfaceOutput{
		NetworkInterface: &nic,
	}, nil
}

func (f *fakeEC2) AttachNetworkInterface(ctx context.Context, params *ec2.AttachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.AttachNetworkInterfaceOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

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
	nic.Status = types.NetworkInterfaceStatusAssociated

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

func (f *fakeEC2) DetachNetworkInterface(ctx context.Context, params *ec2.DetachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DetachNetworkInterfaceOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

	if params.Force == nil || !*params.Force {
		panic("Expecting use of Force.")
	}

	var instID string
	found := false
	for nicID, nic := range f.NICsByID {
		if nic.Attachment != nil && nic.Attachment.AttachmentId != nil && *nic.Attachment.AttachmentId == *params.AttachmentId {
			logrus.WithField("id", nicID).Info("FakeEC2 found NIC to dettach.")
			nic.Status = types.NetworkInterfaceStatusAvailable
			instID = *nic.Attachment.InstanceId
			nic.Attachment = nil
			f.NICsByID[nicID] = nic
			found = true
		}
	}
	if !found {
		return nil, errNotFound("DetachNetworkInterface", "AttachmentId.NotFound")
	}

	inst, ok := f.InstancesByID[instID]
	if !ok {
		panic("FakeEC2: BUG, couldn't find instance for NIC attachment")
	}
	var updatedNICs []types.InstanceNetworkInterface
	found = false
	for _, nic := range inst.NetworkInterfaces {
		if *nic.Attachment.AttachmentId == *params.AttachmentId {
			found = true
			continue
		}
		updatedNICs = append(updatedNICs, nic)
	}
	if !found {
		panic("FakeEC2: BUG, couldn't find NIC on instance")
	}
	inst.NetworkInterfaces = updatedNICs
	f.InstancesByID[instID] = inst

	return &ec2.DetachNetworkInterfaceOutput{ /* not currently used by caller */ }, nil
}

func (f *fakeEC2) AssignPrivateIpAddresses(ctx context.Context, params *ec2.AssignPrivateIpAddressesInput, optFns ...func(*ec2.Options)) (*ec2.AssignPrivateIpAddressesOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

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

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

	if params.NetworkInterfaceId == nil {
		return nil, errBadParam("UnassignPrivateIpAddresses", "NetworkInterfaceId.Missing")
	}

	if len(params.PrivateIpAddresses) == 0 {
		panic("BUG: releasing 0 IPs?")
	}

	// Find the NIC.
	nic, ok := f.NICsByID[*params.NetworkInterfaceId]
	if !ok {
		return nil, errNotFound("UnassignPrivateIpAddresses", "NIC.NotFound")
	}
	for _, newAddr := range params.PrivateIpAddresses {
		var updatedAddrs []types.NetworkInterfacePrivateIpAddress
		found := false
		for _, addr := range nic.PrivateIpAddresses {
			if *addr.PrivateIpAddress == newAddr {
				found = true
				continue
			}
			updatedAddrs = append(updatedAddrs, addr)
		}
		if !found {
			return nil, errNotFound("UnassignPrivateIpAddresses", "Address.NotFound")
		}
		nic.PrivateIpAddresses = updatedAddrs
	}

	f.NICsByID[*params.NetworkInterfaceId] = nic

	return &ec2.UnassignPrivateIpAddressesOutput{
		// Not currently used so not bothering to fill in
	}, nil
}

func (f *fakeEC2) DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if err := f.Errors.NextErrorByCaller(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(optFns) > 0 {
		panic("fakeEC2 doesn't understand opts")
	}

	if params.NetworkInterfaceId == nil {
		panic("BUG: caller should supply network interface ID")
	}

	nic, ok := f.NICsByID[*params.NetworkInterfaceId]
	if !ok {
		return nil, errNotFound("DeleteNetworkInterface", "NetworkInterfaceId.NotFound")
	}

	if nic.Status != types.NetworkInterfaceStatusAvailable {
		return nil, errBadParam("DeleteNetworkInterface", "NetworkInterface.IsAttached")
	}

	delete(f.NICsByID, *params.NetworkInterfaceId)
	return &ec2.DeleteNetworkInterfaceOutput{ /* not used by caller */ }, nil
}

func (f *fakeEC2) NumNICs() int {
	f.lock.Lock()
	defer f.lock.Unlock()

	return len(f.NICsByID)
}

func (f *fakeEC2) GetNIC(eniid string) types.NetworkInterface {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.NICsByID[eniid]
}
