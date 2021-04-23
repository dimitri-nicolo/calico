// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package rbac_test

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	. "github.com/tigera/lma/pkg/rbac"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	core_v1 "k8s.io/api/core/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"

	projectcalico_v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/resources"
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
		Resource: "pods",
	}}
)

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

type mockClient struct {
	roles               map[string][]rbac_v1.PolicyRule
	roleBindings        map[string][]string
	clusterRoles        map[string][]rbac_v1.PolicyRule
	clusterRoleBindings []string
	namespaces          []string
	tiers               []string
}

func (m *mockClient) GetRole(namespace, name string) (*rbac_v1.Role, error) {
	rules := m.roles[namespace+"/"+name]
	if rules == nil {
		log.Debug("GetRole returning error")
		return nil, fmt.Errorf("Role(%s/%s) does not exist", namespace, name)
	}

	log.Debug("GetRole returning no error")
	return &rbac_v1.Role{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: rules,
	}, nil
}

func (m *mockClient) ListRoleBindings(namespace string) ([]*rbac_v1.RoleBinding, error) {
	if m.roleBindings == nil {
		log.Debug("ListRoleBindings returning error")
		return nil, fmt.Errorf("no RoleBindings set")
	}

	names := m.roleBindings[namespace]

	log.Debugf("ListRoleBindings returning %d results", len(names))
	bindings := make([]*rbac_v1.RoleBinding, len(names))
	for i, name := range names {
		kind := "ClusterRole"
		if name[0] == '/' {
			name = name[1:]
			kind = "Role"
		}
		bindings[i] = &rbac_v1.RoleBinding{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      fmt.Sprintf("role-binding-%d", i),
				Namespace: namespace,
			},
			Subjects: []rbac_v1.Subject{{
				Kind: "User",
				Name: "my-user",
			}},
			RoleRef: rbac_v1.RoleRef{
				Kind: kind,
				Name: name,
			},
		}
	}
	return bindings, nil
}

func (m *mockClient) GetClusterRole(name string) (*rbac_v1.ClusterRole, error) {
	rules := m.clusterRoles[name]
	if rules == nil {
		log.Debug("GetClusterRole returning error")
		return nil, fmt.Errorf("ClusterRole(%s) does not exist", name)
	}

	log.Debug("GetClusterRole returning no error")
	return &rbac_v1.ClusterRole{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Rules: rules,
	}, nil
}

func (m *mockClient) ListClusterRoleBindings() ([]*rbac_v1.ClusterRoleBinding, error) {
	if m.clusterRoleBindings == nil {
		return nil, fmt.Errorf("no ClusterRoleBindings set")
	}

	names := m.clusterRoleBindings
	log.Debugf("ListClusterRoleBindings returning %d results", len(names))

	bindings := make([]*rbac_v1.ClusterRoleBinding, len(names))
	for i, name := range names {
		bindings[i] = &rbac_v1.ClusterRoleBinding{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: fmt.Sprintf("clusterrole-binding-%d", i),
			},
			Subjects: []rbac_v1.Subject{{
				Kind: "User",
				Name: "my-user",
			}},
			RoleRef: rbac_v1.RoleRef{
				Kind: "ClusterRole",
				Name: name,
			},
		}
	}
	return bindings, nil
}

func (m *mockClient) ListNamespaces() ([]*core_v1.Namespace, error) {
	if m.namespaces == nil {
		log.Debug("ListNamespaces returning error")
		return nil, fmt.Errorf("no Namespaces set")
	}
	log.Debugf("ListNamespaces returning %d results", len(m.namespaces))
	namespaces := make([]*core_v1.Namespace, len(m.namespaces))
	for i, name := range m.namespaces {
		namespaces[i] = &core_v1.Namespace{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: name,
			},
		}
	}
	return namespaces, nil
}

func (m *mockClient) ListTiers() ([]*projectcalico_v3.Tier, error) {
	if m.tiers == nil {
		log.Debug("ListTiers returning error")
		return nil, fmt.Errorf("no Tiers set")
	}
	log.Debugf("ListTiers returning %d results", len(m.tiers))
	tiers := make([]*projectcalico_v3.Tier, len(m.tiers))
	for i, name := range m.tiers {
		tiers[i] = &projectcalico_v3.Tier{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: name,
			},
		}
	}
	return tiers, nil
}

func expectPresentButEmpty(p Permissions, rts []ResourceType, verbs []Verb) {
	Expect(p).To(HaveLen(len(rts)))
	for _, rt := range rts {
		vs, ok := p[rt]
		Expect(ok).To(BeTrue())
		Expect(vs).To(HaveLen(len(verbs)))
		for _, v := range verbs {
			m, ok := vs[v]
			Expect(ok).To(BeTrue())
			Expect(m).To(BeNil())
		}
	}
}

