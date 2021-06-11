// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package query

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("Events", func(atom Atom, ok bool) {
	actual := atom
	err := IsValidEventsKeysAtom(&actual)
	if ok {
		Expect(err).ShouldNot(HaveOccurred())
	} else {
		Expect(err).Should(HaveOccurred())
	}
},
	Entry("_id", Atom{Key: "_id", Value: "foo"}, true),
	Entry("alert", Atom{Key: "alert", Value: "foo"}, true),
	Entry("type", Atom{Key: "type", Value: "foo"}, true),
)
