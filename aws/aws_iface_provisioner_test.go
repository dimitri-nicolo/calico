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
	"github.com/davecgh/go-spew/spew"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/util/clock"

	"github.com/projectcalico/felix/testutils"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/felix/proto"
	"github.com/projectcalico/libcalico-go/lib/set"

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

	subnetIDWest1Calico    = "subnet-ca100000000000001"
	subnetIDWest1CalicoAlt = "subnet-ca100000000000011"
	subnetIDWest2Calico    = "subnet-ca100000000000002"
	subnetIDWest1Default   = "subnet-def00000000000001"
	subnetIDWest2Default   = "subnet-def00000000000002"

	subnetWest1CIDRCalico       = "100.64.1.0/24"
	subnetWest1CIDRCalicoAlt    = "100.64.3.0/24"
	subnetWest1GatewayCalico    = "100.64.1.1"
	subnetWest1GatewayCalicoAlt = "100.64.3.1"
	subnetWest2CIDRCalico       = "100.64.2.0/24"

	calicoHostIP1    = "100.64.1.5"
	calicoHostIP1Alt = "100.64.3.5"
	calicoHostIP2    = "100.64.1.6"

	wl1Addr    = "100.64.1.64/32"
	wl1AddrAlt = "100.64.3.64/32"

	ipPoolIDWest1Hosts       = "pool-west-1-hosts"
	ipPoolIDWest1HostsAlt    = "pool-west-1-hosts-alt"
	ipPoolIDWest2Hosts       = "pool-west-2-hosts"
	ipPoolIDWest1Gateways    = "pool-west-1-gateways"
	ipPoolIDWest1GatewaysAlt = "pool-west-1-gateways-alt"
	ipPoolIDWest2Gateways    = "pool-west-2-gateways"

	t3LargeCapacity = 22
)

