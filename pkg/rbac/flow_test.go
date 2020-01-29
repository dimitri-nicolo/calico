// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package rbac_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	authzv1 "k8s.io/api/authorization/v1"

	"github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/rbac"
)

type mockAuthorizer struct {
	authorized []authzv1.ResourceAttributes
	calls      []authzv1.ResourceAttributes
}

func (m *mockAuthorizer) Authorize(attr *authzv1.ResourceAttributes) (bool, error) {
	m.calls = append(m.calls, *attr)
	for _, a := range m.authorized {
		if a == *attr {
			return true, nil
		}
	}
	return false, nil
}

var _ = Describe("FlowHelper tests", func() {
	It("caches unauthorized results", func() {

		m := &mockAuthorizer{}
		rh := rbac.NewCachedFlowHelper(m)

		By("checking permissions requiring 4 lookups")
		Expect(rh.CanListHostEndpoints()).To(BeFalse())
		Expect(rh.CanListNetworkSets("ns1")).To(BeFalse())
		Expect(rh.CanListPods("ns1")).To(BeFalse())
		Expect(rh.CanListGlobalNetworkSets()).To(BeFalse())
		Expect(m.calls).To(HaveLen(4))

		By("checking the same permissions with cached results")
		Expect(rh.CanListHostEndpoints()).To(BeFalse())
		Expect(rh.CanListNetworkSets("ns1")).To(BeFalse())
		Expect(rh.CanListPods("ns1")).To(BeFalse())
		Expect(rh.CanListGlobalNetworkSets()).To(BeFalse())
		Expect(m.calls).To(HaveLen(4))
	})

	It("handles global network policies", func() {
		ph, ok := api.PolicyHitFromFlowLogPolicyString("0|tier1|tier1.gnp|allow", 0)
		Expect(ok).To(BeTrue())

		By("Checking with no get access to tier")
		m := &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "list",
				Group:    "projectcalico.org",
				Resource: "tier.globalnetworkpolicies",
			}},
		}
		rh := rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeFalse())
		Expect(m.calls).To(HaveLen(1)) // Fails at first check

		By("Checking with wildcard")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "get",
				Group:    "projectcalico.org",
				Resource: "tiers",
				Name:     "tier1",
			}, {
				Verb:     "list",
				Group:    "projectcalico.org",
				Resource: "tier.globalnetworkpolicies",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(2)) // Succeeds at second check

		By("Checking with specific tier allowed")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "get",
				Group:    "projectcalico.org",
				Resource: "tiers",
				Name:     "tier1",
			}, {
				Verb:     "list",
				Group:    "projectcalico.org",
				Resource: "tier.globalnetworkpolicies",
				Name:     "tier1.*",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(3)) // Succeeds at third check
	})

	It("handles staged global network policies", func() {
		ph, ok := api.PolicyHitFromFlowLogPolicyString("0|tier1|staged:tier1.gnp|allow", 0)
		Expect(ok).To(BeTrue())

		By("Checking with no get access to tier")
		m := &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "list",
				Group:    "projectcalico.org",
				Resource: "tier.stagedglobalnetworkpolicies",
			}},
		}
		rh := rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeFalse())
		Expect(m.calls).To(HaveLen(1)) // Fails at first check

		By("Checking with wildcard")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "get",
				Group:    "projectcalico.org",
				Resource: "tiers",
				Name:     "tier1",
			}, {
				Verb:     "list",
				Group:    "projectcalico.org",
				Resource: "tier.stagedglobalnetworkpolicies",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(2)) // Succeeds at second check

		By("Checking with specific tier allowed")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "get",
				Group:    "projectcalico.org",
				Resource: "tiers",
				Name:     "tier1",
			}, {
				Verb:     "list",
				Group:    "projectcalico.org",
				Resource: "tier.stagedglobalnetworkpolicies",
				Name:     "tier1.*",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(3)) // Succeeds at third check
	})

	It("handles network policies", func() {
		ph, ok := api.PolicyHitFromFlowLogPolicyString("0|tier1|ns1/tier1.np|allow", 0)
		Expect(ok).To(BeTrue())

		By("Checking with no get access to tier")
		m := &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "list",
				Group:    "projectcalico.org",
				Resource: "tier.networkpolicies",
			}},
		}
		rh := rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeFalse())
		Expect(m.calls).To(HaveLen(1)) // Fails at first check

		By("Checking with wildcard")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "get",
				Group:    "projectcalico.org",
				Resource: "tiers",
				Name:     "tier1",
			}, {
				Verb:      "list",
				Group:     "projectcalico.org",
				Resource:  "tier.networkpolicies",
				Namespace: "ns1",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(2)) // Succeeds at second check

		By("Checking with specific tier allowed")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "get",
				Group:    "projectcalico.org",
				Resource: "tiers",
				Name:     "tier1",
			}, {
				Verb:      "list",
				Group:     "projectcalico.org",
				Resource:  "tier.networkpolicies",
				Name:      "tier1.*",
				Namespace: "ns1",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(3)) // Succeeds at third check
	})

	It("handles staged network policies", func() {
		ph, ok := api.PolicyHitFromFlowLogPolicyString("0|tier1|ns1/staged:tier1.np|allow", 0)
		Expect(ok).To(BeTrue())

		By("Checking with no get access to tier")
		m := &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "list",
				Group:    "projectcalico.org",
				Resource: "tier.stagednetworkpolicies",
			}},
		}
		rh := rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeFalse())
		Expect(m.calls).To(HaveLen(1)) // Fails at first check

		By("Checking with wildcard")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "get",
				Group:    "projectcalico.org",
				Resource: "tiers",
				Name:     "tier1",
			}, {
				Verb:      "list",
				Group:     "projectcalico.org",
				Resource:  "tier.stagednetworkpolicies",
				Namespace: "ns1",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(2)) // Succeeds at second check

		By("Checking with specific tier allowed")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:     "get",
				Group:    "projectcalico.org",
				Resource: "tiers",
				Name:     "tier1",
			}, {
				Verb:      "list",
				Group:     "projectcalico.org",
				Resource:  "tier.stagednetworkpolicies",
				Name:      "tier1.*",
				Namespace: "ns1",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(3)) // Succeeds at third check
	})

	It("handles kubernetes network policies", func() {
		ph, ok := api.PolicyHitFromFlowLogPolicyString("0|default|ns1/knp.default.np|allow", 0)
		Expect(ok).To(BeTrue())

		By("Checking with different namespace")
		m := &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:      "list",
				Group:     "networking.k8s.io",
				Resource:  "networkpolicies",
				Namespace: "ns2",
			}},
		}
		rh := rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeFalse())
		Expect(m.calls).To(HaveLen(1))

		By("Checking with same namespace")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:      "list",
				Group:     "networking.k8s.io",
				Resource:  "networkpolicies",
				Namespace: "ns1",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(1))
	})

	It("handles staged kubernetes network policies", func() {
		ph, ok := api.PolicyHitFromFlowLogPolicyString("0|default|ns1/staged:knp.default.np|allow", 0)
		Expect(ok).To(BeTrue())

		By("Checking with different namespace")
		m := &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:      "list",
				Group:     "projectcalico.org",
				Resource:  "stagedkubernetesnetworkpolicies",
				Namespace: "ns2",
			}},
		}
		rh := rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeFalse())
		Expect(m.calls).To(HaveLen(1))

		By("Checking with same namespace")
		m = &mockAuthorizer{
			authorized: []authzv1.ResourceAttributes{{
				Verb:      "list",
				Group:     "projectcalico.org",
				Resource:  "stagedkubernetesnetworkpolicies",
				Namespace: "ns1",
			}},
		}
		rh = rbac.NewCachedFlowHelper(m)
		Expect(rh.CanListPolicy(&ph)).To(BeTrue())
		Expect(m.calls).To(HaveLen(1))
	})
})
