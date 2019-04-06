// Copyright (c) 2019 Tigera, Inc. SelectAll rights reserved.
package xrefcache_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	. "github.com/tigera/compliance/internal/testutils"
)

// The network policy rule selector pseudo resources are managed internally through the NetworkPolicyRuleSelectorManager.
// This component handles the creation and deletion of the rule selector pseudo resources. To test this code, easiest
// thing is to create a bunch of real policies with rule selectors and validate the management of the rule selector
// pseudo resources and their augmented linkage data.
//
// We can do these tests with the GlobalNetworkPolicy resource only since we test the correct decomposition of the
// different network policy resource types into the v3/v1 components separately.

var _ = Describe("Basic CRUD of network policies rule selector pseudo resource types", func() {
	var tester *XRefCacheTester

	BeforeEach(func() {
		tester = NewXrefCacheTester()
	})

	It("should handle basic CRUD of a single rule selector pseudo resource", func() {
		By("applying a GlobalNetworkPolicy with no rules")
		tester.SetGlobalNetworkPolicy(Name1, SelectAll,
			[]apiv3.Rule{},
			nil,
		)

		By("checking the rule selector cache has no entries")
		ids := tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(0))

		By("applying a GlobalNetworkPolicy, ingress, one allow  source rule with all() selector")
		tester.SetGlobalNetworkPolicy(Name1, SelectAll,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, SelectAll, NoNamespaceSelector),
			},
			nil,
		)

		By("checking the cache settings")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(1))
		Expect(ids).To(ConsistOf([]string{"all()"}))
		entry := tester.GetGNPRuleSelectorCacheEntry(SelectAll, NoNamespaceSelector)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		np := tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(1))

		By("applying a second GlobalNetworkPolicy, ingress, one allow source rule with all() selector")
		tester.SetGlobalNetworkPolicy(Name2, SelectAll,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, SelectAll, NoNamespaceSelector),
			},
			nil,
		)

		By("checking the cache settings")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(1))
		Expect(ids).To(ConsistOf([]string{"all()"}))
		entry = tester.GetGNPRuleSelectorCacheEntry(SelectAll, NoNamespaceSelector)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(2))
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(1))
		np = tester.GetGlobalNetworkPolicy(Name2)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(1))

		By("deleting the first network policy")
		tester.DeleteGlobalNetworkPolicy(Name1)

		By("checking the cache settings")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(1))
		Expect(ids).To(ConsistOf([]string{"all()"}))
		entry = tester.GetGNPRuleSelectorCacheEntry(SelectAll, NoNamespaceSelector)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		np = tester.GetGlobalNetworkPolicy(Name2)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(1))

		By("deleting the second network policy")
		tester.DeleteGlobalNetworkPolicy(Name2)

		By("checking the cache settings")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(0))
	})

	It("should handle basic CRUD of multiple rule selector pseudo resources", func() {
		By("applying a GlobalNetworkPolicy with no rules")
		tester.SetGlobalNetworkPolicy(Name1, SelectAll,
			[]apiv3.Rule{},
			nil,
		)

		By("checking the rule selector cache has no entries")
		ids := tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(0))

		By("applying a GlobalNetworkPolicy, ingress two allow source, one allow dest")
		tester.SetGlobalNetworkPolicy(Name1, SelectAll,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, SelectAll, NoNamespaceSelector),
				CalicoRuleSelectors(Allow, Destination, Select1, Select2),
				CalicoRuleSelectors(Allow, Source, Select2, Select3),
			},
			nil,
		)

		By("checking the cache settings - the dest rule won't count")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(2))
		entry := tester.GetGNPRuleSelectorCacheEntry(SelectAll, NoNamespaceSelector)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		entry = tester.GetGNPRuleSelectorCacheEntry(Select2, Select3)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		np := tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(2))

		By("applying a second GlobalNetworkPolicy, ingress, two allow source, one overlaps with first GNP")
		tester.SetGlobalNetworkPolicy(Name2, SelectAll,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, Select1, Select2),
				CalicoRuleSelectors(Allow, Source, Select2, Select3),
			},
			nil,
		)

		By("checking the cache settings")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(3))
		entry = tester.GetGNPRuleSelectorCacheEntry(SelectAll, NoNamespaceSelector)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		entry = tester.GetGNPRuleSelectorCacheEntry(Select1, Select2)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		entry = tester.GetGNPRuleSelectorCacheEntry(Select2, Select3)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(2))
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(2))
		np = tester.GetGlobalNetworkPolicy(Name2)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(2))

		By("Updating the second GlobalNetworkPolicy, to change overlapping entries (and include unhandle deny)")
		tester.SetGlobalNetworkPolicy(Name2, SelectAll,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, SelectAll, NoNamespaceSelector),
				CalicoRuleSelectors(Allow, Source, Select1, Select2),
				CalicoRuleSelectors(Deny, Source, Select2, Select3),
			},
			nil,
		)

		By("checking the cache settings")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(3))
		entry = tester.GetGNPRuleSelectorCacheEntry(SelectAll, NoNamespaceSelector)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(2))
		entry = tester.GetGNPRuleSelectorCacheEntry(Select1, Select2)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		entry = tester.GetGNPRuleSelectorCacheEntry(Select2, Select3)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(2))
		np = tester.GetGlobalNetworkPolicy(Name2)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(2))

		By("deleting the second network policy")
		tester.DeleteGlobalNetworkPolicy(Name2)

		By("checking the cache settings")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(2))
		entry = tester.GetGNPRuleSelectorCacheEntry(SelectAll, NoNamespaceSelector)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		entry = tester.GetGNPRuleSelectorCacheEntry(Select2, Select3)
		Expect(entry).ToNot(BeNil())
		Expect(entry.Policies.Len()).To(Equal(1))
		np = tester.GetGlobalNetworkPolicy(Name1)
		Expect(np).ToNot(BeNil())
		Expect(np.AllowRuleSelectors).To(HaveLen(2))

		By("deleting the first network policy")
		tester.DeleteGlobalNetworkPolicy(Name1)

		By("checking the cache settings")
		ids = tester.GetCachedRuleSelectors()
		Expect(ids).To(HaveLen(0))
	})
})
