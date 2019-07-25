package policycalc_test

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"

	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

var (
	tier1Policy1 = &v3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
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

	serviceAccountNameSource = "sa-source"
	namedPortSourceName      = "source-port"
	namedPortSourcePort      = uint16(10)
	namedPortProtocol        = "TCP"
	namedPortProtocolNumber  = policycalc.ProtoTCP
	namedPortDestinationName = "destination-port"
	namedPortDestinationPort = uint16(11)

	tier3PolicyMatchCached = &v3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "tier3.policymatchcached",
		},
		Spec: v3.GlobalNetworkPolicySpec{
			Tier:     "tier3",
			Selector: "cached == 'true'",
			Types:    []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
			Ingress: []v3.Rule{{
				Action: v3.Allow,
				Source: v3.EntityRule{
					Selector: "source == 'true'",
					ServiceAccounts: &v3.ServiceAccountMatch{
						Names: []string{serviceAccountNameSource},
					},
					Ports: []numorstring.Port{numorstring.NamedPort(namedPortSourceName)},
				},
				Destination: v3.EntityRule{
					Selector: "destination == 'true'",
					Ports:    []numorstring.Port{numorstring.NamedPort(namedPortDestinationName)},
				},
			}},
			Egress: []v3.Rule{{
				Action: v3.Allow,
				Source: v3.EntityRule{
					Selector: "source == 'true'",
					ServiceAccounts: &v3.ServiceAccountMatch{
						Names: []string{serviceAccountNameSource},
					},
					Ports: []numorstring.Port{numorstring.NamedPort(namedPortSourceName)},
				},
				Destination: v3.EntityRule{
					Selector: "destination == 'true'",
					Ports:    []numorstring.Port{numorstring.NamedPort(namedPortDestinationName)},
				},
			}},
		},
	}
	tier3PolicyMatchCachedDenyAll = &v3.GlobalNetworkPolicy{
		TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
		ObjectMeta: metav1.ObjectMeta{
			Name: "tier3.policymatchcached.denyall",
		},
		Spec: v3.GlobalNetworkPolicySpec{
			Tier:     "tier3",
			Selector: "all()",
			Types:    []v3.PolicyType{v3.PolicyTypeIngress, v3.PolicyTypeEgress},
			Ingress: []v3.Rule{{
				Action: v3.Deny,
			}},
			Egress: []v3.Rule{{
				Action: v3.Deny,
			}},
		},
	}

	ns1 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns1",
			Labels: map[string]string{
				"name": "ns1",
			},
		},
	}

	ns2 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns2",
			Labels: map[string]string{
				"name": "ns2",
			},
		},
	}

	cfgDontCalcActionBefore = &pipcfg.Config{
		CalculateOriginalAction: false,
	}

	cfgCalcActionBefore = &pipcfg.Config{
		CalculateOriginalAction: true,
	}
)

