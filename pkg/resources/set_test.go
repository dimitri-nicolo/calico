// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package resources_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/resources"
)

var (
	r1 = apiv3.ResourceID{
		TypeMeta:  resources.TypeK8sPods,
		Name:      "a",
		Namespace: "b",
	}
	r2 = apiv3.ResourceID{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		Name:     "a",
	}
	r3 = apiv3.ResourceID{
		TypeMeta:  resources.TypeCalicoNetworkPolicies,
		Name:      "a",
		Namespace: "b",
	}
	r4 = apiv3.ResourceID{
		TypeMeta:  resources.TypeK8sNetworkPolicies,
		Name:      "a",
		Namespace: "b",
	}
)

var _ = Describe("Set differences", func() {
	It("should get appropriate callbacks for IterDifferences", func() {
		By("Creating two sets with different contents")
		s1 := resources.SetFrom(r1, r2, r3)
		s2 := resources.SetFrom(r3, r4)

		By("Iterating through differences and storing results")
		s1NotS2 := resources.NewSet()
		s2NotS1 := resources.NewSet()

		s1.IterDifferences(s2,
			func(diff apiv3.ResourceID) error {
				s1NotS2.Add(diff)
				return nil
			},
			func(diff apiv3.ResourceID) error {
				s2NotS1.Add(diff)
				return nil
			},
		)

		By("Checking the results")
		Expect(s1NotS2).To(Equal(resources.SetFrom(r1, r2)))
		Expect(s2NotS1).To(Equal(resources.SetFrom(r4)))
	})
})
