// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package rbac_test

import (
	"encoding/json"

	. "github.com/projectcalico/apiserver/pkg/rbac"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	rbac_v1 "k8s.io/api/rbac/v1"
	"k8s.io/apiserver/pkg/authentication/user"

	rbacmock "github.com/projectcalico/apiserver/pkg/rbac/mock"
)

var (
	defaultResourceTypes = []ResourceType{{
		APIGroup: "projectcalico.org",
		Resource: "hostendpoints",
	}, {
		APIGroup: "projectcalico.org",
		Resource: "tiers",
	}, {
		APIGroup: "projectcalico.org",
		Resource: "stagedkubernetesnetworkpolicies",
	}, {
		APIGroup: "projectcalico.org",
		Resource: "networkpolicies",
	}, {
		APIGroup: "projectcalico.org",
		Resource: "stagednetworkpolicies",
	}, {
		APIGroup: "projectcalico.org",
		Resource: "globalnetworkpolicies",
	}, {
		APIGroup: "projectcalico.org",
		Resource: "stagedglobalnetworkpolicies",
	}, {
		APIGroup: "projectcalico.org",
		Resource: "networksets",
	}, {
		APIGroup: "projectcalico.org",
		Resource: "globalnetworksets",
	}, {
		APIGroup: "networking.k8s.io",
		Resource: "networkpolicies",
	}, {
		APIGroup: "extensions",
		Resource: "networkpolicies",
	}, {
		APIGroup: "",
		Resource: "namespaces",
	}, {
		APIGroup: "",
		Resource: "pods",
	}}
)

var allResourceVerbs []ResourceVerbs

func init() {
	for _, rt := range defaultResourceTypes {
		allResourceVerbs = append(allResourceVerbs, ResourceVerbs{
			rt, AllVerbs,
		})
	}
}

func isTieredPolicy(rt ResourceType) bool {
	if rt.APIGroup != "projectcalico.org" {
		return false
	}
	switch rt.Resource {
	case "networkpolicies", "stagednetworkpolicies", "globalnetworkpolicies", "stagedglobalnetworkpolicies":
		return true
	}
	return false
}

func isNamespaced(rt ResourceType) bool {
	switch rt.Resource {
	case "networkpolicies", "stagednetworkpolicies", "stagedkubernetesnetworkpolicies", "networksets", "pods":
		return true
	}
	return false
}

func expectPresentButEmpty(p Permissions, rvs []ResourceVerbs) {
	Expect(p).To(HaveLen(len(rvs)))
	for _, rv := range rvs {
		vs, ok := p[rv.ResourceType]
		Expect(ok).To(BeTrue())
		Expect(vs).To(HaveLen(len(rv.Verbs)))
		for _, v := range rv.Verbs {
			m, ok := vs[v]
			Expect(ok).To(BeTrue())
			Expect(m).To(BeNil())
		}
	}
}