var _ = Describe("RBAC calculator tests", func() {
	var calc Calculator
	var mock *mockClient
	var myUser user.Info

	// Calculate the distribution of default resources that are namespaced and tiered.
	var numNonNSNonTiered, numNonNSTiered, numNSNonTiered, numNSTiered, numNonTiered, numTiered, numNS, numNonNS int
	for _, tm := range DefaultResources {
		rh := resources.GetResourceHelperByTypeMeta(tm)
		if rh.IsNamespaced() {
			if rh.IsTieredPolicy() {
				numNSTiered++
				numTiered++
			} else {
				numNSNonTiered++
				numNonTiered++
			}
			numNS++
		} else {
			if rh.IsTieredPolicy() {
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
		mock = &mockClient{
			roles:               map[string][]rbac_v1.PolicyRule{},
			roleBindings:        map[string][]string{},
			clusterRoles:        map[string][]rbac_v1.PolicyRule{},
			clusterRoleBindings: []string{},
			namespaces:          []string{"ns1", "ns2", "ns3", "ns4", "ns5"},
			tiers:               []string{"default", "tier1", "tier2", "tier3", "tier4"},
		}
		calc = NewCalculator(mock, mock, mock, mock, mock, mock)
		myUser = &user.DefaultInfo{
			Name:   "my-user",
			UID:    "abcde",
			Groups: []string{},
			Extra:  map[string][]string{},
		}
	})

	It("handles errors in the Namespace enumeration", func() {
		mock.namespaces = nil
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no Namespaces set"))
		expectPresentButEmpty(res, defaultResourceTypes, AllVerbs)
	})

	It("handles errors in the ClusterRoleBinding enumeration", func() {
		mock.clusterRoleBindings = nil
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no ClusterRoleBindings set"))
		expectPresentButEmpty(res, defaultResourceTypes, AllVerbs)
	})

	It("handles errors in the ClusterRole enumeration from ClusterRoleBinding", func() {
		mock.clusterRoleBindings = []string{"test"}
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ClusterRole(test) does not exist"))
		expectPresentButEmpty(res, defaultResourceTypes, AllVerbs)
	})

	It("handles errors in the RoleBinding enumeration", func() {
		mock.roleBindings = nil
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no RoleBindings set"))
		expectPresentButEmpty(res, defaultResourceTypes, AllVerbs)
	})

	It("handles errors in the ClusterRole enumeration from RoleBinding", func() {
		mock.roleBindings = map[string][]string{"ns1": {"test"}}
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ClusterRole(test) does not exist"))
		expectPresentButEmpty(res, defaultResourceTypes, AllVerbs)
	})

	It("handles errors in the Role enumeration from RoleBinding", func() {
		mock.roleBindings = map[string][]string{"ns1": {"/test"}}
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Role(ns1/test) does not exist"))
		expectPresentButEmpty(res, defaultResourceTypes, AllVerbs)
	})

	It("matches cluster scoped wildcard name matches for all resources", func() {
		mock.clusterRoleBindings = []string{"all-resources"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"all-resources": {{Verbs: []string{"update", "create", "list", "get"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
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
					Expect(ms).To(HaveLen(len(mock.tiers)))
					for _, t := range mock.tiers {
						Expect(ms).To(ContainElement(Match{Namespace: "", Tier: t}))
					}
				} else {
					Expect(ms).To(HaveLen(1))
					Expect(ms[0]).To(Equal(Match{Namespace: "", Tier: ""}))
				}
			}
		}
	})

	It("matches cluster scoped wildcard tier matches for all resources with get access to limited tiers", func() {
		gettableTiers := []string{"default", "tier2"}
		mock.clusterRoleBindings = []string{"all-resources", "get-tiers"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"all-resources": {{Verbs: []string{"delete", "patch", "watch"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
			"get-tiers":     {{Verbs: []string{"get"}, Resources: []string{"tiers"}, ResourceNames: gettableTiers, APIGroups: []string{"projectcalico.org"}}},
		}

		// Matches for tiered policy should only contain the gettable tiers. Also tier get should contain separate
		// named entries for each tier.
		// -  Get for each tier (2)
		// -  Watch for each gettable tier (since we cannot List all tiers).
		// -  Delete/Patch for each tiered policy type in each tier  (no watch because no List)
		// -  Delete/Patch for other resource types (including tiers)  (no watch because no List)
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
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
				if r.Resource == "tiers" && (v == VerbGet || v == VerbWatch) {
					// We expect tier get and watch to be expanded by gettable tiers.
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

	It("matches namespace scoped wildcard name matches for all resources, get access all tiers", func() {
		mock.clusterRoleBindings = []string{"get-tiers"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-tiers": {{Verbs: []string{"get"}, Resources: []string{"tiers"}, APIGroups: []string{"projectcalico.org"}}},
		}
		mock.roleBindings = map[string][]string{"ns1": {"/all-resources"}}
		mock.roles = map[string][]rbac_v1.PolicyRule{
			"ns1/all-resources": {{Verbs: []string{"update", "create", "list"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}
		// We should only get results for namespaced resources + get for tiers
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
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

				if r.Resource == "tiers" && v == VerbGet {
					// We expect all tiers to be gettable.
					Expect(ms).To(HaveLen(1))
					Expect(ms[0]).To(Equal(Match{}))
				} else if !expectedVerbs[v] {
					// If not one of the expected verbs then we expect a nil match set.
					Expect(ms).To(BeNil())
				} else if !isNamespaced(r) {
					// If not namespaced then we expect a nil match set.
					Expect(ms).To(BeNil())
				} else if isTieredPolicy(r) {
					// This is a tiered policy - we expect an entry for each tier in namespace ns1
					Expect(ms).To(HaveLen(len(mock.tiers)))
					for _, t := range mock.tiers {
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
		mock.roleBindings = map[string][]string{"ns1": {"/all-resources"}}
		mock.roles = map[string][]rbac_v1.PolicyRule{
			"ns1/all-resources": {{Verbs: []string{"update", "create", "list"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}
		// We should only get results for namespaced non-tiered policies
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
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

	It("matches namespace scoped wildcard name matches for all resources with get access to limited tiers", func() {
		gettableTiers := []string{"tier2", "tier3"}
		mock.clusterRoleBindings = []string{"get-tiers"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-tiers": {{Verbs: []string{"get"}, Resources: []string{"tiers"}, ResourceNames: gettableTiers, APIGroups: []string{"*"}}},
		}
		mock.roleBindings = map[string][]string{"ns1": {"/test"}}
		mock.roles = map[string][]rbac_v1.PolicyRule{
			"ns1/test": {{Verbs: []string{"delete", "patch", "list", "watch"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}

		// Since we do not have get access to all tiers, the wildcard tier match will be expanded. Also the tier
		// resource will be expanded too. So we'd expect:
		// -  Get for each tier (2)
		// -  Delete/Patch/Watch/List for each namespaced tiered policy type in each tier (4 * 2)
		// -  Delete/Patch/Watch/List for other namespaced resource types
		res, err := calc.CalculatePermissions(myUser, defaultResourceTypes, AllVerbs)
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
					// We expect to be able to get the gettable tiers.
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

	It("matches namespace scoped wildcard name for CNP + cluster scoped tier-specific CNP + namespace scoped tier-specific CNP, with get access on all tiers", func() {
		mock.clusterRoleBindings = []string{"get-tiers", "wildcard-create", "tier1-patch"}
		mock.roleBindings = map[string][]string{
			"ns2": {"wildcard-delete", "tier2-create", "tier1-patch", "tier2-delete"},
			"ns3": {"tier2-delete", "tier1-listwatch"},
		}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
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
		res, err := calc.CalculatePermissions(myUser, []ResourceType{{APIGroup: "projectcalico.org", Resource: "networkpolicies"}}, AllVerbs)
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

	It("has fully gettable and watchable tiers, but not listable", func() {
		mock.clusterRoleBindings = []string{"get-watch-tiers"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-watch-tiers": {{
				Verbs:     []string{"get", "watch"},
				Resources: []string{"tiers"},
				APIGroups: []string{"projectcalico.org"},
			}},
		}

		// We should have watch access for tiers, expanded to cover each tier individually.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "tiers"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(BeNil())
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "default"}, {Tier: "tier1"}, {Tier: "tier2"}, {Tier: "tier3"}, {Tier: "tier4"}}))
	})

	It("has fully gettable tiers, but no list and limited watch access to tiers", func() {
		mock.clusterRoleBindings = []string{"get-tiers", "watch-list-tiers1-2"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
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

		// We should have watch access for specific gettable tiers.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "tiers"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(BeNil())
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
	})

	It("has limited get access to tiers and no list access and limited and partially overlapping watch access to tiers", func() {
		mock.clusterRoleBindings = []string{"get-tiers1-2", "watch-list-tiers2-3"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-tiers1-2": {{
				Verbs:         []string{"get"},
				Resources:     []string{"tiers"},
				ResourceNames: []string{"tier1", "tier2"},
				APIGroups:     []string{"projectcalico.org"},
			}},
			"watch-list-tiers2-3": {{
				Verbs:         []string{"watch"},
				Resources:     []string{"tiers"},
				ResourceNames: []string{"tier2", "tier3"},
				APIGroups:     []string{"projectcalico.org"},
			}},
		}

		// Tier2 and 3 are watchable, but only tier 1 and 2 are gettable. The watchable list should only include the
		// overlap (i.e. tier2).
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "tiers"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbGet))
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(BeNil())
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier2"}}))
	})

	It("has watchable networkpolicies in all tiers and listable in tier1 and tier2", func() {
		mock.clusterRoleBindings = []string{"get-watch-np"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
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

		// We should have watch access limited to tier1 and tier2.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
	})

	It("has listable networkpolicies in all tiers and watchable in tier1 and tier2", func() {
		mock.clusterRoleBindings = []string{"get-watch-np"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
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

		// Watch access overlaps fully with list access.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Tier: "default"}, {Tier: "tier1"}, {Tier: "tier2"}, {Tier: "tier3"}, {Tier: "tier4"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
	})

	It("has listable networkpolicies in tier1 and tier2 and watchable in tier2 and tier3", func() {
		mock.clusterRoleBindings = []string{"get-watch-np"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-watch-np": {{
				Verbs:     []string{"get"},
				Resources: []string{"tiers"},
				APIGroups: []string{"projectcalico.org"},
			}, {
				Verbs:         []string{"list"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier1.*", "tier2.*"},
			}, {
				Verbs:         []string{"watch"},
				Resources:     []string{"tier.networkpolicies"},
				ResourceNames: []string{"tier2.*", "tier3.*"},
				APIGroups:     []string{"projectcalico.org"},
			}},
		}

		// Watch access is limited to watch/list overlap (i.e. tier2)
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier2"}}))
	})

	It("has listable/watchable networkpolicies in all tiers, gettable only in tier2 and tier3", func() {
		mock.clusterRoleBindings = []string{"get-watch-np"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
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

		// List/Watch access limited to gettable tiers.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Tier: "tier2"}, {Tier: "tier3"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Tier: "tier2"}, {Tier: "tier3"}}))
	})

	It("has listable networkpolicies in tiers 1,2,3 in ns1; watchable in 2,3,4 in all namespaces; gettable in 1,2,4", func() {
		mock.clusterRoleBindings = []string{"get-watch-np"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-watch-np": {{
				Verbs:         []string{"get"},
				Resources:     []string{"tiers"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier1", "tier2", "tier4"},
			}, {
				Verbs:         []string{"watch"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier2.*", "tier3.*", "tier4.*"},
			}},
		}
		mock.roleBindings = map[string][]string{"ns1": {"/list-np-ns1"}}
		mock.roles = map[string][]rbac_v1.PolicyRule{
			"ns1/list-np-ns1": {{
				Verbs:         []string{"list"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier1.*", "tier2.*", "tier3.*"},
			}},
		}

		// Watch access limited to overlap with listable and gettable.  List access limited to overlap with gettable and
		// overlapping namespaces.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Namespace: "ns1", Tier: "tier1"}, {Namespace: "ns1", Tier: "tier2"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Namespace: "ns1", Tier: "tier2"}}))
	})

	It("has listable networkpolicies in tiers 1,2,3 in all namespaces; watchable in 2,3,4 in ns1; gettable in 1,2,4", func() {
		mock.clusterRoleBindings = []string{"get-list-np"}
		mock.clusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-list-np": {{
				Verbs:         []string{"get"},
				Resources:     []string{"tiers"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier1", "tier2", "tier4"},
			}, {
				Verbs:         []string{"list"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier1.*", "tier2.*", "tier3.*"},
			}},
		}
		mock.roleBindings = map[string][]string{"ns1": {"/watch-np-ns1"}}
		mock.roles = map[string][]rbac_v1.PolicyRule{
			"ns1/watch-np-ns1": {{
				Verbs:         []string{"watch"},
				Resources:     []string{"tier.networkpolicies"},
				APIGroups:     []string{"projectcalico.org"},
				ResourceNames: []string{"tier2.*", "tier3.*", "tier4.*"},
			}},
		}

		// Watch access limited to overlap with listable and gettable.  List access limited to overlap with gettable and
		// overlapping namespaces.
		rt := ResourceType{APIGroup: "projectcalico.org", Resource: "networkpolicies"}
		res, err := calc.CalculatePermissions(myUser, []ResourceType{rt}, AllVerbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveKey(rt))
		nps := res[rt]
		Expect(nps).To(HaveKey(VerbList))
		Expect(nps).To(HaveKey(VerbWatch))
		Expect(nps[VerbList]).To(Equal([]Match{{Tier: "tier1"}, {Tier: "tier2"}}))
		Expect(nps[VerbWatch]).To(Equal([]Match{{Namespace: "ns1", Tier: "tier2"}}))
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
