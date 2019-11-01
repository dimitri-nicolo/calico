package pip_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/resources"
	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/syncer"
	"github.com/tigera/compliance/pkg/xrefcache"
	"github.com/tigera/es-proxy/pkg/pip"
)

var (
	// Policy resources
	pr1 = &v3.NetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "tier1.np",
		},
		Spec: v3.NetworkPolicySpec{
			Tier:     "tier1",
			Selector: "has('hello1')",
		},
	}
	pr2 = &v3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "tier1.gnp",
		},
		Spec: v3.GlobalNetworkPolicySpec{
			Tier:     "tier1",
			Selector: "has('hello2')",
		},
	}
	pr3 = &networkingv1.NetworkPolicy{
		TypeMeta: resources.TypeK8sNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny-ingress",
			Namespace: "ns1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}

	// Staged policy updates
	supr1 = &v3.StagedNetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "tier1.np",
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionSet,
			Tier:         "tier1",
			Selector:     "has('goodbye1')",
		},
	}
	supr2 = &v3.StagedGlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "tier1.gnp",
		},
		Spec: v3.StagedGlobalNetworkPolicySpec{
			Tier:     "tier1",
			Selector: "has('goodbye2')",
		},
	}
	supr3 = &v3.StagedKubernetesNetworkPolicy{
		TypeMeta: resources.TypeK8sNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny-ingress",
			Namespace: "ns1",
		},
		Spec: v3.StagedKubernetesNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	// Staged policy deletes
	sdpr1 = &v3.StagedNetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "tier1.np",
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionDelete,
			Tier:         "tier1",
		},
	}
	sdpr2 = &v3.StagedGlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "tier1.gnp",
		},
		Spec: v3.StagedGlobalNetworkPolicySpec{
			StagedAction: v3.StagedActionDelete,
			Tier:         "tier1",
		},
	}
	sdpr3 = &v3.StagedKubernetesNetworkPolicy{
		TypeMeta: resources.TypeK8sNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny-ingress",
			Namespace: "ns1",
		},
		Spec: v3.StagedKubernetesNetworkPolicySpec{
			StagedAction: v3.StagedActionDelete,
		},
	}
)

