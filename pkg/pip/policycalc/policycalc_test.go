package policycalc_test

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var (
	tier1Policy1 = &v3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: v1.ObjectMeta{
			Name: "tier1.policy1",
		},
		Spec: v3.GlobalNetworkPolicySpec{
			Tier:     "tier1",
			Selector: "color == 'red'",
			Types:    []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
			Ingress:  []v3.Rule{{Action: v3.Allow}},
			Egress:   []v3.Rule{{Action: v3.Allow}},
		},
	}

	tier1Policy2 = &v3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: v1.ObjectMeta{
			Name: "tier1.policy1",
		},
		Spec: v3.GlobalNetworkPolicySpec{
			Tier:     "tier1",
			Selector: "color == 'blue'",
			Types:    []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
			Ingress:  []v3.Rule{{Action: v3.Deny}},
			Egress:   []v3.Rule{{Action: v3.Deny}},
		},
	}

	tier2Policy1 = &v3.NetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: v1.ObjectMeta{
			Name:      "tier2.policy1",
			Namespace: "ns1",
		},
		Spec: v3.NetworkPolicySpec{
			Tier:     "tier2",
			Selector: "color == 'purple'",
			Types:    []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
			Ingress:  []v3.Rule{{Action: v3.Deny}},
			Egress:   []v3.Rule{{Action: v3.Pass}},
		},
	}

	// Matches all in namespace ns1, ingress and egress with no rules, will cause end of tier drop if
	// no other policies.
	tier3Policy1 = &v3.NetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: v1.ObjectMeta{
			Name:      "tier3.policy1",
			Namespace: "ns1",
		},
		Spec: v3.NetworkPolicySpec{
			Tier:     "tier3",
			Types:    []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
			Selector: "all()",
		},
	}

	// Matches all in namespace ns1, ingress allow all.
	tier3Policy2 = &v3.NetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: v1.ObjectMeta{
			Name:      "tier3.policy2",
			Namespace: "ns1",
		},
		Spec: v3.NetworkPolicySpec{
			Tier:     "tier3",
			Types:    []v3.PolicyType{v3.PolicyTypeIngress},
			Selector: "all()",
			Ingress:  []v3.Rule{{Action: v3.Allow}},
		},
	}

	// Matches all in namespace ns1, egress allow all.
	tier3Policy3 = &v3.NetworkPolicy{
		TypeMeta: resources.TypeCalicoNetworkPolicies,
		ObjectMeta: v1.ObjectMeta{
			Name:      "tier3.policy3",
			Namespace: "ns1",
		},
		Spec: v3.NetworkPolicySpec{
			Tier:     "tier3",
			Types:    []v3.PolicyType{v3.PolicyTypeEgress},
			Selector: "all()",
			Egress:   []v3.Rule{{Action: v3.Allow}},
		},
	}

	tier3Policy4 = &v3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: v1.ObjectMeta{
			Name: "tier3.policy4",
		},
		Spec: v3.GlobalNetworkPolicySpec{
			Tier:     "tier3",
			Selector: "all()",
			Types:    []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
			Ingress:  []v3.Rule{{Action: v3.Pass}},
			Egress:   []v3.Rule{{Action: v3.Pass}},
		},
	}

	ns1 = &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns1",
			Labels: map[string]string{
				"name": "ns1",
			},
		},
	}

	ns2 = &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns2",
			Labels: map[string]string{
				"name": "ns2",
			},
		},
	}

	cfgDontCalcActionBefore = &policycalc.Config{
		CalculateOriginalAction: false,
	}

	cfgCalcActionBefore = &policycalc.Config{
		CalculateOriginalAction: true,
	}
)

