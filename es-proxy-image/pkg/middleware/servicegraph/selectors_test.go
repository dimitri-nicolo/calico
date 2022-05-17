// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph_test

import (
	v1 "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/projectcalico/calico/es-proxy/pkg/middleware/servicegraph"
)

var _ = Describe("Selector tests", func() {
	It("Can process selectors", func() {
		By("creating a selector with all ORs")
		sel1 := NewGraphSelectorConstructor(v1.OpOr,
			NewGraphSelectorConstructor(v1.OpEqual, "a", "b"),
			NewGraphSelectorConstructor(v1.OpEqual, "b", 2),
			nil,
		)
		Expect(sel1.SelectorString()).ToNot(BeNil())
		Expect(*sel1.SelectorString()).To(Equal("a = \"b\" OR b = 2"))

		By("ORing with another selector with all ORs and a duplicate entry")
		sel2 := NewGraphSelectorConstructor(v1.OpOr,
			NewGraphSelectorConstructor(v1.OpEqual, "a", "b2"),
			NewGraphSelectorConstructor(v1.OpEqual, "b", 2),
			nil,
		)
		sel3 := NewGraphSelectorConstructor(v1.OpOr, sel1, sel2)
		Expect(sel3.SelectorString()).ToNot(BeNil())
		Expect(*sel3.SelectorString()).To(Equal("a = \"b\" OR a = \"b2\" OR b = 2"))

		By("ANDing with another selector")
		sel4 := NewGraphSelectorConstructor(v1.OpAnd,
			sel3,
			sel1,
		)
		Expect(sel4.SelectorString()).ToNot(BeNil())
		Expect(*sel4.SelectorString()).To(Equal("(a = \"b\" OR a = \"b2\" OR b = 2) AND (a = \"b\" OR b = 2)"))

		By("ANDing with another selector")
		sel5 := NewGraphSelectorConstructor(v1.OpAnd,
			NewGraphSelectorConstructor(v1.OpNotEqual, "x", "y"),
			nil,
		)
		sel6 := NewGraphSelectorConstructor(v1.OpAnd,
			sel5,
			sel4,
		)
		Expect(sel6.SelectorString()).ToNot(BeNil())
		Expect(*sel6.SelectorString()).To(Equal("(a = \"b\" OR a = \"b2\" OR b = 2) AND (a = \"b\" OR b = 2) AND x != \"y\""))

		By("Checking the conversion works with no selectors")
		gs := GraphSelectorsConstructor{}.ToGraphSelectors()
		Expect(gs).To(Equal(v1.GraphSelectors{}))

		By("Checking the JSON renders with valid selectors")
		gsel1 := GraphSelectorsConstructor{
			L3Flows: NewGraphSelectorConstructor(v1.OpEqual, "a", "b"),
			L7Flows: NewGraphSelectorConstructor(v1.OpEqual, "c", "d"),
			DNSLogs: NewGraphSelectorConstructor(v1.OpEqual, "e", "f"),
		}.And(GraphSelectorsConstructor{
			L3Flows: NewGraphSelectorConstructor(v1.OpEqual, "a1", "b1"),
			L7Flows: NewGraphSelectorConstructor(v1.OpEqual, "c1", "d1"),
			DNSLogs: NewGraphSelectorConstructor(v1.OpEqual, "e1", "f1"),
		}).Or(GraphSelectorsConstructor{
			L3Flows: NewGraphSelectorConstructor(v1.OpEqual, "a2", "b2"),
			L7Flows: NewGraphSelectorConstructor(v1.OpEqual, "c2", "d2"),
			DNSLogs: NewGraphSelectorConstructor(v1.OpEqual, "e.2", "f.2"),
		})

		gs = gsel1.ToGraphSelectors()
		ptr := func(s string) *string {
			return &s
		}
		Expect(gs).To(Equal(v1.GraphSelectors{
			L3Flows: ptr("(a = \"b\" AND a1 = \"b1\") OR a2 = \"b2\""),
			L7Flows: ptr("(c = \"d\" AND c1 = \"d1\") OR c2 = \"d2\""),
			DNSLogs: ptr("\"e.2\" = \"f.2\" OR (e = \"f\" AND e1 = \"f1\")"),
			Alerts:  nil,
		}))

		By("Checking in-operator")
		selIn := NewGraphSelectorConstructor(v1.OpIn,
			"a",
			[]string{"b", "c", "d"},
		)
		//Expect(selIn.SelectorString()).To(Equal("a IN (\"b\", \"c\", \"d\")"))
		Expect(selIn.SelectorString()).ToNot(BeNil())
		Expect(*selIn.SelectorString()).To(Equal("a = \"b\" OR a = \"c\" OR a = \"d\""))

		selIn = NewGraphSelectorConstructor(v1.OpIn,
			"a",
			[]string{},
		)
		Expect(selIn.SelectorString()).To(BeNil())
	})
})