var (
	wl1CIDR    = ip.MustParseCIDROrIP(wl1Addr)
	wl1CIDRAlt = ip.MustParseCIDROrIP(wl1AddrAlt)

	defaultPools = map[string]set.Set{
		subnetIDWest1Calico: set.FromArray([]string{ipPoolIDWest1Hosts, ipPoolIDWest1Gateways}),
		subnetIDWest2Calico: set.FromArray([]string{ipPoolIDWest2Hosts, ipPoolIDWest2Gateways}),
	}

	alternatePools = map[string]set.Set{
		subnetIDWest1CalicoAlt: set.FromArray([]string{ipPoolIDWest1HostsAlt, ipPoolIDWest1GatewaysAlt}),
		subnetIDWest2Calico:    set.FromArray([]string{ipPoolIDWest2Hosts, ipPoolIDWest2Gateways}),
	}

	noWorkloadDatastore = DatastoreState{
		LocalAWSRoutesByDst:       nil,
		LocalRouteDestsBySubnetID: nil,
		PoolIDsBySubnetID:         defaultPools,
	}

	noWorkloadDatastoreAltPools = DatastoreState{
		LocalAWSRoutesByDst:       nil,
		LocalRouteDestsBySubnetID: nil,
		PoolIDsBySubnetID:         alternatePools,
	}

	singleWorkloadDatastore = DatastoreState{
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
		PoolIDsBySubnetID: defaultPools,
	}
	singleWorkloadDatastoreAltPool = DatastoreState{
		LocalAWSRoutesByDst: map[ip.CIDR]*proto.RouteUpdate{
			wl1CIDR: &proto.RouteUpdate{
				Dst:           wl1AddrAlt,
				LocalWorkload: true,
				AwsSubnetId:   subnetIDWest1CalicoAlt,
			},
		},
		LocalRouteDestsBySubnetID: map[string]set.Set{
			subnetIDWest1CalicoAlt: set.FromArray([]ip.CIDR{wl1CIDRAlt}),
		},
		PoolIDsBySubnetID: alternatePools,
	}

	firstAllocatedMAC, _  = net.ParseMAC("00:00:00:00:10:00")
	secondAllocatedMAC, _ = net.ParseMAC("00:00:00:00:10:01")

	responsePoolsNoNICs = &IfaceState{
		PrimaryNICMAC:      primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{},
		SubnetCIDR:         ip.MustParseCIDROrIP(subnetWest1CIDRCalico),
		GatewayAddr:        ip.FromString(subnetWest1GatewayCalico),
	}
	responseSingleWorkload = &IfaceState{
		PrimaryNICMAC: primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{
			firstAllocatedMAC.String(): {
				ID:                 "eni-00000000000001000",
				MAC:                firstAllocatedMAC,
				PrimaryIPv4Addr:    ip.FromString(calicoHostIP1),
				SecondaryIPv4Addrs: []ip.Addr{ip.MustParseCIDROrIP(wl1Addr).Addr()},
			},
		},
		SubnetCIDR:  ip.MustParseCIDROrIP(subnetWest1CIDRCalico),
		GatewayAddr: ip.FromString(subnetWest1GatewayCalico),
	}
	responseNICAfterWorkloadsDeleted = &IfaceState{
		PrimaryNICMAC: primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{
			firstAllocatedMAC.String(): {
				ID:                 "eni-00000000000001000",
				MAC:                firstAllocatedMAC,
				PrimaryIPv4Addr:    ip.FromString(calicoHostIP1),
				SecondaryIPv4Addrs: nil,
			},
		},
		SubnetCIDR:  ip.MustParseCIDROrIP(subnetWest1CIDRCalico),
		GatewayAddr: ip.FromString(subnetWest1GatewayCalico),
	}
	singleWorkloadResponseAltHostIP = &IfaceState{
		PrimaryNICMAC: primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{
			firstAllocatedMAC.String(): {
				ID:                 "eni-00000000000001000",
				MAC:                firstAllocatedMAC,
				PrimaryIPv4Addr:    ip.FromString(calicoHostIP2), // Different IP
				SecondaryIPv4Addrs: []ip.Addr{ip.MustParseCIDROrIP(wl1Addr).Addr()},
			},
		},
		SubnetCIDR:  ip.MustParseCIDROrIP(subnetWest1CIDRCalico),
		GatewayAddr: ip.FromString(subnetWest1GatewayCalico),
	}

	responseAltPoolsNoNICs = &IfaceState{
		PrimaryNICMAC:      primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{},
		SubnetCIDR:         ip.MustParseCIDROrIP(subnetWest1CIDRCalicoAlt),
		GatewayAddr:        ip.FromString(subnetWest1GatewayCalicoAlt),
	}
	responseAltPoolsAfterWorkloadsDeleted = &IfaceState{
		PrimaryNICMAC: primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{
			secondAllocatedMAC.String(): {
				ID:                 "eni-00000000000001001",
				MAC:                secondAllocatedMAC,
				PrimaryIPv4Addr:    ip.FromString(calicoHostIP1Alt),
				SecondaryIPv4Addrs: nil,
			},
		},
		SubnetCIDR:  ip.MustParseCIDROrIP(subnetWest1CIDRCalicoAlt),
		GatewayAddr: ip.FromString(subnetWest1GatewayCalicoAlt),
	}
	responseAltPoolSingleWorkload = &IfaceState{
		PrimaryNICMAC: primaryNICMAC,
		SecondaryNICsByMAC: map[string]Iface{
			secondAllocatedMAC.String(): {
				ID:                 "eni-00000000000001001",
				MAC:                secondAllocatedMAC,
				PrimaryIPv4Addr:    ip.FromString(calicoHostIP1Alt),
				SecondaryIPv4Addrs: []ip.Addr{ip.MustParseCIDROrIP(wl1AddrAlt).Addr()},
			},
		},
		SubnetCIDR:  ip.MustParseCIDROrIP(subnetWest1CIDRCalicoAlt),
		GatewayAddr: ip.FromString(subnetWest1GatewayCalicoAlt),
	}
)

type sipTestFakes struct {
	IPAM      *fakeIPAM
	EC2       *fakeEC2
	Clock     *clock.FakeClock
	CapacityC chan SecondaryIfaceCapacities
}