var _ = Describe("Test sending in pip updates to the xrefcache", func() {
	var xc xrefcache.XrefCache

	BeforeEach(func() {
		xc = xrefcache.NewXrefCache(&config.Config{}, func() {})
		xc.OnStatusUpdate(syncer.NewStatusUpdateInSync())

		xc.OnUpdates([]syncer.Update{{
			Type:       syncer.UpdateTypeSet,
			ResourceID: resources.GetResourceID(pr1),
			Resource:   pr1,
		}, {
			Type:       syncer.UpdateTypeSet,
			ResourceID: resources.GetResourceID(pr2),
			Resource:   pr2,
		}, {
			Type:       syncer.UpdateTypeSet,
			ResourceID: resources.GetResourceID(pr3),
			Resource:   pr3,
		}})
	})

	It("Handles setting of staged policy sets", func() {
		By("Sending set updates for staged policy sets")
		modified, err := pip.ApplyPIPPolicyChanges(xc, []pip.ResourceChange{{
			Action:   "update",
			Resource: supr1,
		}, {
			Action:   "update",
			Resource: supr2,
		}, {
			Action:   "update",
			Resource: supr3,
		}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking the enforced policies have been updated")
		ce := xc.Get(resources.GetResourceID(pr1))
		Expect(ce).ToNot(BeNil())
		p1, ok := ce.GetPrimary().(*v3.NetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p1.Spec.Selector).To(Equal("has('goodbye1')"))

		ce = xc.Get(resources.GetResourceID(pr2))
		Expect(ce).ToNot(BeNil())
		p2, ok := ce.GetPrimary().(*v3.GlobalNetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p2.Spec.Selector).To(Equal("has('goodbye2')"))

		ce = xc.Get(resources.GetResourceID(pr3))
		Expect(ce).ToNot(BeNil())
		p3, ok := ce.GetPrimary().(*networkingv1.NetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p3.Spec.PolicyTypes).To(Equal([]networkingv1.PolicyType{
			networkingv1.PolicyTypeEgress,
		}))

		By("Checking the modified set")
		Expect(modified.IsModified(pr1)).To(BeTrue())
		Expect(modified.IsModified(pr2)).To(BeTrue())
		Expect(modified.IsModified(pr3)).To(BeTrue())
	})

	It("Handles deletion of staged policy sets", func() {
		By("Sending delete updates for staged policy sets")
		modified, err := pip.ApplyPIPPolicyChanges(xc, []pip.ResourceChange{{
			Action:   "delete",
			Resource: supr1,
		}, {
			Action:   "delete",
			Resource: supr2,
		}, {
			Action:   "delete",
			Resource: supr3,
		}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking the enforced policies have not been updated")
		ce := xc.Get(resources.GetResourceID(pr1))
		Expect(ce).ToNot(BeNil())
		p1, ok := ce.GetPrimary().(*v3.NetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p1.Spec.Selector).To(Equal("has('hello1')"))

		ce = xc.Get(resources.GetResourceID(pr2))
		Expect(ce).ToNot(BeNil())
		p2, ok := ce.GetPrimary().(*v3.GlobalNetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p2.Spec.Selector).To(Equal("has('hello2')"))

		ce = xc.Get(resources.GetResourceID(pr3))
		Expect(ce).ToNot(BeNil())
		p3, ok := ce.GetPrimary().(*networkingv1.NetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p3.Spec.PolicyTypes).To(Equal([]networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
		}))

		By("Checking the modified set")
		Expect(modified.IsModified(pr1)).To(BeFalse())
		Expect(modified.IsModified(pr2)).To(BeFalse())
		Expect(modified.IsModified(pr3)).To(BeFalse())
	})

	It("Handles deletion of staged policy deletes", func() {
		By("Sending delete updates for staged policy deletes")
		modified, err := pip.ApplyPIPPolicyChanges(xc, []pip.ResourceChange{{
			Action:   "delete",
			Resource: sdpr1,
		}, {
			Action:   "delete",
			Resource: sdpr2,
		}, {
			Action:   "delete",
			Resource: sdpr3,
		}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking the enforced policies have not been updated")
		ce := xc.Get(resources.GetResourceID(pr1))
		Expect(ce).ToNot(BeNil())
		p1, ok := ce.GetPrimary().(*v3.NetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p1.Spec.Selector).To(Equal("has('hello1')"))

		ce = xc.Get(resources.GetResourceID(pr2))
		Expect(ce).ToNot(BeNil())
		p2, ok := ce.GetPrimary().(*v3.GlobalNetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p2.Spec.Selector).To(Equal("has('hello2')"))

		ce = xc.Get(resources.GetResourceID(pr3))
		Expect(ce).ToNot(BeNil())
		p3, ok := ce.GetPrimary().(*networkingv1.NetworkPolicy)
		Expect(ok).To(BeTrue())
		Expect(p3.Spec.PolicyTypes).To(Equal([]networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
		}))

		By("Checking the modified set")
		Expect(modified.IsModified(pr1)).To(BeFalse())
		Expect(modified.IsModified(pr2)).To(BeFalse())
		Expect(modified.IsModified(pr3)).To(BeFalse())
	})

	It("Handles setting of staged policy deletes", func() {
		By("Sending set updates for staged policy deletes")
		modified, err := pip.ApplyPIPPolicyChanges(xc, []pip.ResourceChange{{
			Action:   "update",
			Resource: sdpr1,
		}, {
			Action:   "update",
			Resource: sdpr2,
		}, {
			Action:   "update",
			Resource: sdpr3,
		}})
		Expect(err).ToNot(HaveOccurred())

		By("Checking the enforced policies have been deleted")
		ce := xc.Get(resources.GetResourceID(pr1))
		Expect(ce).To(BeNil())

		ce = xc.Get(resources.GetResourceID(pr2))
		Expect(ce).To(BeNil())

		ce = xc.Get(resources.GetResourceID(pr3))
		Expect(ce).To(BeNil())

		By("Checking the modified set")
		Expect(modified.IsModified(pr1)).To(BeTrue())
		Expect(modified.IsModified(pr2)).To(BeTrue())
		Expect(modified.IsModified(pr3)).To(BeTrue())
	})
})