var _ = Describe("Policy calculator tests - tier/policy/rule/profile enumeration", func() {
	It("handles: no policy -> single policy that drops all in namespace ns1", func() {

		By("Having no policy before")
		rdBefore := &policycalc.ResourceData{
			Tiers:      policycalc.Tiers{},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}

		By("Having a single drop all in namespace ns1 policy")
		rdAfter := &policycalc.ResourceData{
			Tiers: policycalc.Tiers{{
				tier3Policy1,
			}},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}
		modified := make(policycalc.ModifiedResources)
		modified.Add(tier3Policy1)

		By("Creating the policy calculators which calculates before and after")
		pc := policycalc.NewPolicyCalculator(cfgCalcActionBefore, rdBefore, rdAfter, modified)

		By("Checking a flow not in namespace ns1 is unaffected")
		f := &policycalc.Flow{
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns2",
				Labels:    map[string]string{},
			},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns2",
				Labels:    map[string]string{},
			},
			Action: policycalc.ActionAllow,
		}

		processed, before, after := pc.Action(f)
		Expect(processed).To(BeFalse())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionAllow))

		By("Checking a flow with source in namespace ns1 is recalculated")
		f = &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{},
			},
			Destination: policycalc.FlowEndpointData{},
			Action:      policycalc.ActionDeny,
		}

		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionDeny))

		By("Checking a flow with destination in namespace ns1 is recalculated")
		f = &policycalc.Flow{
			Reporter: policycalc.ReporterTypeDestination,
			Source:   policycalc.FlowEndpointData{},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{},
			},
			Action: policycalc.ActionDeny,
		}

		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionDeny))
	})

	It("handles: single policy selecting ns1 with no rules -> next policy ingress allows all for ns1", func() {

		By("Having a single drop all in namespace ns1 policy")
		rdBefore := &policycalc.ResourceData{
			Tiers: policycalc.Tiers{{
				tier3Policy1,
			}},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}

		By("Adding an allow all ingress rule after the no-rule policy")
		rdAfter := &policycalc.ResourceData{
			Tiers: policycalc.Tiers{{
				tier3Policy1,
				tier3Policy2,
			}},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}
		modified := make(policycalc.ModifiedResources)
		modified.Add(tier3Policy2)

		By("Creating the policy calculators which calculates before and after")
		pc := policycalc.NewPolicyCalculator(cfgCalcActionBefore, rdBefore, rdAfter, modified)

		By("Checking a flow with source in ns1 is unaffected")
		f := &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{},
			},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns2",
				Labels:    map[string]string{},
			},
			Action: policycalc.ActionAllow,
		}

		processed, before, after := pc.Action(f)
		Expect(processed).To(BeFalse())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Action(f)
		Expect(processed).To(BeFalse())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionAllow))

		By("Checking a flow with dest in namespace ns1 is recalculated")
		f = &policycalc.Flow{
			Reporter: policycalc.ReporterTypeDestination,
			Source:   policycalc.FlowEndpointData{},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{},
			},
			Action: policycalc.ActionAllow,
		}

		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionDeny))
		Expect(after).To(Equal(policycalc.ActionAllow))
	})

	It("handles: single policy selecting ns1 with no rules -> next policy egress allows all for ns1", func() {

		By("Having a single drop all in namespace ns1 policy")
		rdBefore := &policycalc.ResourceData{
			Tiers: policycalc.Tiers{{
				tier3Policy1,
			}},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}

		By("Adding an allow all egress rule after the no-rule policy")
		rdAfter := &policycalc.ResourceData{
			Tiers: policycalc.Tiers{{
				tier3Policy1,
				tier3Policy3,
			}},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}
		modified := make(policycalc.ModifiedResources)
		modified.Add(tier3Policy3)

		By("Creating the policy calculators which calculates before and after")
		pc := policycalc.NewPolicyCalculator(cfgCalcActionBefore, rdBefore, rdAfter, modified)

		By("Checking a flow with dest in ns1 is unaffected")
		f := &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns2",
				Labels:    map[string]string{},
			},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{},
			},
			Action: policycalc.ActionAllow,
		}

		processed, before, after := pc.Action(f)
		Expect(processed).To(BeFalse())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Action(f)
		Expect(processed).To(BeFalse())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionAllow))

		By("Checking a flow with source in namespace ns1 is recalculated")
		f = &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{},
			},
			Destination: policycalc.FlowEndpointData{},
			Action:      policycalc.ActionAllow,
		}

		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionDeny))
		Expect(after).To(Equal(policycalc.ActionAllow))
	})

	It("handles: multiple tiers", func() {

		By("Having no resources before")
		rdBefore := &policycalc.ResourceData{
			Tiers:      policycalc.Tiers{},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}

		By("Adding a bunch of policies across multiple tiers")
		rdAfter := &policycalc.ResourceData{
			Tiers: policycalc.Tiers{
				{tier1Policy1, tier1Policy2},
				{tier2Policy1},
				{tier3Policy4},
			},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}
		modified := make(policycalc.ModifiedResources)
		modified.Add(tier1Policy1)
		modified.Add(tier1Policy2)
		modified.Add(tier2Policy1)
		modified.Add(tier3Policy4)

		By("Creating the policy calculators which calculates after and leaves before action unchanged")
		pc := policycalc.NewPolicyCalculator(cfgDontCalcActionBefore, rdBefore, rdAfter, modified)

		By("Checking a red->red flow is allowed")
		f := &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns2",
				Labels:    map[string]string{"color": "red"},
			},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{"color": "red"},
			},
			Action: policycalc.ActionDeny,
		}

		processed, before, after := pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionDeny))
		Expect(after).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionDeny))
		Expect(after).To(Equal(policycalc.ActionAllow))

		By("Checking a red->blue flow is denied")
		f = &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns2",
				Labels:    map[string]string{"color": "red"},
			},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{"color": "blue"},
			},
			Action: policycalc.ActionAllow,
		}

		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionDeny))

		By("Checking a blue->red flow is denied")
		f = &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns2",
				Labels:    map[string]string{"color": "blue"},
			},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{"color": "red"},
			},
			Action: policycalc.ActionAllow,
		}

		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionDeny))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionAllow))

		By("Checking a net->purple flow is denied")
		f = &policycalc.Flow{
			Reporter: policycalc.ReporterTypeDestination,
			Source:   policycalc.FlowEndpointData{},
			Destination: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Labels:    map[string]string{"color": "purple"},
			},
			Action: policycalc.ActionAllow,
		}

		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionAllow))
		Expect(after).To(Equal(policycalc.ActionDeny))

		By("Checking a purple->net flow is denied")
		f = &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns2",
				Labels:    map[string]string{"color": "purple"},
			},
			Destination: policycalc.FlowEndpointData{},
			Action:      policycalc.ActionDeny,
		}

		processed, before, after = pc.Action(f)
		Expect(processed).To(BeTrue())
		Expect(before).To(Equal(policycalc.ActionDeny))
		Expect(after).To(Equal(policycalc.ActionAllow))
	})
})