func setup(t *testing.T) (*SecondaryIfaceProvisioner, *sipTestFakes) {
	RegisterTestingT(t)
	fakeIPAM := newMockIPAM()
	theTime, err := time.Parse("2006-01-02 15:04:05.000", "2021-09-15 16:00:00.000")
	Expect(err).NotTo(HaveOccurred())
	fakeClock := clock.NewFakeClock(theTime)
	capacityC := make(chan SecondaryIfaceCapacities, 1)
	ec2Client, fakeEC2 := newFakeEC2Client()

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
	fakeEC2.addSubnet(subnetIDWest1CalicoAlt, azWest1, subnetWest1CIDRCalicoAlt)
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
			// Drain any previous message.
			select {
			case <-capacityC:
			default:
			}
			capacityC <- capacities
		}),
		OptNewEC2ClientkOverride(func(ctx context.Context) (*EC2Client, error) {
			return ec2Client, nil
		}),
	)

	return sip, &sipTestFakes{
		IPAM:      fakeIPAM,
		EC2:       fakeEC2,
		Clock:     fakeClock,
		CapacityC: capacityC,
	}
}

func setupAndStart(t *testing.T) (*SecondaryIfaceProvisioner, *sipTestFakes, func()) {
	sip, fake := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	doneC := sip.Start(ctx)
	return sip, fake, func() {
		cancel()
		Eventually(doneC).Should(BeClosed())
		fake.EC2.Errors.ExpectAllErrorsConsumed()
	}
}

func TestSecondaryIfaceProvisioner_OnDatastoreUpdateShouldNotBlock(t *testing.T) {
	sip, _ := setup(t)

	// Hit on-update many times without starting the main loop, it should never block.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for x := 0; x < 1000; x++ {
			sip.OnDatastoreUpdate(DatastoreState{
				LocalAWSRoutesByDst:       nil,
				LocalRouteDestsBySubnetID: nil,
				PoolIDsBySubnetID:         nil,
			})
		}
	}()

	Eventually(done).Should(BeClosed())
}

func TestSecondaryIfaceProvisioner_NoPoolsOrWorkloadsStartOfDay(t *testing.T) {
	sip, fakes, tearDown := setupAndStart(t)
	defer tearDown()

	// Send an empty snapshot.
	sip.OnDatastoreUpdate(DatastoreState{
		LocalAWSRoutesByDst:       nil,
		LocalRouteDestsBySubnetID: nil,
		PoolIDsBySubnetID:         nil,
	})

	// Should get an empty response.
	Eventually(sip.ResponseC()).Should(Receive(Equal(&IfaceState{})))
	Eventually(fakes.CapacityC).Should(Receive(Equal(SecondaryIfaceCapacities{
		MaxCalicoSecondaryIPs: t3LargeCapacity,
	})))
}

func TestSecondaryIfaceProvisioner_AWSPoolsButNoWorkloadsMainline(t *testing.T) {
	sip, _, tearDown := setupAndStart(t)
	defer tearDown()

	sip.OnDatastoreUpdate(DatastoreState{
		LocalAWSRoutesByDst:       nil,
		LocalRouteDestsBySubnetID: nil,
		PoolIDsBySubnetID: map[string]set.Set{
			subnetIDWest1Calico: set.FromArray([]string{ipPoolIDWest1Hosts, ipPoolIDWest1Gateways}),
			subnetIDWest2Calico: set.FromArray([]string{ipPoolIDWest2Hosts, ipPoolIDWest2Gateways}),
		},
	})

	// Should respond with the Calico subnet details for the node's AZ..
	Eventually(sip.ResponseC()).Should(Receive(Equal(responsePoolsNoNICs)))
}

func TestSecondaryIfaceProvisioner_AWSPoolsSingleWorkload_Mainline(t *testing.T) {
	sip, fakes, tearDown := setupAndStart(t)
	defer tearDown()

	// Send snapshot with single workload.
	sip.OnDatastoreUpdate(singleWorkloadDatastore)

	// Since this is a fresh system with only one NIC being allocated, everything is deterministic and we should
	// always get the same result.
	Eventually(sip.ResponseC()).Should(Receive(Equal(responseSingleWorkload)))
	Eventually(fakes.CapacityC).Should(Receive(Equal(SecondaryIfaceCapacities{
		MaxCalicoSecondaryIPs: t3LargeCapacity,
	})))

	// Remove the workload again, IP should be released.
	sip.OnDatastoreUpdate(noWorkloadDatastore)
	Eventually(sip.ResponseC()).Should(Receive(Equal(responseNICAfterWorkloadsDeleted)))
}

