// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	. "github.com/tigera/compliance/internal/testutils"
	"github.com/tigera/compliance/pkg/xrefcache"
)

var _ = Describe("Basic CRUD of network policies with no other resources present", func() {
	var tester *XRefCacheTester

	BeforeEach(func() {
		tester = NewXrefCacheTester()
	})

	// Ensure  the client resource list is in-sync with the resource helper.
	It("should handle basic CRUD of GlobalNetworkPolicy and determine non-xref state", func() {
		By("applying a GlobalNetworkPolicy, ingress no rules")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, []apiv3.Rule{}, nil)

		By("checking the cache settings")
		np := tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(xrefcache.CacheEntryProtectedIngress))

		By("applying a GlobalNetworkPolicy, egress no rules")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, nil, []apiv3.Rule{})

		By("checking the cache settings")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(xrefcache.CacheEntryProtectedEgress))

		By("applying a GlobalNetworkPolicy, ingress, one allow source rule with internet CIDR")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, []apiv3.Rule{CalicoRuleAllowSourceNetsInternet}, nil)

		By("checking the cache settings")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryInternetExposedIngress))

		By("applying a GlobalNetworkPolicy, ingress, one allow destination rule with internet CIDR")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, []apiv3.Rule{CalicoRuleAllowDestinationNetsInternet}, nil)

		By("checking the cache settings - dest CIDR not relevant for ingress rule")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(
			xrefcache.CacheEntryProtectedIngress |
				xrefcache.CacheEntryInternetExposedIngress |
				xrefcache.CacheEntryOtherNamespaceExposedIngress,
		))

		By("applying a GlobalNetworkPolicy, ingress, one allow source rule with private CIDR")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, []apiv3.Rule{CalicoRuleAllowSourceNetsPrivate}, nil)

		By("checking the cache settings")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(xrefcache.CacheEntryProtectedIngress))

		By("applying a GlobalNetworkPolicy, ingress and egress no rules")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, []apiv3.Rule{}, []apiv3.Rule{})

		By("checking the cache settings")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryProtectedEgress))

		By("applying a GlobalNetworkPolicy, egress, one allow destination rule with internet CIDR")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, nil, []apiv3.Rule{CalicoRuleAllowDestinationNetsInternet})

		By("checking the cache settings")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(xrefcache.CacheEntryProtectedEgress | xrefcache.CacheEntryInternetExposedEgress))

		By("applying a GlobalNetworkPolicy, egress, one allow source rule with internet CIDR")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, nil, []apiv3.Rule{CalicoRuleAllowSourceNetsInternet})

		By("checking the cache settings - source CIDR not relevant for egress rule")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(
			xrefcache.CacheEntryProtectedEgress |
				xrefcache.CacheEntryInternetExposedEgress |
				xrefcache.CacheEntryOtherNamespaceExposedEgress,
		))

		By("applying a GlobalNetworkPolicy, egress, one allow destination rule with private CIDR")
		tester.SetGlobalNetworkPolicy(Name1, NoLabels, nil, []apiv3.Rule{CalicoRuleAllowDestinationNetsPrivate})

		By("checking the cache settings")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.Flags).To(Equal(xrefcache.CacheEntryProtectedEgress))

		By("deleting the first network policy")
		tester.DeleteGlobalNetworkPolicy(Name1)

		By("checking the cache settings")
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).To(BeNil())
	})
})
