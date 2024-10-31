// Copyright (c) 2019 Tigera, Inc. SelectAll rights reserved.
package xrefcache_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	. "github.com/projectcalico/calico/compliance/internal/testutils"
	"github.com/projectcalico/calico/compliance/pkg/syncer"
	"github.com/projectcalico/calico/compliance/pkg/xrefcache"
	"github.com/projectcalico/calico/libcalico-go/lib/resources"
)

var _ = Describe("Pods cache verification", func() {
	var tester *XrefCacheTester

	BeforeEach(func() {
		tester = NewXrefCacheTester()
	})

	It("should handle basic CRUD of a pod with no other resources", func() {
		By("sending in-sync")
		tester.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		By("applying a pod")
		tester.SetPod(Name1, Namespace1, NoLabels, IP1, NoServiceAccount, NoPodOptions)

		By("checking the cache settings")
		ep := tester.GetPod(Name1, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())
		Expect(ep.GetFlowLogAggregationName()).To(Equal(ep.GetObjectMeta().GetName()))

		By("applying another pod in a different namespace")
		tester.SetPod(Name1, Namespace2, NoLabels, IP2, NoServiceAccount, NoPodOptions)

		By("checking the cache settings")
		ep = tester.GetPod(Name1, Namespace2)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())
		Expect(ep.GetFlowLogAggregationName()).To(Equal(ep.GetObjectMeta().GetName()))

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

	It("should handle a pod with Envoy enabled", func() {
		By("applying a pod")
		tester.SetPod(Name1, Namespace1, NoLabels, IP1, NoServiceAccount, PodOptEnvoyEnabled)

		By("sending in-sync")
		tester.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		By("checking the cache settings")
		ep := tester.GetPod(Name1, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(Equal(xrefcache.CacheEntryEnvoyEnabled))
		Expect(ep.AppliedPolicies.Len()).To(BeZero())
		Expect(ep.GetFlowLogAggregationName()).To(Equal(ep.GetObjectMeta().GetName()))

		By("deleting the first pod")
		tester.DeletePod(Name1, Namespace1)

		By("checking the cache settings")
		ep = tester.GetPod(Name1, Namespace1)
		Expect(ep).To(BeNil())
	})

	It("should handle a pod with generate name", func() {
		By("sending in-sync")
		tester.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		By("applying a pod")
		tester.SetPod(Name1, Namespace1, NoLabels, IP1, NoServiceAccount, PodOptSetGenerateName)

		By("checking the cache settings")
		ep := tester.GetPod(Name1, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())
		Expect(ep.GetFlowLogAggregationName()).To(Equal("pod-*"))

		By("deleting the first pod")
		tester.DeletePod(Name1, Namespace1)

		By("checking the cache settings")
		ep = tester.GetPod(Name1, Namespace1)
		Expect(ep).To(BeNil())
	})

	It("should handle basic CRUD of a host endpoint", func() {
		By("sending in-sync")
		tester.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		By("applying a host endpoint")
		tester.SetHostEndpoint(Name1, NoLabels, IP1)

		By("checking the cache settings")
		ep := tester.GetHostEndpoint(Name1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())
		// SetHostEndpoint always sets node to node1.
		Expect(ep.GetFlowLogAggregationName()).To(Equal("node1"))

		By("applying a different host endpoint")
		tester.SetHostEndpoint(Name2, NoLabels, IP2)

		By("checking the cache settings")
		ep = tester.GetHostEndpoint(Name2)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Flags).To(BeZero())
		Expect(ep.AppliedPolicies.Len()).To(BeZero())
		// SetHostEndpoint always sets node to node1.
		Expect(ep.GetFlowLogAggregationName()).To(Equal("node1"))

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
		tester.SetGlobalNetworkPolicy(TierDefault, Name1, Select1,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, Select1, NoNamespaceSelector),
			},
			nil,
			&Order1,
		)

		By("applying np2 select2 with an ingress allow select2 rule")
		tester.SetGlobalNetworkPolicy(TierDefault, Name2, Select2,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, Select2, NoNamespaceSelector),
			},
			nil,
			&Order1,
		)

		By("applying np1 select1 with an egress allow select1 rule")
		tester.SetNetworkPolicy(TierDefault, Name1, Namespace1, Select1,
			nil,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Destination, Select1, NoNamespaceSelector),
			},
			&Order1,
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
		gnp1 := tester.GetGlobalNetworkPolicy(TierDefault, Name1)
		Expect(gnp1).NotTo(BeNil())
		Expect(gnp1.SelectedPods.Len()).To(Equal(1))
		gnp2 := tester.GetGlobalNetworkPolicy(TierDefault, Name2)
		Expect(gnp2).NotTo(BeNil())
		Expect(gnp2.SelectedPods.Len()).To(Equal(0))
		np1 := tester.GetNetworkPolicy(TierDefault, Name1, Namespace1)
		Expect(np1).NotTo(BeNil())
		Expect(np1.SelectedPods.Len()).To(Equal(1))

		By("checking cross-ref calculated flags are not yet set")
		Expect(gnp1.Flags).To(BeZero())
		Expect(np1.Flags).To(BeZero())

		By("sending in-sync")
		tester.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		By("checking cross-ref calculated flags")
		Expect(gnp1.Flags).To(Equal(
			xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryInternetExposedIngress |
				xrefcache.CacheEntryOtherNamespaceExposedIngress,
		))
		Expect(np1.Flags).To(Equal(
			xrefcache.CacheEntryProtectedEgress, // no egress internet because NP won't match a global NS
		))

		By("checking the pod settings have inherited the expected policy configuration from gnp1 and np1")
		Expect(pod.Flags).To(Equal(
			xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryInternetExposedIngress |
				xrefcache.CacheEntryOtherNamespaceExposedIngress | xrefcache.CacheEntryProtectedEgress,
		))

		By("updating np1 to include a namespace selector")
		tester.SetNetworkPolicy(TierDefault, Name1, Namespace1, Select1,
			nil,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Destination, Select1, Select1),
			},
			&Order1,
		)

		By("checking the np1 flags have been updated")
		np1 = tester.GetNetworkPolicy(TierDefault, Name1, Namespace1)
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

	It("should track the set of applied policies and overall settings (with namespaced networkset)", func() {
		By("applying np1 select1 with an ingress allow select1 rule")
		tester.SetGlobalNetworkPolicy(TierDefault, Name1, Select1,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, Select1, NoNamespaceSelector),
			},
			nil,
			&Order1,
		)

		By("applying np2 select2 with an ingress allow select2 rule")
		tester.SetGlobalNetworkPolicy(TierDefault, Name2, Select2,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Source, Select2, NoNamespaceSelector),
			},
			nil,
			&Order1,
		)

		By("applying np1 select1 with an egress allow select1 rule")
		tester.SetNetworkPolicy(TierDefault, Name1, Namespace1, Select1,
			nil,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Destination, Select1, NoNamespaceSelector),
			},
			&Order1,
		)

		By("creating ns1 with label1 and internet exposed")
		tester.SetNetworkSet(Name1, Namespace1, Label1, Public)

		By("creating ns2 with label2 and all addresses private")
		tester.SetNetworkSet(Name2, Namespace1, Label2, Private)

		By("creating a pod1 with label 1")
		tester.SetPod(Name1, Namespace1, Label1, IP1, NoServiceAccount, NoPodOptions)

		By("checking pod1 xref with two policies in the cache")
		pod := tester.GetPod(Name1, Namespace1)
		Expect(pod).NotTo(BeNil())
		Expect(pod.AppliedPolicies.Len()).To(Equal(2))
		gnp1 := tester.GetGlobalNetworkPolicy(TierDefault, Name1)
		Expect(gnp1).NotTo(BeNil())
		Expect(gnp1.SelectedPods.Len()).To(Equal(1))
		gnp2 := tester.GetGlobalNetworkPolicy(TierDefault, Name2)
		Expect(gnp2).NotTo(BeNil())
		Expect(gnp2.SelectedPods.Len()).To(Equal(0))
		np1 := tester.GetNetworkPolicy(TierDefault, Name1, Namespace1)
		Expect(np1).NotTo(BeNil())
		Expect(np1.SelectedPods.Len()).To(Equal(1))

		By("checking cross-ref calculated flags are not yet set")
		Expect(gnp1.Flags).To(BeZero())
		Expect(np1.Flags).To(BeZero())

		By("sending in-sync")
		tester.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		Expect(gnp1.Flags).To(Equal(
			xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryInternetExposedIngress |
				xrefcache.CacheEntryOtherNamespaceExposedIngress,
		))

		Expect(np1.Flags).To(Equal(
			xrefcache.CacheEntryProtectedEgress |
				xrefcache.CacheEntryInternetExposedEgress,
		))

		By("checking the pod settings have inherited the expected policy configuration from gnp1 and np1")
		Expect(pod.Flags).To(Equal(
			xrefcache.CacheEntryProtectedIngress | xrefcache.CacheEntryInternetExposedIngress |
				xrefcache.CacheEntryOtherNamespaceExposedIngress | xrefcache.CacheEntryProtectedEgress |
				xrefcache.CacheEntryInternetExposedEgress,
		))

		By("updating np1 to include a namespace selector")
		tester.SetNetworkPolicy(TierDefault, Name1, Namespace1, Select1,
			nil,
			[]apiv3.Rule{
				CalicoRuleSelectors(Allow, Destination, Select1, Select1),
			},
			&Order1,
		)

		By("checking the np1 flags have been updated")
		np1 = tester.GetNetworkPolicy(TierDefault, Name1, Namespace1)
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

	It("should handle tracking matching services", func() {
		By("sending in-sync")
		tester.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		By("applying pod1 IP1")
		tester.SetPod(Name1, Namespace1, NoLabels, IP1, NoServiceAccount, NoPodOptions)

		By("applying pod2 IP2")
		tester.SetPod(Name2, Namespace1, NoLabels, IP2, NoServiceAccount, NoPodOptions)

		By("applying pod3 with no IP")
		pod3 := tester.SetPod(Name2, Namespace2, NoLabels, 0, NoServiceAccount, NoPodOptions)
		pod3Id := resources.GetResourceID(pod3)

		By("applying service1 with IP1 IP2 IP3")
		svcEps1 := tester.SetEndpoints(Name1, Namespace1, IP1|IP2|IP3)
		svcEpsID1 := resources.GetResourceID(svcEps1)
		svc1 := apiv3.ResourceID{
			TypeMeta:  resources.TypeK8sServices,
			Name:      svcEpsID1.Name,
			Namespace: svcEpsID1.Namespace,
		}

		By("applying service2 with IP1 IP3 and pod3Id ref")
		svcEps2 := tester.SetEndpoints(Name2, Namespace1, IP1|IP3, pod3Id)
		svcEpsID2 := resources.GetResourceID(svcEps2)
		svc2 := apiv3.ResourceID{
			TypeMeta:  resources.TypeK8sServices,
			Name:      svcEpsID2.Name,
			Namespace: svcEpsID2.Namespace,
		}

		By("checking that pod1 refs service1 and service2")
		ep := tester.GetPod(Name1, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(2))
		Expect(ep.Services.Contains(svc1)).To(BeTrue())
		Expect(ep.Services.Contains(svc2)).To(BeTrue())

		By("checking that pod2 refs service1")
		ep = tester.GetPod(Name2, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(1))
		Expect(ep.Services.Contains(svc1)).To(BeTrue())

		By("checking that pod3 refs service2")
		ep = tester.GetPod(Name2, Namespace2)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(1))
		Expect(ep.Services.Contains(svc2)).To(BeTrue())

		By("updating service2 with IP2 IP3 and removing pod3")
		tester.SetEndpoints(Name2, Namespace1, IP2|IP3)

		By("checking that pod1 no longer refs service2")
		ep = tester.GetPod(Name1, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(1))
		Expect(ep.Services.Contains(svc1)).To(BeTrue())
		Expect(ep.Services.Contains(svc2)).To(BeFalse())

		By("checking that pod2 refs service1 and service2")
		ep = tester.GetPod(Name2, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(2))
		Expect(ep.Services.Contains(svc1)).To(BeTrue())
		Expect(ep.Services.Contains(svc2)).To(BeTrue())

		By("checking that pod3 no longer refs service2")
		ep = tester.GetPod(Name2, Namespace2)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(0))

		By("deleting and re-adding pod2 and checking services are the same")
		tester.DeletePod(Name2, Namespace1)
		Expect(tester.GetPod(Name2, Namespace1)).To(BeNil())
		tester.SetPod(Name2, Namespace1, NoLabels, IP2, NoServiceAccount, NoPodOptions)
		ep = tester.GetPod(Name2, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(2))
		Expect(ep.Services.Contains(svc1)).To(BeTrue())
		Expect(ep.Services.Contains(svc2)).To(BeTrue())

		By("updating service1 with IP3")
		tester.SetEndpoints(Name1, Namespace1, IP3)

		By("checking that pod2 no longer refs service1")
		ep = tester.GetPod(Name2, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(1))
		Expect(ep.Services.Contains(svc1)).To(BeFalse())
		Expect(ep.Services.Contains(svc2)).To(BeTrue())

		By("updating pod2 with IP3")
		tester.SetPod(Name2, Namespace1, NoLabels, IP3, NoServiceAccount, NoPodOptions)

		By("checking that pod2 no refs service1 and service2")
		ep = tester.GetPod(Name2, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(2))
		Expect(ep.Services.Contains(svc1)).To(BeTrue())
		Expect(ep.Services.Contains(svc2)).To(BeTrue())

		By("deleting service2")
		tester.DeleteEndpoints(Name2, Namespace1)
		Expect(tester.GetEndpoints(Name2, Namespace1)).To(BeNil())

		By("checking that pod2 no refs service2")
		ep = tester.GetPod(Name2, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(Equal(1))
		Expect(ep.Services.Contains(svc1)).To(BeTrue())
		Expect(ep.Services.Contains(svc2)).To(BeFalse())

		By("deleting service1")
		tester.DeleteEndpoints(Name1, Namespace1)
		Expect(tester.GetEndpoints(Name1, Namespace1)).To(BeNil())

		By("checking both pods reference no services")
		ep = tester.GetPod(Name1, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(BeZero())
		ep = tester.GetPod(Name2, Namespace1)
		Expect(ep).NotTo(BeNil())
		Expect(ep.Services.Len()).To(BeZero())
	})
})