var _ = Describe("RBAC calculator tests", func() {
	var calc Calculator
	var mock *rbacmock.MockClient
	var myUser user.Info

	// Calculate the distribution of default resources that are namespaced and tiered.
	var numNonNSNonTiered, numNonNSTiered, numNSNonTiered, numNSTiered, numNonTiered, numTiered, numNS, numNonNS int
	for _, rt := range defaultResourceTypes {
		if isNamespaced(rt) {
			if isTieredPolicy(rt) {
				numNSTiered++
				numTiered++
			} else {
				numNSNonTiered++
				numNonTiered++
			}
			numNS++
		} else {
			if isTieredPolicy(rt) {
				numNonNSTiered++
				numTiered++
			} else {
				numNonNSNonTiered++
				numNonTiered++
			}
			numNonNS++
		}
	}

	BeforeEach(func() {
		mock = &rbacmock.MockClient{
			Roles:               map[string][]rbac_v1.PolicyRule{},
			RoleBindings:        map[string][]string{},
			ClusterRoles:        map[string][]rbac_v1.PolicyRule{},
			ClusterRoleBindings: []string{},
			Namespaces:          []string{"ns1", "ns2", "ns3", "ns4", "ns5"},
			Tiers:               []string{"default", "tier1", "tier2", "tier3", "tier4"},
		}
		calc = NewCalculator(mock, mock, mock, mock, mock, mock, mock, 0)
		myUser = &user.DefaultInfo{
			Name:   "my-user",
			UID:    "abcde",
			Groups: []string{},
			Extra:  map[string][]string{},
		}
	})

	It("handles errors in the Namespace enumeration", func() {
		mock.Namespaces = nil
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no Namespaces set"))
		expectPresentButEmpty(res, allResourceVerbs)
	})

	It("handles errors in the ClusterRoleBinding enumeration", func() {
		mock.ClusterRoleBindings = nil
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no ClusterRoleBindings set"))
		expectPresentButEmpty(res, allResourceVerbs)
	})

	It("handles errors in the ClusterRole enumeration from ClusterRoleBinding", func() {
		mock.ClusterRoleBindings = []string{"test"}
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).NotTo(HaveOccurred())
		expectPresentButEmpty(res, allResourceVerbs)
	})

	It("handles errors in the RoleBinding enumeration", func() {
		mock.RoleBindings = nil
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no RoleBindings set"))
		expectPresentButEmpty(res, allResourceVerbs)
	})

	It("handles errors in the ClusterRole enumeration from RoleBinding", func() {
		mock.RoleBindings = map[string][]string{"ns1": {"test"}}
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).NotTo(HaveOccurred())
		expectPresentButEmpty(res, allResourceVerbs)
	})

	It("handles errors in the Role enumeration from RoleBinding", func() {
		mock.RoleBindings = map[string][]string{"ns1": {"/test"}}
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Role(ns1/test) does not exist"))
		expectPresentButEmpty(res, allResourceVerbs)
	})

	It("matches cluster scoped wildcard name matches for all resources", func() {
		mock.ClusterRoleBindings = []string{"all-resources"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"all-resources": {{Verbs: []string{"update", "create", "list", "get"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(len(defaultResourceTypes)))
		expectedVerbs := map[Verb]bool{VerbUpdate: true, VerbCreate: true, VerbList: true, VerbGet: true}
		Expect(res).To(HaveLen(len(defaultResourceTypes)))
		for _, r := range defaultResourceTypes {
			Expect(res).To(HaveKey(r))
			vs := res[r]
			Expect(vs).To(HaveLen(len(AllVerbs)))
			for _, v := range AllVerbs {
				Expect(vs).To(HaveKey(v))
				ms := vs[v]
				if !expectedVerbs[v] {
					Expect(ms).To(BeNil())
				} else if isTieredPolicy(r) {
					Expect(ms).To(HaveLen(len(mock.Tiers)))
					for _, t := range mock.Tiers {
						Expect(ms).To(ContainElement(Match{Namespace: "", Tier: t}))
					}
				} else if r.Resource == "namespaces" && v == VerbGet {
					Expect(ms).To(HaveLen(5))
					Expect(ms).To(Equal([]Match{{Namespace: "ns1"}, {Namespace: "ns2"}, {Namespace: "ns3"}, {Namespace: "ns4"}, {Namespace: "ns5"}}))
				} else if r.Resource == "tiers" && v == VerbGet {
					Expect(ms).To(HaveLen(5))
					Expect(ms).To(Equal([]Match{{Tier: "default"}, {Tier: "tier1"}, {Tier: "tier2"}, {Tier: "tier3"}, {Tier: "tier4"}}))
				}
			}
		}
	})

	It("matches cluster scoped wildcard tier matches for all resources with get access to limited Tiers", func() {
		gettableTiers := []string{"default", "tier2"}
		mock.ClusterRoleBindings = []string{"all-resources", "get-tiers"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"all-resources": {{Verbs: []string{"delete", "patch"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
			"get-tiers":     {{Verbs: []string{"get"}, Resources: []string{"tiers"}, ResourceNames: gettableTiers, APIGroups: []string{"projectcalico.org"}}},
		}

		// Matches for tiered policy should only contain the gettable Tiers. Also tier get should contain separate
		// named entries for each tier.
		// -  Get for each tier (2)
		// -  Delete/Patch for each tiered policy type in each tier
		// -  Delete/Patch for other resource types (including Tiers)
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(len(defaultResourceTypes)))
		expectedVerbs := map[Verb]bool{VerbDelete: true, VerbPatch: true}

		for _, r := range defaultResourceTypes {
			Expect(res).To(HaveKey(r))
			vs := res[r]
			Expect(vs).To(HaveLen(len(AllVerbs)))
			for _, v := range AllVerbs {
				Expect(vs).To(HaveKey(v))
				ms := vs[v]
				if r.Resource == "tiers" && v == VerbGet {
					// We expect tier get and watch to be expanded by gettable Tiers.
					Expect(ms).To(HaveLen(len(gettableTiers)))
					for _, t := range gettableTiers {
						Expect(ms).To(ContainElement(Match{Namespace: "", Tier: t}))
					}
				} else if !expectedVerbs[v] {
					// Not delete or patch, so expect no results.
					Expect(ms).To(BeNil())
				} else if isTieredPolicy(r) {
					// Tiered policy, expect delete/patch for each tier.
					Expect(ms).To(HaveLen(len(gettableTiers)))
					for _, t := range gettableTiers {
						Expect(ms).To(ContainElement(Match{Namespace: "", Tier: t}))
					}
				} else {
					// Non-tiered policy, expect delete/patch.
					Expect(ms).To(HaveLen(1))
					Expect(ms[0]).To(Equal(Match{Namespace: "", Tier: ""}))
				}
			}
		}
	})

	It("matches wildcard name matches for all resources in namespace ns1, get access all Tiers", func() {
		mock.ClusterRoleBindings = []string{"get-tiers"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-tiers": {{Verbs: []string{"get"}, Resources: []string{"tiers"}, APIGroups: []string{"projectcalico.org"}}},
		}
		mock.RoleBindings = map[string][]string{"ns1": {"/all-resources"}}
		mock.Roles = map[string][]rbac_v1.PolicyRule{
			"ns1/all-resources": {{Verbs: []string{"update", "create", "list"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}
		// We should only get results for namespaced resources + get for Tiers
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(len(defaultResourceTypes)))
		expectedVerbs := map[Verb]bool{VerbUpdate: true, VerbCreate: true, VerbList: true}

		for _, r := range defaultResourceTypes {
			Expect(res).To(HaveKey(r))
			vs := res[r]
			Expect(vs).To(HaveLen(len(AllVerbs)))
			for _, v := range AllVerbs {
				Expect(vs).To(HaveKey(v))
				ms := vs[v]

				if r.Resource == "namespaces" && v == VerbGet {
					// We expect all Tiers to be gettable.
					Expect(ms).To(HaveLen(0))
				} else if r.Resource == "tiers" && v == VerbGet {
					// We expect all Tiers to be gettable.
					Expect(ms).To(HaveLen(5))
					Expect(ms).To(Equal([]Match{{Tier: "default"}, {Tier: "tier1"}, {Tier: "tier2"}, {Tier: "tier3"}, {Tier: "tier4"}}))
				} else if !expectedVerbs[v] {
					// If not one of the expected verbs then we expect a nil match set.
					Expect(ms).To(BeNil())
				} else if !isNamespaced(r) {
					// If not namespaced then we expect a nil match set.
					Expect(ms).To(BeNil())
				} else if isTieredPolicy(r) {
					// This is a tiered policy - we expect an entry for each tier in namespace ns1
					Expect(ms).To(HaveLen(len(mock.Tiers)))
					for _, t := range mock.Tiers {
						Expect(ms).To(ContainElement(Match{Namespace: "ns1", Tier: t}))
					}
				} else {
					// This is not a tiered policy, so we expect a single entry for namespace ns1.
					Expect(ms).To(HaveLen(1))
					Expect(ms[0]).To(Equal(Match{Namespace: "ns1", Tier: ""}))
				}
			}
		}
	})

	It("matches namespace scoped wildcard name matches for all resources, no get access to any tier", func() {
		mock.RoleBindings = map[string][]string{"ns1": {"/all-resources"}}
		mock.Roles = map[string][]rbac_v1.PolicyRule{
			"ns1/all-resources": {{Verbs: []string{"update", "create", "list"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}
		// We should only get results for namespaced non-tiered policies
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(len(defaultResourceTypes)))
		expectedVerbs := map[Verb]bool{VerbUpdate: true, VerbCreate: true, VerbList: true}

		for _, r := range defaultResourceTypes {
			Expect(res).To(HaveKey(r))
			vs := res[r]
			Expect(vs).To(HaveLen(len(AllVerbs)))
			for _, v := range AllVerbs {
				Expect(vs).To(HaveKey(v))
				ms := vs[v]

				if isNamespaced(r) && !isTieredPolicy(r) && expectedVerbs[v] {
					// We expect a single result for namespaced, non-tiered policies for the expected verbs.
					Expect(ms).To(HaveLen(1))
					Expect(ms[0]).To(Equal(Match{Namespace: "ns1", Tier: ""}))
				} else {
					// Otherwise we expect no results.
					Expect(ms).To(BeNil())
				}
			}
		}
	})

	It("matches namespace scoped wildcard name matches for all resources with get access to limited Tiers", func() {
		gettableTiers := []string{"tier2", "tier3"}
		mock.ClusterRoleBindings = []string{"get-tiers"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-tiers": {{Verbs: []string{"get"}, Resources: []string{"tiers"}, ResourceNames: gettableTiers, APIGroups: []string{"*"}}},
		}
		mock.RoleBindings = map[string][]string{"ns1": {"/test"}}
		mock.Roles = map[string][]rbac_v1.PolicyRule{
			"ns1/test": {{Verbs: []string{"delete", "patch", "list", "watch"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}

		// Since we do not have get access to all Tiers, the wildcard tier match will be expanded. Also the tier
		// resource will be expanded too. So we'd expect:
		// -  Get for each tier (2)
		// -  Delete/Patch/Watch/List for each namespaced tiered policy type in each tier (4 * 2)
		// -  Delete/Patch/Watch/List for other namespaced resource types
		res, err := calc.CalculatePermissions(myUser, allResourceVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(len(defaultResourceTypes)))
		expectedVerbs := map[Verb]bool{VerbDelete: true, VerbPatch: true, VerbList: true, VerbWatch: true}

		for _, r := range defaultResourceTypes {
			Expect(res).To(HaveKey(r))
			vs := res[r]
			Expect(vs).To(HaveLen(len(AllVerbs)))
			for _, v := range AllVerbs {
				Expect(vs).To(HaveKey(v))
				ms := vs[v]

				if v == VerbGet && r.Resource == "tiers" {
					// We expect to be able to get the gettable Tiers.
					Expect(ms).To(HaveLen(len(gettableTiers)))
					for _, t := range gettableTiers {
						Expect(ms).To(ContainElement(Match{Tier: t}))
					}
				} else if isNamespaced(r) && expectedVerbs[v] {
					// We expect results for namespaced for the expected verbs.
					if isTieredPolicy(r) {
						// For tiered policy, a result per gettable tier.
						Expect(ms).To(HaveLen(len(gettableTiers)))
						for _, t := range gettableTiers {
							Expect(ms).To(ContainElement(Match{Namespace: "ns1", Tier: t}))
						}
					} else {
						// For non-tiered policy, a single result.
						Expect(ms).To(HaveLen(1))
						Expect(ms[0]).To(Equal(Match{Namespace: "ns1", Tier: ""}))
					}
				} else {
					// Otherwise we expect no results.
					Expect(ms).To(BeNil())
				}
			}
		}
	})

	It("matches namespace scoped wildcard name for CNP + cluster scoped tier-specific CNP + namespace scoped tier-specific CNP, with get access on all Tiers", func() {
		mock.ClusterRoleBindings = []string{"get-tiers", "wildcard-create", "tier1-patch"}
		mock.RoleBindings = map[string][]string{
			"ns2": {"wildcard-delete", "tier2-create", "tier1-patch", "tier2-delete"},
			"ns3": {"tier2-delete", "tier1-listwatch"},
		}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-tiers": {{Verbs: []string{"get"}, Resources: []string{"tiers"}, APIGroups: []string{"projectcalico.org"}}},
			"tier1-patch": {{
				Verbs:         []string{"patch"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier1.*"},
			}},
			"tier1-listwatch": {{
				Verbs:         []string{"watch", "list"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier1.*"},
			}},
			"tier2-create": {{
				Verbs:         []string{"create"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier2.*"},
			}},
			"tier2-delete": {{
				Verbs:         []string{"delete"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier2.*"},
			}},
			"wildcard-delete": {{
				Verbs:     []string{"delete"},
				Resources: []string{"tier.networkpolicies"},
				APIGroups: []string{"projectcalico.org"},
			}},
			"wildcard-create": {{
				Verbs:     []string{"create"},
				Resources: []string{"tier.networkpolicies"},
				APIGroups: []string{"projectcalico.org"},
			}},
		}

		// Request permissions for calico network policies only.
		res, err := calc.CalculatePermissions(myUser, []ResourceVerbs{{ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(1))
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		Expect(res).To(HaveKey(rt))
		m := res[rt]
		Expect(m["get"]).To(BeNil())
		Expect(m["update"]).To(BeNil())
		Expect(m["list"]).To(Equal([]Match{{Namespace: "ns3", Tier: "tier1"}}))
		Expect(m["watch"]).To(Equal([]Match{{Namespace: "ns3", Tier: "tier1"}}))
		Expect(m["create"]).To(ConsistOf([]Match{
			{Namespace: "", Tier: "default"},
			{Namespace: "", Tier: "tier1"},
			{Namespace: "", Tier: "tier2"},
			{Namespace: "", Tier: "tier3"},
			{Namespace: "", Tier: "tier4"},
		}))
		Expect(m["delete"]).To(ConsistOf([]Match{
			{Namespace: "ns2", Tier: "default"},
			{Namespace: "ns2", Tier: "tier1"},
			{Namespace: "ns2", Tier: "tier2"},
			{Namespace: "ns2", Tier: "tier3"},
			{Namespace: "ns2", Tier: "tier4"},
			{Namespace: "ns3", Tier: "tier2"},
		}))
		Expect(m["patch"]).To(Equal([]Match{{Namespace: "", Tier: "tier1"}}))
	})

	It("has fully gettable and watchable Tiers, but not listable", func() {
		mock.ClusterRoleBindings = []string{"get-watch-Tiers"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-watch-Tiers": {{
				Verbs:     []string{"get", "watch"},
				Resources: []string{"tiers"},
				APIGroups: []string{"projectcalico.org"},
			}},
		}

		// We should have watch access at cluster scope
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "tiers"}
		res, err := calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(BeNil())
		Expect(nps[VerbWatch]).To(Equal([]Match{{}}))
	})

	It("has fully gettable Tiers, but no list and limited watch access to Tiers", func() {
		mock.ClusterRoleBindings = []string{"get-tiers", "watch-list-tiers1-2"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-tiers": {{
				Verbs:     []string{"get"},
				Resources: []string{"tiers"},
				APIGroups: []string{"projectcalico.org"},
			}},
			"watch-list-tiers1-2": {{
				Verbs:         []string{"watch"},
				Resources:     []string{"tiers"},
				ResourceNames: []string{"tier1", "tier2"},
				APIGroups:     []string{"projectcalico.org"},
			}},
		}

		// We should have watch access for specific gettable Tiers.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "tiers"}
		res, err := calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(BeNil())
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
	})

	It("has fully gettable and createable namespaces limited watch access to Namespaces", func() {
		mock.ClusterRoleBindings = []string{"get-create-namespaces", "watch-ns1-2"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-create-namespaces": {{
				Verbs:     []string{"get", "create"},
				Resources: []string{"namespaces"},
				APIGroups: []string{""},
			}},
			"watch-ns1-2": {{
				Verbs:         []string{"watch"},
				Resources:     []string{"namespaces"},
				ResourceNames: []string{"ns1", "ns2"},
				APIGroups:     []string{""},
			}},
		}

		// Namespace gets should be expanded and so whould wathc it cluster-wide watch is not authorized.
		rt := ResourceType{APIGroup: "", Resource: "namespaces"}
		res, err := calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbGet))
		Expect(nps).To(HaveKey(VerbCreate))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Namespace: "ns1"}, {Namespace: "ns2"}}))
		Expect(nps[VerbGet]).To(Equal([]Match{{Namespace: "ns1"}, {Namespace: "ns2"}, {Namespace: "ns3"}, {Namespace: "ns4"}, {Namespace: "ns5"}}))
		Expect(nps[VerbCreate]).To(Equal([]Match{{}}))
	})

	It("has watchable networkpolicies in all Tiers and listable in tier1 and tier2", func() {
		mock.ClusterRoleBindings = []string{"get-watch-np"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-watch-np": {{
				Verbs:     []string{"get"},
				Resources: []string{"tiers"},
				APIGroups: []string{"projectcalico.org"},
			}, {
				Verbs:     []string{"watch"},
				Resources: []string{"tier.networkpolicies"},
				APIGroups: []string{"projectcalico.org"},
			}, {
				Verbs:         []string{"list"},
				Resources:     []string{"tier.networkpolicies"},
				ResourceNames: []string{"tier1.*", "tier2.*"},
				APIGroups:     []string{"projectcalico.org"},
			}},
		}

		// We should have watch access for each tier.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "default"}, {Tier: "tier1"}, {Tier: "tier2"}, {Tier: "tier3"}, {Tier: "tier4"}}))
	})

	It("has listable networkpolicies in all Tiers and watchable in tier1 and tier2", func() {
		mock.ClusterRoleBindings = []string{"get-watch-np"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-watch-np": {{
				Verbs:     []string{"get"},
				Resources: []string{"tiers"},
				APIGroups: []string{"projectcalico.org"},
			}, {
				Verbs:     []string{"list"},
				Resources: []string{"tier.networkpolicies"},
				APIGroups: []string{"projectcalico.org"},
			}, {
				Verbs:         []string{"watch"},
				Resources:     []string{"tier.networkpolicies"},
				ResourceNames: []string{"tier1.*", "tier2.*"},
				APIGroups:     []string{"projectcalico.org"},
			}},
		}

		// List access for each tier, watch access limited to two Tiers.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Tier: "default"}, {Tier: "tier1"}, {Tier: "tier2"}, {Tier: "tier3"}, {Tier: "tier4"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
	})

	It("has listable/watchable networkpolicies in all Tiers, gettable only in tier2 and tier3", func() {
		mock.ClusterRoleBindings = []string{"get-watch-np"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-watch-np": {{
				Verbs:         []string{"get"},
				Resources:     []string{"tiers"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier2", "tier3"},
			}, {
				Verbs:     []string{"list"},
				Resources: []string{"tier.networkpolicies"},
				APIGroups: []string{"projectcalico.org"},
			}, {
				Verbs:     []string{"watch"},
				Resources: []string{"tier.networkpolicies"},
				APIGroups: []string{"projectcalico.org"},
			}},
		}

		// List/Watch access limited to gettable Tiers.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Tier: "tier2"}, {Tier: "tier3"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier2"}, {Tier: "tier3"}}))
	})

	It("requeries the cache for an unknown resource type", func() {
		mock.ClusterRoleBindings = []string{"get-fake"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-fake": {{
				Verbs:     []string{"get"},
				Resources: []string{"dummy0", "dummy1", "dummy2"},
				APIGroups: []string{"fake"},
			}},
		}

		// Query resource "dummy0". This should be cached first iteration of the mock client.
		rt := ResourceType{APIGroup: "fake", Resource: "dummy0"}
		res, err := calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps[VerbGet]).To(Equal([]Match{{}}))

		// Query resource "dummy2". This is not in the cache.  A second query will update dummy0 to dummy1, but dummy2
		// will still not be in the cache so will not be permitted.
		rt = ResourceType{APIGroup: "fake", Resource: "dummy2"}
		res, err = calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps = res[rt]
		Expect(nps[VerbGet]).To(BeNil())

		// Query resource "dummy2" again. This is not in the cache, but a second query will update dummy1 to dummy2.
		rt = ResourceType{APIGroup: "fake", Resource: "dummy2"}
		res, err = calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps = res[rt]
		Expect(nps[VerbGet]).To(Equal([]Match{{}}))

		// Query resource "dummy0". This is not in the cache anymore because the mock client has clocked past it.
		rt = ResourceType{APIGroup: "fake", Resource: "dummy0"}
		res, err = calc.CalculatePermissions(myUser, []ResourceVerbs{{rt, AllVerbs}})
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps = res[rt]
		Expect(nps[VerbGet]).To(BeNil())
	})

	It("can marshal and unmarshal a Permissions into json", func() {
		By("marshaling a Permissions struct")
		p := Permissions{
			ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}: map[Verb][]Match{
				VerbGet: {{Tier: "a", Namespace: "b"}},
			},
			ResourceType{Resource: "pods"}: map[Verb][]Match{
				VerbList: {{Namespace: "b"}},
			},
		}

		v, err := json.Marshal(p)
		Expect(err).NotTo(HaveOccurred())

		expected := `{
  "networkpolicies.projectcalico.org": {"get": [{"tier": "a", "namespace": "b"}]},
  "pods": {"list": [{"tier": "", "namespace": "b"}]}
}`
		Expect(v).To(MatchJSON(expected))

		By("Unmarshaling the json and comparing to the original")
		p2 := Permissions{}
		err = json.Unmarshal(v, &p2)
		Expect(err).NotTo(HaveOccurred())
		Expect(p2).To(Equal(p))
	})
})
