// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package resources_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/compliance/pkg/resources"
)

var (
	r1 = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypePods,
		NameNamespace:    resources.NameNamespace{Name: "a", Namespace: "b"},
	}
	r2 = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeGlobalNetworkPolicies,
		NameNamespace:    resources.NameNamespace{Name: "a"},
	}
	r3 = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeNetworkPolicies,
		NameNamespace:    resources.NameNamespace{Name: "a", Namespace: "b"},
	}
	r4 = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeK8sNetworkPolicies,
		NameNamespace:    resources.NameNamespace{Name: "a", Namespace: "b"},
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
			func(diff resources.ResourceID) error {
				s1NotS2.Add(diff)
				return nil
			},
			func(diff resources.ResourceID) error {
				s2NotS1.Add(diff)
				return nil
			},
		)

		By("Checking the results")
		Expect(s1NotS2).To(Equal(resources.SetFrom(r1, r2)))
		Expect(s2NotS1).To(Equal(resources.SetFrom(r4)))
	})
})
