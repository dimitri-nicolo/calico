// Copyright (c) 2015-2019 Tigera, Inc. All rights reserved.
package utils_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/cni-plugin/internal/pkg/utils"
)

var _ = Describe("utils", func() {
	table.DescribeTable("Mesos Labels", func(raw, sanitized string) {
		result := utils.SanitizeMesosLabel(raw)
		Expect(result).To(Equal(sanitized))
	},
		table.Entry("valid", "k", "k"),
		table.Entry("dashes", "-my-val", "my-val"),
		table.Entry("double periods", "$my..val", "my.val"),
		table.Entry("special chars", "m$y.val", "m-y.val"),
		table.Entry("slashes", "//my/val/", "my.val"),
		table.Entry("mix of special chars",
			"some_val-with.lots*of^weird#characters", "some_val-with.lots-of-weird-characters"),
	)
})
var _ = Describe("validate start/endRange of IPAMData", func() {

	subnet := "10.10.0.0/24"

	It("should return expected start/end range for empty IPAMData", func() {
		ipamData := make(map[string]interface{})

		err := utils.UpdateHostLocalIPAMDataForWindows(subnet, ipamData)

		Expect(err).NotTo(HaveOccurred())
		Expect(ipamData["rangeStart"]).To(Equal("10.10.0.3"))
		Expect(ipamData["rangeEnd"]).To(Equal("10.10.0.254"))
	})

	It("should return expected start/end range for invalid Range in IPAMData", func() {
		ipamData := map[string]interface{}{
			"rangeStart": "10.10.1.2",
			"rangeEnd":   "10.10.0.255",
		}

		err := utils.UpdateHostLocalIPAMDataForWindows(subnet, ipamData)

		Expect(err).NotTo(HaveOccurred())
		Expect(ipamData["rangeStart"]).To(Equal("10.10.0.3"))
		Expect(ipamData["rangeEnd"]).To(Equal("10.10.0.254"))
	})

	It("should return same start/end range provided in IPAMData", func() {
		ipamData := map[string]interface{}{
			"rangeStart": "10.10.0.15",
			"rangeEnd":   "10.10.0.50",
		}

		err := utils.UpdateHostLocalIPAMDataForWindows(subnet, ipamData)

		Expect(err).NotTo(HaveOccurred())
		Expect(ipamData["rangeStart"]).To(Equal("10.10.0.15"))
		Expect(ipamData["rangeEnd"]).To(Equal("10.10.0.50"))
	})

	It("should return expected start/end range for empty IPs in IPAMData", func() {
		ipamData := map[string]interface{}{
			"rangeStart": "",
			"rangeEnd":   "",
		}

		err := utils.UpdateHostLocalIPAMDataForWindows(subnet, ipamData)

		Expect(err).NotTo(HaveOccurred())
		Expect(ipamData["rangeStart"]).To(Equal("10.10.0.3"))
		Expect(ipamData["rangeEnd"]).To(Equal("10.10.0.254"))
	})

	It("should return expected start/end range for /23 CIDR", func() {
		subnet = "10.0.0.0/23"
		ipamData := map[string]interface{}{
			"rangeStart": "",
			"rangeEnd":   "",
		}

		err := utils.UpdateHostLocalIPAMDataForWindows(subnet, ipamData)

		Expect(err).NotTo(HaveOccurred())
		Expect(ipamData["rangeStart"]).To(Equal("10.0.0.3"))
		Expect(ipamData["rangeEnd"]).To(Equal("10.0.1.254"))
	})

	It("should fail to validate Invalid Ip in range", func() {
		subnet = "10.10.10.10/24"
		ipamData := map[string]interface{}{
			"rangeStart": "10.10.10.256",
			"rangeEnd":   "0.42.42.42",
		}

		err := utils.UpdateHostLocalIPAMDataForWindows(subnet, ipamData)
		Expect(err).To(HaveOccurred())
	})

	It("should fail to validate Invalid CIDR value", func() {
		subnet = "10.10.10.256/24"
		ipamData := map[string]interface{}{
			"rangeStart": "",
			"rangeEnd":   "",
		}

		err := utils.UpdateHostLocalIPAMDataForWindows(subnet, ipamData)
		Expect(err).To(HaveOccurred())
	})
})
