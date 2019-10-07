// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package query

import (
	"github.com/alecthomas/participle"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("Atom",
	func(input, expected string, ok bool) {
		atom := &Atom{}
		parser := participle.MustBuild(atom)
		err := parser.ParseString(input, atom)

		if !ok {
			Expect(err).Should(HaveOccurred())
		} else {
			Expect(err).ShouldNot(HaveOccurred())
			actual := atom.String()
			Expect(actual).Should(Equal(expected))
		}
	},
	Entry("a = b", "a = b", "a = b", true),
	Entry("a != b", "a != b", "a != b", true),
	Entry("a > b", "a > b", "a > b", true),
	Entry("a >= b", "a >= b", "a >= b", true),
	Entry("a < b", "a < b", "a < b", true),
	Entry("a <= b", "a <= b", "a <= b", true),

	Entry(`"a" = "b"`, `"a" = "b"`, "a = b", true),
	Entry("a = ", "a = ", "", false),
	Entry("a = b.c", "a = b.c", "", false),
	Entry("a.b = c", "a.b = c", "", false),
	Entry("a.b = b.c", "a.b = b.c", "", false),
	Entry(`"a.b" = "b.c"`, `"a.b" = "b.c"`, `"a.b" = "b.c"`, true),
	Entry(`"a.b""" = "b.c"`, `"a.b"" = "b.c"`, "", false),

	Entry("a = 0", "a = 0", `a = "0"`, true),
	Entry("a = 0.1", "a = 0.1", `a = "0.1"`, true),
)

var _ = DescribeTable("Query", func(input, expected string, ok bool) {
	actual, err := ParseQuery(input)

	if !ok {
		Expect(err).Should(HaveOccurred())
	} else {
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actual.String()).Should(Equal(expected))
	}
},
	Entry(`a = b`,
		`a = b`,
		`a = b`,
		true),
	Entry(`a = b AND b = c`,
		`a = b AND b = c`,
		`a = b AND b = c`,
		true),
	Entry(`a = b OR b = c`,
		`a = b OR b = c`,
		`a = b OR b = c`,
		true),
	Entry(`NOT a = b`,
		`NOT a = b`,
		`NOT a = b`,
		true),
	Entry(`a = b AND NOT b = c`,
		`a = b AND NOT b = c`,
		`a = b AND NOT b = c`,
		true),
	Entry(`a = b AND (NOT b = c)`,
		`a = b AND (NOT b = c)`,
		`a = b AND (NOT b = c)`,
		true),
	Entry(`a = b AND (b = c OR NOT d = e)`,
		`a = b AND (b = c OR NOT d = e)`,
		`a = b AND (b = c OR NOT d = e)`,
		true),
	Entry(`(a = b AND b = c) OR d = e`,
		`(a = b AND b = c) OR d = e`,
		`(a = b AND b = c) OR d = e`,
		true),
	Entry(`(a = b AND b = c) OR "d.e" = "e.f"`,
		`(a = b AND b = c) OR "d.e" = "e.f"`,
		`(a = b AND b = c) OR "d.e" = "e.f"`,
		true),
)
