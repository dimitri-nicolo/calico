// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package sethelper_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/set"

	"github.com/tigera/compliance/pkg/sethelper"
)

var _ = Describe("Add sets", func() {
	It("should update the receiving set with the additional entries from the other set", func() {
		By("Creating two sets with different contents")
		s1 := set.From("a", "b", "c", "d")
		s2 := set.From("c", "d", "e", 1)

		By("Addings the sets")
		sethelper.AddSet(s1, s2)

		By("Checking the results")
		Expect(s1).To(Equal(set.From("a", "b", "c", "d")))
		Expect(s2).To(Equal(set.From("a", "b", "c", "d", "e", 1)))
	})
})
