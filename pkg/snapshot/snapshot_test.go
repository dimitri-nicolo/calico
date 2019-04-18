// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package snapshot

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/list/mock"
	"github.com/tigera/compliance/pkg/resources"
)

var _ = Describe("Snapshot", func() {
	var (
		src          *mock.Source
		dest         *mock.Destination
		testTypeMeta = resources.TypeCalicoHostEndpoints
	)

	BeforeEach(func() {
		retrySleepTime = time.Millisecond
		src = mock.NewSource()
		dest = mock.NewDestination(nil)
	})

	It("should decide that it is not yet time to make a snapshot", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()

		dest.LoadList(&list.TimestampedResourceList{
			ResourceList:              &apiv3.HostEndpointList{TypeMeta: testTypeMeta},
			RequestCompletedTimestamp: metav1.Time{time.Now().Add(-time.Hour)},
		})

		Expect(Run(ctx, testTypeMeta, src, dest)).ToNot(HaveOccurred())
		Expect(dest.RetrieveCalls).To(Equal(1))
		Expect(src.RetrieveCalls).To(Equal(0))
		Expect(dest.StoreCalls).To(Equal(0))
	})

	It("should decide that it is time to make a snapshot but fail because src is empty", func() {
		Expect(Run(context.Background(), testTypeMeta, src, dest)).To(HaveOccurred())
		Expect(dest.RetrieveCalls).To(Equal(1))
		Expect(src.RetrieveCalls).To(Equal(1))
		Expect(dest.StoreCalls).To(Equal(0))
	})

	It("should decide that it is time to make a snapshot and successfully store list from src to dest", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()

		src.LoadList(&list.TimestampedResourceList{
			ResourceList:              &apiv3.HostEndpointList{TypeMeta: testTypeMeta},
			RequestCompletedTimestamp: metav1.Time{time.Now().Add(-25 * time.Hour)},
		})

		Expect(Run(ctx, testTypeMeta, src, dest)).ToNot(HaveOccurred())
		Expect(dest.RetrieveCalls).To(Equal(1))
		Expect(src.RetrieveCalls).To(Equal(1))
		Expect(dest.StoreCalls).To(Equal(1))

		retrievedList, _ := dest.RetrieveList(testTypeMeta, nil, nil, true)
		Expect(retrievedList.ResourceList.GetObjectKind().GroupVersionKind()).To(Equal(testTypeMeta.GroupVersionKind()))
	})
})
