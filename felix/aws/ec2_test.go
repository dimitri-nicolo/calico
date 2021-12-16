// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

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

package aws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"k8s.io/apimachinery/pkg/util/clock"

	"github.com/projectcalico/calico/libcalico-go/lib/health"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testRegion  = "us-west-2"
	testEniId   = "eni-i-000"
	testEniId1  = "eni-i-001"
	testInstId  = "i-000"
	testInstId1 = "i-001"
)

type mockClient struct {
	UsageCounter int
}

func newMockClient() *mockClient {
	return &mockClient{UsageCounter: 0}
}

func (c *mockClient) GetInstanceIdentityDocument(
	ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options),
) (*imds.GetInstanceIdentityDocumentOutput, error) {
	c.UsageCounter++
	return &imds.GetInstanceIdentityDocumentOutput{InstanceIdentityDocument: imds.InstanceIdentityDocument{
		InstanceID: testInstId,
	}}, nil
}

func (c *mockClient) GetRegion(
	ctx context.Context, params *imds.GetRegionInput, optFns ...func(*imds.Options),
) (*imds.GetRegionOutput, error) {
	c.UsageCounter++

	return &imds.GetRegionOutput{
		Region: testRegion,
	}, nil
}

func (c *mockClient) ModifyNetworkInterfaceAttribute(ctx context.Context, params *ec2.ModifyNetworkInterfaceAttributeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyNetworkInterfaceAttributeOutput, error) {
	c.UsageCounter++

	return nil, nil
}

