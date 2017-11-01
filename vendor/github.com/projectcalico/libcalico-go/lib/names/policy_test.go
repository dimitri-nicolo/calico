// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package names_test

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/names"
)

var _ = DescribeTable("Parse Tiered policy name",
	func(policy string, expectError bool, expectedTier string) {
		tier, err := names.TierFromPolicyName(policy)
		if expectError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(tier).To(Equal(expectedTier))
		}
	},
	Entry("Empty policy name", "", true, ""),
	Entry("K8s network policy", "knp.default.foopolicy", false, "default"),
	Entry("Policy name without tier", "foopolicy", false, "default"),
	Entry("Correct tiered policy name", "baztier.foopolicy", false, "baztier"),
)
