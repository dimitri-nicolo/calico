// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package query

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("WAF",
	func(input Atom, expected string, ok bool) {
		actual := input

		err := IsValidWAFAtom(&actual)
		if ok {
			Expect(err).ShouldNot(HaveOccurred())
			Expect(actual.Value).Should(Equal(expected))
		} else {
			Expect(err).Should(HaveOccurred())
		}
	},
	Entry("owaspId", Atom{Key: "owaspId", Value: "4"}, "4", true),
	Entry("owaspId", Atom{Key: "owaspId", Value: "-1"}, "-1", false),
)
