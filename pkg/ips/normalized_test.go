// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package ips_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/tigera/compliance/pkg/ips"
)

var _ = Describe("Normalized IP address tests", func() {
	It("normalize valid addresses", func() {
		By("Removing leading zeros for IPv4")
		Expect(ips.NormalizeIP("001.200.043.016")).To(Equal("1.200.43.16"))

		By("Removing leading zeros for IPv6")
		Expect(ips.NormalizeIP("0:0::1234")).To(Equal("::1234"))

		By("Using consistent case")
		upper, err := ips.NormalizeIP("::123A")
		Expect(err).ToNot(HaveOccurred())
		lower, err := ips.NormalizeIP("::123a")
		Expect(err).ToNot(HaveOccurred())
		Expect(upper).To(Equal(lower))

		By("Converting IPv4 in IPv6 back to IPv4")
		Expect(ips.NormalizeIP("::ffff:0102:0304")).To(Equal("1.2.3.4"))

		By("Converting a set of IP strings")
		s, err := ips.NormalizedIPSet("::ffff:0102:0304", "0:0::1234", "001.200.043.016")
		Expect(err).ToNot(HaveOccurred())
		Expect(s.Equals(set.From("1.200.43.16", "::1234", "1.2.3.4"))).To(BeTrue())
	})

	It("errors for invalid address formats", func() {
		By("Normalizing an invalid IP")
		_, err := ips.NormalizeIP("a.123.2.3")
		Expect(err).To(HaveOccurred())

		By("Converting a set of IP strings with one invalid")
		s, err := ips.NormalizedIPSet("::ffff:0102:0304", "0:0::1234", "001.200.043.016", "a.1.2.3")
		Expect(err).To(HaveOccurred())
		Expect(s.Equals(set.From("1.200.43.16", "::1234", "1.2.3.4"))).To(BeTrue())
	})
})
