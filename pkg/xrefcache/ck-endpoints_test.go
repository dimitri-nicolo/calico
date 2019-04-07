// Copyright (c) 2019 Tigera, Inc. SelectAll rights reserved.
package xrefcache_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	. "github.com/tigera/compliance/internal/testutils"
	"github.com/tigera/compliance/pkg/xrefcache"
)

var _ = Describe("Pods cache verification", func() {
	var tester *XRefCacheTester

	BeforeEach(func() {
		tester = NewXrefCacheTester()
	})

	It("should handle basic CRUD of a pod with no other resources", func() {
		By("applying a pod")
		tester.SetPod(Name1, Namespace1, NoLabels, IP1, NoServiceAccount, NoPodOptions)

		By("checking the cache settings")
		ep := tester.GetPod(Name1, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())

		By("applying another pod in a different namespace")
		tester.SetPod(Name1, Namespace2, NoLabels, IP2, NoServiceAccount, NoPodOptions)

		By("checking the cache settings")
		ep = tester.GetPod(Name1, Namespace2)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())

		By("deleting the first pod")
		tester.DeletePod(Name1, Namespace1)

		By("checking the cache settings")
		ep = tester.GetPod(Name1, Namespace1)
		Expect(ep).To(BeNil())

		By("deleting the second pod")
		tester.DeletePod(Name1, Namespace2)

		By("checking the cache settings")
		ep = tester.GetPod(Name1, Namespace2)
		Expect(ep).To(BeNil())
	})

	It("should handle basic CRUD of a host endpoint", func() {
		By("applying a host endpoint")
		tester.SetHostEndpoint(Name1, NoLabels, IP1)

		By("checking the cache settings")
		ep := tester.GetHostEndpoint(Name1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())

		By("applying a different host endpoint")
		tester.SetHostEndpoint(Name2, NoLabels, IP2)

		By("checking the cache settings")
		ep = tester.GetHostEndpoint(Name2)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())

		By("deleting the first host endpoint")
		tester.DeleteHostEndpoint(Name1)

		By("checking the cache settings")
		ep = tester.GetHostEndpoint(Name1)
		Expect(ep).To(BeNil())

		By("deleting the second host endpoint")
		tester.DeleteHostEndpoint(Name2)

		By("checking the cache settings")
		ep = tester.GetHostEndpoint(Name2)
		Expect(ep).To(BeNil())
	})

	It("should track the set of applied policies and overall settings", func() {
		By("applying np1 select1 with an ingress allow select1 rule")
		tester.SetGlobalNetworkPolicy(Name1, Select1,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, Select1, NoNamespaceSelector),
			},
			nil,
		)

		By("applying np2 select2 with an ingress allow select2 rule")
		tester.SetGlobalNetworkPolicy(Name2, Select2,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, Select2, NoNamespaceSelector),
			},
			nil,
		)

		By("applying np1 select1 with an egress allow select1 rule")
		tester.SetNetworkPolicy(Name1, Namespace1, Select1,
			nil,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Destination, Select1, NoNamespaceSelector),
			},
		)

		By("creating ns1 with label3 and internet exposed")
		tester.SetGlobalNetworkSet(Name1, Label1, Public)

		By("creating ns2 with label2 and all addresses private")
		tester.SetGlobalNetworkSet(Name2, Label2, Private)

		By("creating a pod1 with label 1")
		tester.SetPod(Name1, Namespace1, Label1, IP1, NoServiceAccount, NoPodOptions)

		By("checking pod1 xref with two policies in the cache")
		pod := tester.GetPod(Name1, Namespace1)
		Expect(pod).NotTo(BeNil())
		Expect(pod.AppliedPolicies.Len()).To(Equal(2))
		gnp1 := tester.GetGlobalNetworkPolicy(Name1)
		Expect(gnp1).NotTo(BeNil())
		Expect(gnp1.SelectedPods.Len()).To(Equal(1))
		Expect(gnp1.Flags).To(Equal(
			xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryInternetExposedIngress |
				xrefcache.CacheEntryOtherNamespaceExposedIngress,
		))
		gnp2 := tester.GetGlobalNetworkPolicy(Name2)
		Expect(gnp2).NotTo(BeNil())
		Expect(gnp2.SelectedPods.Len()).To(Equal(0))
		np1 := tester.GetNetworkPolicy(Name1, Namespace1)
		Expect(np1).NotTo(BeNil())
		Expect(np1.SelectedPods.Len()).To(Equal(1))
		Expect(np1.Flags).To(Equal(
			xrefcache.CacheEntryProtectedEgress, // no egress internet because NP won't match a global NS
		))

		By("checking the pod settings have inherited the expected policy configuration from gnp1 and np1")
		Expect(pod.Flags).To(Equal(
			xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryInternetExposedIngress |
				xrefcache.CacheEntryOtherNamespaceExposedIngress | xrefcache.CacheEntryProtectedEgress,
		))

		By("updating np1 to include a namespace selector")
		tester.SetNetworkPolicy(Name1, Namespace1, Select1,
			nil,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Destination, Select1, Select1),
			},
		)

		By("checking the np1 flags have been updated")
		np1 = tester.GetNetworkPolicy(Name1, Namespace1)
		Expect(np1).NotTo(BeNil())
		Expect(np1.SelectedPods.Len()).To(Equal(1))
		Expect(np1.Flags).To(Equal(
			xrefcache.CacheEntryProtectedEgress | xrefcache.CacheEntryOtherNamespaceExposedEgress,
		))

		By("checking the pod settings have inherited the expected policy configuration from gnp1 and np1")
		pod = tester.GetPod(Name1, Namespace1)
		Expect(pod).NotTo(BeNil())
		Expect(pod.Flags).To(Equal(
			xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryProtectedEgress |
				xrefcache.CacheEntryInternetExposedIngress |
				xrefcache.CacheEntryOtherNamespaceExposedIngress | xrefcache.CacheEntryOtherNamespaceExposedEgress,
		))
	})
})