func TestSecondaryIfaceProvisioner_AWSPoolsSingleWorkload_ErrBackoff(t *testing.T) {
	// Test that a range of different errors all result in a successful retry with backoff.
	// The fakeEC2 methods are all instrumented with the ErrorProducer so that we can make them fail
	// on command >:)

	for _, callToFail := range []string{
		"DescribeInstances",
		"DescribeNetworkInterfaces",
		"DescribeSubnets",
		"DescribeInstanceTypes",
		"DescribeNetworkInterfaces",
		"CreateNetworkInterface",
		"AttachNetworkInterface",
		"AssignPrivateIpAddresses",
	} {
		t.Run(callToFail, func(t *testing.T) {
			sip, fake, tearDown := setupAndStart(t)
			defer tearDown()

			// Queue up an error on a key AWS call. Note: tearDown() checks that all queued errors
			// were consumed so any typo in the name would be caught.
			fake.EC2.Errors.QueueError(callToFail)

			sip.OnDatastoreUpdate(singleWorkloadDatastore)

			// Should fail to respond.
			Consistently(sip.ResponseC()).ShouldNot(Receive())

			// Advance time to trigger the backoff.
			// Initial backoff should be between 1000 and 1100 ms (due to jitter).
			Expect(fake.Clock.HasWaiters()).To(BeTrue())
			fake.Clock.Step(999 * time.Millisecond)
			Expect(fake.Clock.HasWaiters()).To(BeTrue())
			fake.Clock.Step(102 * time.Millisecond)
			Expect(fake.Clock.HasWaiters()).To(BeFalse())

			// With only one NIC being added, FakeIPAM and FakeEC2 are deterministic.
			expResponse := responseSingleWorkload
			if callToFail == "CreateNetworkInterface" {
				// Failing CreateNetworkInterface triggers the allocated IP to be released and then a second
				// allocation performed.
				expResponse = singleWorkloadResponseAltHostIP
			}
			Eventually(sip.ResponseC()).Should(Receive(Equal(expResponse)))

			// Whether we did an IPAM reallocation or not, we should have only one IP in use at the end.
			Expect(fake.IPAM.NumUsedIPs()).To(BeNumerically("==", 1))
		})
	}
}

func TestSecondaryIfaceProvisioner_AWSPoolsSingleWorkload_ErrBackoffInterrupted(t *testing.T) {
	sip, fake, tearDown := setupAndStart(t)
	defer tearDown()

	// Queue up an error on a key AWS call.
	fake.EC2.Errors.QueueError("DescribeNetworkInterfaces")

	sip.OnDatastoreUpdate(singleWorkloadDatastore)

	// Should fail to respond.
	Consistently(sip.ResponseC()).ShouldNot(Receive())

	// Should be a timer waiting for backoff.
	Expect(fake.Clock.HasWaiters()).To(BeTrue())

	// Send a datastore update, should trigger the backoff to be abandoned.
	sip.OnDatastoreUpdate(singleWorkloadDatastore)

	// Since this is a fresh system with only one NIC being allocated, everything is deterministic and we should
	// always get the same result.
	Eventually(sip.ResponseC()).Should(Receive(Equal(responseSingleWorkload)))
	Expect(fake.Clock.HasWaiters()).To(BeFalse())
}

