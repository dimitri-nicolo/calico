// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package sethelper_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/set"

	"github.com/projectcalico/calico/compliance/pkg/sethelper"
)

var _ = Describe("Set differences", func() {
	It("should get appropriate callbacks for IterDifferences", func() {
		By("Creating two sets with different contents")
		s1 := set.From("a", "b", "c", "d")
		s2 := set.From("c", "d", "e", 1)

		By("Iterating through differences and storing results")
		s1NotS2 := set.New()
		s2NotS1 := set.New()

		sethelper.IterDifferences(s1, s2,
			func(diff interface{}) error {
				s1NotS2.Add(diff)
				return nil
			},
			func(diff interface{}) error {
				s2NotS1.Add(diff)
				return nil
			},
		)

		By("Checking the results")
		Expect(s1NotS2).To(Equal(set.From("a", "b")))
		Expect(s2NotS1).To(Equal(set.From("e", 1)))
	})
})
