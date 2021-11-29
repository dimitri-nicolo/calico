// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package index_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/tigera/lma/pkg/elastic/index"
)

var _ = Describe("Index construction tests", func() {
	It("handles alerts", func() {
		Expect(Alerts().GetIndex("foobar")).To(Equal("tigera_secure_ee_events.foobar*"))
	})

	It("handles dns logs", func() {
		Expect(DnsLogs().GetIndex("foobar")).To(Equal("tigera_secure_ee_dns.foobar.*"))
	})

	It("handles flow logs", func() {
		Expect(FlowLogs().GetIndex("foobar")).To(Equal("tigera_secure_ee_flows.foobar.*"))
	})

	It("handles l7 logs", func() {
		Expect(L7Logs().GetIndex("foobar")).To(Equal("tigera_secure_ee_l7.foobar.*"))
	})
})