// TestSecondaryIfaceProvisioner_PoolChange Checks that changing the IP pools to use a different subnet causes the
// provisioner to release NICs and provision the new ones.
func TestSecondaryIfaceProvisioner_PoolChange(t *testing.T) {
	sip, fakes, tearDown := setupAndStart(t)
	defer tearDown()

	// Send snapshot with single workload on the original subnet.
	sip.OnDatastoreUpdate(singleWorkloadDatastore)

	// Since this is a fresh system with only one NIC being allocated, everything is deterministic and we should
	// always get the same result.
	Eventually(sip.ResponseC()).Should(Receive(Equal(responseSingleWorkload)))
	Eventually(fakes.CapacityC).Should(Receive(Equal(SecondaryIfaceCapacities{
		MaxCalicoSecondaryIPs: t3LargeCapacity,
	})))

	// Remove the workload again, IP should be released but NIC should stick around.
	sip.OnDatastoreUpdate(noWorkloadDatastore)
	Eventually(sip.ResponseC()).Should(Receive(Equal(responseNICAfterWorkloadsDeleted)))

	// Change the pools.
	sip.OnDatastoreUpdate(noWorkloadDatastoreAltPools)
	// Should get a response with updated gateway addresses _but_ no secondary NIC (because there was no workload
	// to trigger addition of the secondary NIC).
	Eventually(sip.ResponseC()).Should(Receive(Equal(responseAltPoolsNoNICs)))

	// Swap IPAM to prefer the alt host pool.  Normally the label selector on the pool would ensure the right
	// pool is used but we don't have that much function here.
	fakes.IPAM.setFreeIPs(calicoHostIP1Alt)

	// Add a workload in the alt pool, should get a secondary NIC using the alt pool.
	sip.OnDatastoreUpdate(singleWorkloadDatastoreAltPool)
	Eventually(sip.ResponseC()).Should(Receive(Equal(responseAltPoolSingleWorkload)))

	// Delete the workload.  Should keep the NIC but remove the secondary IP.
	sip.OnDatastoreUpdate(noWorkloadDatastoreAltPools)
	Eventually(sip.ResponseC()).Should(Receive(Equal(responseAltPoolsAfterWorkloadsDeleted)))
}

// TODO Security group copying
// TODO max out number of IPs
// TODO non-local workload
// TODO Local workload clashes with primary IP
// TODO Add second workload; first workload should be unaffected
// TODO UnassignPrivateIpAddresses fails
// TODO DetachNetworkInterface fails
// TODO DeleteNEtowrkInterface fails
// TODO Clean up orphan NICs

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
					if nic.Attachment.InstanceId != nil && *nic.Attachment.InstanceId == v {
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

type ipamAlloc struct {
	Addr   ip.Addr
	Handle string
	Args   ipam.AutoAssignArgs
}

type fakeIPAM struct {
	lock sync.Mutex

	freeIPs     []string
	requests    []ipam.AutoAssignArgs
	allocations []ipamAlloc
	origNumIPs  int
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
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}

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
			Addr:   ip.FromString(chosenIP),
			Handle: *args.HandleID,
			Args:   args,
		})
		m.freeIPs = m.freeIPs[1:]
		_, addr, err := cnet.ParseCIDROrIP(chosenIP)
		if err != nil {
			panic("failed to parse test IP")
		}
		v4Allocs.IPs = append(v4Allocs.IPs, *addr)
	}

	logrus.Infof("FakeIPAM allocation: %v", v4Allocs)

	return v4Allocs, nil, nil
}

func (m *fakeIPAM) ReleaseIPs(ctx context.Context, ips []cnet.IP) ([]cnet.IP, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	releaseCount := 0
	var out []cnet.IP
	var newAllocs []ipamAlloc
	for _, ipToRelease := range ips {
		logrus.Infof("Fake IPAM releasing IP: %v", ipToRelease)
		addrToRelease := ip.FromCalicoIP(ipToRelease)
		for _, alloc := range m.allocations {
			if alloc.Addr == addrToRelease {
				out = append(out, addrToRelease.AsCalicoNetIP())
				releaseCount++
				m.freeIPs = append(m.freeIPs, alloc.Addr.String())
				continue
			}
			newAllocs = append(newAllocs, alloc)
		}
	}
	m.allocations = newAllocs

	if releaseCount != len(ips) {
		// TODO not sure how calico IPAM handles this
		panic("asked to release non-allocated IP")
	}

	return out, nil
}

func (m *fakeIPAM) IPsByHandle(ctx context.Context, handleID string) ([]cnet.IP, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var out []cnet.IP
	for _, alloc := range m.allocations {
		if alloc.Handle == handleID {
			out = append(out, alloc.Addr.AsCalicoNetIP())
		}
	}
	logrus.Infof("Fake IPAM IPsByHandle %q = %v", handleID, out)

	return out, nil
}

func (m *fakeIPAM) NumFreeIPs() int {
	m.lock.Lock()
	defer m.lock.Unlock()

	return len(m.freeIPs)
}

func (m *fakeIPAM) NumUsedIPs() int {
	m.lock.Lock()
	defer m.lock.Unlock()

	return len(m.allocations)
}

func (m *fakeIPAM) setFreeIPs(ips ...string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.freeIPs = ips
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
