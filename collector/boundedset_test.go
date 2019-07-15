// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bounded set", func() {
	var (
		bs          *boundedSet
		testMaxSize int
	)
	BeforeEach(func() {
		testMaxSize = 2
		bs = NewBoundedSet(testMaxSize)
	})
	It("should be contain the correct elements", func() {
		By("checking the length when there are no elements")
		Expect(bs.TotalCount()).To(BeZero())

		By("adding items")
		bs.Add(net.ParseIP(localIp1Str))
		bs.Add(net.ParseIP(localIp2Str))
		bs.Add(net.ParseIP(localIp2Str)) // Duplicate should have no effect

		By("checking the length")
		Expect(bs.TotalCount()).Should(Equal(2))

		By("checking the contents")
		Expect(bs.Contains(net.ParseIP(localIp1Str))).To(BeTrue())
		Expect(bs.Contains(net.ParseIP(localIp2Str))).To(BeTrue())
		Expect(bs.Contains(net.ParseIP(remoteIp1Str))).To(BeFalse())

		By("adding an extra element and the total count changes but not the contents")
		bs.Add(net.ParseIP(remoteIp1Str))
		Expect(bs.TotalCount()).Should(Equal(3))
		Expect(bs.Contains(net.ParseIP(localIp1Str))).To(BeTrue())
		Expect(bs.Contains(net.ParseIP(localIp2Str))).To(BeTrue())
		Expect(bs.Contains(net.ParseIP(remoteIp1Str))).To(BeFalse())

		By("converting to a slice and checking the contents of the slice")
		ips := bs.ToIPSlice()
		Expect(ips).To(ConsistOf([]net.IP{net.ParseIP(localIp1Str), net.ParseIP(localIp2Str)}))

		By("copying the set and checking the contents")
		newBs := bs.Copy()
		Expect(newBs.TotalCount()).Should(Equal(2))
		Expect(newBs.Contains(net.ParseIP(localIp1Str))).To(BeTrue())
		Expect(newBs.Contains(net.ParseIP(localIp2Str))).To(BeTrue())
		Expect(newBs.Contains(net.ParseIP(remoteIp1Str))).To(BeFalse())

		By("Updating the copy, the copy is updated, the original set isn't")
		newBs.Add(net.ParseIP(remoteIp1Str))
		newBs.Add(net.ParseIP(remoteIp2Str))
		Expect(newBs.TotalCount()).Should(Equal(4))
		Expect(bs.TotalCount()).Should(Equal(3))

		By("Resetting the set")
		bs.Reset()
		Expect(bs.TotalCount()).Should(Equal(0))
		Expect(bs.Contains(net.ParseIP(localIp1Str))).To(BeFalse())
		Expect(bs.Contains(net.ParseIP(localIp2Str))).To(BeFalse())
	})
	It("should be combine multiple boundedSet", func() {
		By("checking the length when there are no elements")
		Expect(bs.TotalCount()).To(BeZero())

		By("adding items")
		bs.Add(net.ParseIP(localIp1Str))
		bs.Add(net.ParseIP(localIp2Str))
		Expect(bs.TotalCount()).To(Equal(2))
		Expect(bs.Contains(net.ParseIP(localIp1Str))).To(BeTrue())
		Expect(bs.Contains(net.ParseIP(localIp2Str))).To(BeTrue())

		By("creating a second bounded set from a array")
		inputIps := []net.IP{net.ParseIP(remoteIp1Str), net.ParseIP(remoteIp2Str)}
		secondSetMaxSize := 3
		secondSetTotalCount := 5
		moreBs := NewBoundedSetFromSliceWithTotalCount(secondSetMaxSize, inputIps, secondSetTotalCount)
		Expect(moreBs.TotalCount()).To(Equal(secondSetTotalCount))
		Expect(moreBs.Contains(net.ParseIP(remoteIp1Str))).To(BeTrue())
		Expect(moreBs.Contains(net.ParseIP(remoteIp2Str))).To(BeTrue())

		By("combining both sets")
		bs.Combine(moreBs)

		By("Initial set's size increases")
		Expect(bs.TotalCount()).To(Equal(secondSetTotalCount + testMaxSize))
		Expect(bs.Contains(net.ParseIP(localIp1Str))).To(BeTrue())
		Expect(bs.Contains(net.ParseIP(localIp2Str))).To(BeTrue())

		By("Initial set doesn't contain elements greater than maxSize")
		Expect(bs.Contains(net.ParseIP(remoteIp1Str))).To(BeFalse())
		Expect(bs.Contains(net.ParseIP(remoteIp2Str))).To(BeFalse())

		By("The secondary set is unchanged")
		Expect(moreBs.TotalCount()).To(Equal(secondSetTotalCount))
		Expect(moreBs.Contains(net.ParseIP(remoteIp1Str))).To(BeTrue())
		Expect(moreBs.Contains(net.ParseIP(remoteIp2Str))).To(BeTrue())
	})
})