var _ = Describe("Policy calculator tests - tier/policy/rule/profile enumeration", func() {
	var ep *policycalc.EndpointCache

	BeforeEach(func() {
		ep = policycalc.NewEndpointCache()
	})

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
		pc := policycalc.NewPolicyCalculator(cfgCalcActionBefore, ep, rdBefore, rdAfter, modified)

		By("Checking a flow not in namespace ns1 is unaffected")
		f := &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
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

		processed, before, after := pc.Calculate(f)
		Expect(processed).To(BeFalse())
		Expect(before.Source.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))

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

		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Source.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Source.Action).To(Equal(policycalc.ActionDeny))

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

		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Destination.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionDeny))
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
		pc := policycalc.NewPolicyCalculator(cfgCalcActionBefore, ep, rdBefore, rdAfter, modified)

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

		processed, before, after := pc.Calculate(f)
		Expect(processed).To(BeFalse())
		Expect(before.Source.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Calculate(f)
		Expect(before.Destination.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionAllow))

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

		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Destination.Action).To(Equal(policycalc.ActionDeny))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionAllow))
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
		pc := policycalc.NewPolicyCalculator(cfgCalcActionBefore, ep, rdBefore, rdAfter, modified)

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

		processed, before, after := pc.Calculate(f)
		Expect(processed).To(BeFalse())
		Expect(before.Source.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeFalse())
		Expect(before.Destination.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionAllow))

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

		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Source.Action).To(Equal(policycalc.ActionDeny))
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))
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
		pc := policycalc.NewPolicyCalculator(cfgDontCalcActionBefore, ep, rdBefore, rdAfter, modified)

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

		processed, before, after := pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Source.Action).To(Equal(policycalc.ActionDeny))
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Calculate(f)
		Expect(before.Destination.Action).To(Equal(policycalc.ActionDeny))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionAllow))

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

		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Source.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Destination.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionDeny))

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

		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Source.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Source.Action).To(Equal(policycalc.ActionDeny))

		f.Reporter = policycalc.ReporterTypeDestination
		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Destination.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionInvalid))

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

		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Destination.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Destination.Action).To(Equal(policycalc.ActionDeny))

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

		processed, before, after = pc.Calculate(f)
		Expect(processed).To(BeTrue())
		Expect(before.Source.Action).To(Equal(policycalc.ActionDeny))
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))
	})

	It("handles: pod source and destination info filled in from cache", func() {

		By("Having a policy that denies all")
		rdBefore := &policycalc.ResourceData{
			Tiers: policycalc.Tiers{{
				tier3PolicyMatchCachedDenyAll,
			}},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}

		By("Adding a policy that matches on ingress and egress cached data before the deny")
		rdAfter := &policycalc.ResourceData{
			Tiers: policycalc.Tiers{{
				tier3PolicyMatchCached,
				tier3PolicyMatchCachedDenyAll,
			}},
			Namespaces: []*corev1.Namespace{ns1, ns2},
		}
		modified := make(policycalc.ModifiedResources)
		modified.Add(tier3PolicyMatchCached)

		By("Updating the endpoint cache with the source and dest endpoints in")
		ep.OnUpdates([]syncer.Update{
			{
				Type: syncer.UpdateTypeDeleted,
			},
			{
				Type: syncer.UpdateTypeSet,
				Resource: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:         "pod1-abcde",
						Namespace:    "ns1",
						GenerateName: "pod1-",
						Labels: map[string]string{
							"cached":      "true",
							"source":      "true",
							"destination": "false",
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: serviceAccountNameSource,
						Containers: []corev1.Container{{
							Ports: []corev1.ContainerPort{{
								Name:          namedPortSourceName,
								ContainerPort: int32(namedPortSourcePort),
								Protocol:      corev1.Protocol(namedPortProtocol),
							}},
						}},
					},
				},
			},
			{
				Type: syncer.UpdateTypeSet,
				Resource: &v3.HostEndpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name: "hostendpoint",
						Labels: map[string]string{
							"cached":      "true",
							"source":      "false",
							"destination": "true",
						},
					},
					Spec: v3.HostEndpointSpec{
						Ports: []v3.EndpointPort{{
							Name:     namedPortDestinationName,
							Protocol: numorstring.ProtocolFromString(namedPortProtocol),
							Port:     namedPortDestinationPort,
						}},
					},
				},
			},
		})

		By("Creating the policy calculators which only calculates after")
		pc := policycalc.NewPolicyCalculator(cfgDontCalcActionBefore, ep, rdBefore, rdAfter, modified)

		By("Creating a flow with all of the cached data missing and running through the policy calculator")
		f := &policycalc.Flow{
			Reporter: policycalc.ReporterTypeSource,
			Source: policycalc.FlowEndpointData{
				Type:      policycalc.EndpointTypeWep,
				Namespace: "ns1",
				Name:      "pod1-*",
				Port:      &namedPortSourcePort,
			},
			Destination: policycalc.FlowEndpointData{
				Type: policycalc.EndpointTypeHep,
				Name: "hostendpoint",
				Port: &namedPortDestinationPort,
			},
			Proto:  &namedPortProtocolNumber,
			Action: policycalc.ActionDeny,
		}
		processed, before, after := pc.Calculate(f)
		Expect(processed).To(BeTrue())

		By("Checking before flow is unchanged")
		Expect(before.Source.Action).To(Equal(policycalc.ActionDeny))
		Expect(before.Source.Include).To(BeTrue())
		Expect(before.Destination.Include).To(BeFalse())

		By("Checking after flow source is allow and included")
		Expect(after.Source.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Source.Include).To(BeTrue())

		By("Checking after flow destination is also allow and included - this has been added by policycalc")
		Expect(after.Destination.Action).To(Equal(policycalc.ActionAllow))
		Expect(after.Destination.Include).To(BeTrue())
	})
})