func (c *mockClient) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	c.UsageCounter++

	deviceIndexZero := int32(0)

	return &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{
				Instances: []types.Instance{
					{
						InstanceId: &testInstId1,
						NetworkInterfaces: []types.InstanceNetworkInterface{
							{
								NetworkInterfaceId: &testEniId1,
								Attachment: &types.InstanceNetworkInterfaceAttachment{
									DeviceIndex: &deviceIndexZero,
								},
							},
						},
					},
					{
						InstanceId: &testInstId,
						NetworkInterfaces: []types.InstanceNetworkInterface{
							{
								NetworkInterfaceId: &testEniId,
								Attachment: &types.InstanceNetworkInterfaceAttachment{
									DeviceIndex: &deviceIndexZero,
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func (c *mockClient) CreateNetworkInterface(ctx context.Context, params *ec2.CreateNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.CreateNetworkInterfaceOutput, error) {
	panic("implement me")
}

func (c *mockClient) AttachNetworkInterface(ctx context.Context, params *ec2.AttachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.AttachNetworkInterfaceOutput, error) {
	panic("implement me")
}

func (c *mockClient) AssignPrivateIpAddresses(ctx context.Context, params *ec2.AssignPrivateIpAddressesInput, optFns ...func(*ec2.Options)) (*ec2.AssignPrivateIpAddressesOutput, error) {
	panic("implement me")
}

func (c *mockClient) UnassignPrivateIpAddresses(ctx context.Context, params *ec2.UnassignPrivateIpAddressesInput, optFns ...func(*ec2.Options)) (*ec2.UnassignPrivateIpAddressesOutput, error) {
	panic("implement me")
}

func (c *mockClient) DetachNetworkInterface(ctx context.Context, params *ec2.DetachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DetachNetworkInterfaceOutput, error) {
	panic("implement me")
}

func (c *mockClient) DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error) {
	panic("implement me")
}

func (c *mockClient) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	panic("implement me")
}

func (c *mockClient) DescribeInstanceTypes(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	panic("implement me")
}

func (c *mockClient) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	panic("implement me")
}

func (c *mockClient) AssociateAddress(ctx context.Context, params *ec2.AssociateAddressInput, optFns ...func(*ec2.Options)) (*ec2.AssociateAddressOutput, error) {
	panic("implement me")
}

func (c *mockClient) DisassociateAddress(ctx context.Context, params *ec2.DisassociateAddressInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateAddressOutput, error) {
	panic("implement me")
}

func (c *mockClient) DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	panic("implement me")
}

var (
	_ ec2API = (*mockClient)(nil) // Compile-time type assert.
)

type mockSrcDstCheckUpdater struct {
	fakeClock        *clock.FakeClock
	healthAggregator *health.HealthAggregator
	retryCount       int
	totalRetries     int
}

func newMockSrcDstCheckUpdater(healthAgg *health.HealthAggregator, fc *clock.FakeClock) *mockSrcDstCheckUpdater {
	return &mockSrcDstCheckUpdater{
		fakeClock:        fc,
		healthAggregator: healthAgg,
		retryCount:       0,
		totalRetries:     6,
	}
}

func (updater *mockSrcDstCheckUpdater) Update(option string) error {
	// taking jitter into consideration, each fakeClock step should be slightly longer
	fakeClockSteps := []time.Duration{
		40 * time.Second,
		70 * time.Second,
		3 * time.Minute,
		5 * time.Minute,
		9 * time.Minute,
		9 * time.Minute,
		9 * time.Minute,
	}

	Expect(updater.healthAggregator.Summary().Ready).To(BeFalse())

	updater.fakeClock.Step(fakeClockSteps[updater.retryCount])
	updater.retryCount += 1
	if updater.retryCount > updater.totalRetries {
		return nil
	}
	return errors.New("Some AWS EC2 errors")
}

func newAPIError(code, msg string) error {
	err := &smithy.GenericAPIError{
		Code:    code,
		Message: msg,
		Fault:   0,
	}
	return fmt.Errorf("wrapped err: %w", err)
}

var _ = Describe("AWS Tests", func() {
	It("should correctly convert between errors and awserrors", func() {
		fakeCode := "fakeCode"
		fakeMsg := "fakeMsg"

		awsErr := newAPIError(fakeCode, fakeMsg)
		errMsg := convertError(awsErr)
		Expect(errMsg).To(Equal(fmt.Sprintf("%s: %s", fakeCode, fakeMsg)))

		fakeMsg = "fake non-aws error"
		err := fmt.Errorf(fakeMsg)
		errMsg = convertError(err)
		Expect(errMsg).To(Equal(fakeMsg))
	})

	It("should handle retriable server error", func() {
		internalErrCode := "InternalError"
		internalErrMsg := "internal error"

		awsErr := newAPIError(internalErrCode, internalErrMsg)
		Expect(retriable(awsErr)).To(BeTrue())

		fakeCode := "fakeCode"
		fakeMsg := "fakeMsg"

		awsErr = newAPIError(fakeCode, fakeMsg)
		Expect(retriable(awsErr)).To(BeFalse())

		fakeMsg = "non-aws error"
		err := fmt.Errorf(fakeMsg)
		Expect(retriable(err)).To(BeFalse())
	})

	It("should handle EC2Metadata interactions correctly", func() {
		mock := newMockClient()
		Expect(getEC2InstanceID(context.TODO(), mock)).To(Equal(testInstId))
		Expect(getEC2Region(context.TODO(), mock)).To(Equal(testRegion))
		Expect(mock.UsageCounter).To(BeNumerically("==", 2))
	})

	It("should handle EC2 interactions correctly", func() {
		mock := newMockClient()
		client := &EC2Client{
			EC2Svc:     mock,
			InstanceID: testInstId,
		}

		Expect(client.GetMyPrimaryEC2NetworkInterfaceID(context.TODO())).To(Equal(testEniId))
		Expect(client.SetEC2SourceDestinationCheck(context.TODO(), testEniId, false)).NotTo(HaveOccurred())
		Expect(mock.UsageCounter).To(BeNumerically("==", 2))

		By("verifying Availability")
		os.Setenv("AWS_EC2_METADATA_DISABLED", "true")
		_, err := NewEC2Client(context.TODO())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("access disabled to EC2 IMDS via client option"))
	})

	It("should retry EC2 source-destination-check on error and exit the retry loop on success", func() {
		fc := clock.NewFakeClock(time.Now())
		healthAgg := health.NewHealthAggregator()
		mockSrcDstCheckUpdater := newMockSrcDstCheckUpdater(healthAgg, fc)

		WaitForEC2SrcDstCheckUpdate("Disable", healthAgg, mockSrcDstCheckUpdater, fc)
		Expect(mockSrcDstCheckUpdater.retryCount).To(Equal(1 + mockSrcDstCheckUpdater.totalRetries))
		Expect(healthAgg.Summary().Ready).To(BeTrue())
	})
})
