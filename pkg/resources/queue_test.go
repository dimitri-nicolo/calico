// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package resources_test

import (
	"container/heap"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/resources"
)

var (
	tr1 = apiv3.ResourceID{
		TypeMeta: resources.TypeK8sNamespaces,
		Name:     "ns1",
	}
	tr2 = apiv3.ResourceID{
		TypeMeta: resources.TypeK8sNamespaces,
		Name:     "ns2",
	}
	tr3 = apiv3.ResourceID{
		TypeMeta: resources.TypeK8sNamespaces,
		Name:     "ns3",
	}
	tr4 = apiv3.ResourceID{
		TypeMeta: resources.TypeK8sNamespaces,
		Name:     "ns4",
	}
)

var _ = Describe("Resource priority queue", func() {
	It("should empty the queue in the correct order", func() {
		By("Creating a queue and populating with different priority resources")
		q := &resources.PriorityQueue{}
		heap.Init(q)

		heap.Push(q, &resources.QueueItem{
			ResourceID: tr1,
			Priority:   2,
		})
		heap.Push(q, &resources.QueueItem{
			ResourceID: tr2,
			Priority:   1,
		})
		heap.Push(q, &resources.QueueItem{
			ResourceID: tr3,
			Priority:   3,
		})
		heap.Push(q, &resources.QueueItem{
			ResourceID: tr4,
			Priority:   2,
		})

		By("Checking the items are popped in the correct order")
		qi, ok := heap.Pop(q).(*resources.QueueItem)
		Expect(ok).To(BeTrue())
		Expect(qi.ResourceID).To(Equal(tr3))
		qi, ok = heap.Pop(q).(*resources.QueueItem)
		Expect(ok).To(BeTrue())
		ra := qi.ResourceID
		qi, ok = heap.Pop(q).(*resources.QueueItem)
		Expect(ok).To(BeTrue())
		rb := qi.ResourceID
		Expect(ra == tr1 || ra == tr4).To(BeTrue())
		Expect(rb == tr1 || rb == tr4).To(BeTrue())
		Expect(ra).ToNot(Equal(rb))
		qi, ok = heap.Pop(q).(*resources.QueueItem)
		Expect(ok).To(BeTrue())
		Expect(qi.ResourceID).To(Equal(tr2))
	})
})
