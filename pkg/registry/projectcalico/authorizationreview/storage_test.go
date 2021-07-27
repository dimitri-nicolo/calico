// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package authorizationreview_test

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	rbac_v1 "k8s.io/api/rbac/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/apiserver/pkg/apis/projectcalico"
	"github.com/projectcalico/apiserver/pkg/rbac"
	rbacmock "github.com/projectcalico/apiserver/pkg/rbac/mock"
	. "github.com/projectcalico/apiserver/pkg/registry/projectcalico/authorizationreview"
)

var _ = Describe("RBAC calculator tests", func() {
	var calc rbac.Calculator
	var mock *rbacmock.MockClient
	var myUser user.Info
	var myContext context.Context
	var rest *REST

	BeforeEach(func() {
		mock = &rbacmock.MockClient{
			Roles:               map[string][]rbac_v1.PolicyRule{},
			RoleBindings:        map[string][]string{},
			ClusterRoles:        map[string][]rbac_v1.PolicyRule{},
			ClusterRoleBindings: []string{},
			Namespaces:          []string{"ns1", "ns2", "ns3", "ns4", "ns5"},
			Tiers:               []string{"default", "tier1", "tier2", "tier3", "tier4"},
		}
		calc = rbac.NewCalculator(mock, mock, mock, mock, mock, mock, mock, 0)
		myUser = &user.DefaultInfo{
			Name:   "my-user",
			UID:    "abcde",
			Groups: []string{},
			Extra:  map[string][]string{},
		}
		myContext = request.WithUser(context.Background(), myUser)
		rest = NewREST(calc)
	})

	It("handles errors in the Namespace enumeration", func() {
		// Set namespaces to nil to force an error in the mock client.
		mock.Namespaces = nil

		res, err := rest.Create(myContext, &projectcalico.AuthorizationReview{
			Spec: v3.AuthorizationReviewSpec{
				ResourceAttributes: []v3.AuthorizationReviewResourceAttributes{
					{
						APIGroup:  "",
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
				},
			},
		}, nil, nil)
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeNil())
	})

	It("handles namespace get auth evaluation with no permissions", func() {
		res, err := rest.Create(myContext, &projectcalico.AuthorizationReview{
			Spec: v3.AuthorizationReviewSpec{
				ResourceAttributes: []v3.AuthorizationReviewResourceAttributes{
					{
						APIGroup:  "",
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
				},
			},
		}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		ar := res.(*projectcalico.AuthorizationReview)
		Expect(ar.Status.AuthorizedResourceVerbs).To(Equal([]v3.AuthorizedResourceVerbs{
			{
				Resource: "namespaces",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "get",
					},
				},
			},
		}))
	})

	It("handles namespace get auth evaluation", func() {
		mock.ClusterRoleBindings = []string{"get-namespaces"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"get-namespaces": {{Verbs: []string{"get"}, Resources: []string{"namespaces"}, APIGroups: []string{""}}},
		}

		res, err := rest.Create(myContext, &projectcalico.AuthorizationReview{
			Spec: v3.AuthorizationReviewSpec{
				ResourceAttributes: []v3.AuthorizationReviewResourceAttributes{
					{
						APIGroup:  "",
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
				},
			},
		}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		ar := res.(*projectcalico.AuthorizationReview)
		// get for namespace is expanded across configured namespaces.
		Expect(ar.Status.AuthorizedResourceVerbs).To(Equal([]v3.AuthorizedResourceVerbs{
			{
				Resource: "namespaces",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "get",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Namespace: "ns1"}, {Namespace: "ns2"}, {Namespace: "ns3"}, {Namespace: "ns4"}, {Namespace: "ns5"},
						},
					},
				},
			},
		}))
	})

	It("handles namespace patch auth evaluation", func() {
		mock.ClusterRoleBindings = []string{"patch-namespaces"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"patch-namespaces": {{Verbs: []string{"patch"}, Resources: []string{"namespaces"}, APIGroups: []string{""}}},
		}

		res, err := rest.Create(myContext, &projectcalico.AuthorizationReview{
			Spec: v3.AuthorizationReviewSpec{
				ResourceAttributes: []v3.AuthorizationReviewResourceAttributes{
					{
						APIGroup:  "",
						Resources: []string{"namespaces"},
						Verbs:     []string{"patch"},
					},
				},
			},
		}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		ar := res.(*projectcalico.AuthorizationReview)
		// Verbs other than get for namespace use cluster scoped if appropriate and will not expand across namespaces.
		Expect(ar.Status.AuthorizedResourceVerbs).To(Equal([]v3.AuthorizedResourceVerbs{
			{
				Resource: "namespaces",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "patch",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Namespace: ""},
						},
					},
				},
			},
		}))
	})

	It("has entries for each requested verb/resource combination", func() {
		mock.ClusterRoleBindings = []string{"allow-all"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"allow-all": {{Verbs: []string{"*"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}

		res, err := rest.Create(myContext, &projectcalico.AuthorizationReview{
			Spec: v3.AuthorizationReviewSpec{
				ResourceAttributes: []v3.AuthorizationReviewResourceAttributes{
					{
						APIGroup:  "",
						Resources: []string{"namespaces", "pods"},
						Verbs:     []string{"create", "delete"},
					},
					{
						APIGroup:  "projectcalico.org",
						Resources: []string{"networkpolicies"},
						// Try some duplicates to make sure they are contracted.
						Verbs: []string{"patch", "create", "delete", "patch", "delete"},
					},
				},
			},
		}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		ar := res.(*projectcalico.AuthorizationReview)
		// Verbs other than get for namespace use cluster scoped if appropriate and will not expand across namespaces.
		Expect(ar.Status.AuthorizedResourceVerbs).To(HaveLen(3))
		Expect(ar.Status.AuthorizedResourceVerbs).To(Equal([]v3.AuthorizedResourceVerbs{
			{
				APIGroup: "",
				Resource: "namespaces",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "create",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "", Namespace: ""},
						},
					},
					{
						Verb: "delete",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "", Namespace: ""},
						},
					},
				},
			},
			{
				APIGroup: "",
				Resource: "pods",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "create",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "", Namespace: ""},
						},
					},
					{
						Verb: "delete",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "", Namespace: ""},
						},
					},
				},
			},
			{
				APIGroup: "projectcalico.org",
				Resource: "networkpolicies",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "create",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "default", Namespace: ""},
							{Tier: "tier1", Namespace: ""},
							{Tier: "tier2", Namespace: ""},
							{Tier: "tier3", Namespace: ""},
							{Tier: "tier4", Namespace: ""},
						},
					},
					{
						Verb: "delete",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "default", Namespace: ""},
							{Tier: "tier1", Namespace: ""},
							{Tier: "tier2", Namespace: ""},
							{Tier: "tier3", Namespace: ""},
							{Tier: "tier4", Namespace: ""},
						},
					},
					{
						Verb: "patch",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "default", Namespace: ""},
							{Tier: "tier1", Namespace: ""},
							{Tier: "tier2", Namespace: ""},
							{Tier: "tier3", Namespace: ""},
							{Tier: "tier4", Namespace: ""},
						},
					},
				},
			},
		}))
	})

	It("missing cluster role binding", func() {
		mock.ClusterRoleBindings = []string{"missing", "allow-all"}
		mock.ClusterRoles = map[string][]rbac_v1.PolicyRule{
			"allow-all": {{Verbs: []string{"*"}, Resources: []string{"*"}, APIGroups: []string{"*"}}},
		}

		res, err := rest.Create(myContext, &projectcalico.AuthorizationReview{
			Spec: v3.AuthorizationReviewSpec{
				ResourceAttributes: []v3.AuthorizationReviewResourceAttributes{
					{
						APIGroup:  "",
						Resources: []string{"namespaces", "pods"},
						Verbs:     []string{"create", "delete"},
					},
					{
						APIGroup:  "projectcalico.org",
						Resources: []string{"networkpolicies"},
						// Try some duplicates to make sure they are contracted.
						Verbs: []string{"patch", "create", "delete", "patch", "delete"},
					},
				},
			},
		}, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		ar := res.(*projectcalico.AuthorizationReview)
		// Verbs other than get for namespace use cluster scoped if appropriate and will not expand across namespaces.
		Expect(ar.Status.AuthorizedResourceVerbs).To(HaveLen(3))
		Expect(ar.Status.AuthorizedResourceVerbs).To(Equal([]v3.AuthorizedResourceVerbs{
			{
				APIGroup: "",
				Resource: "namespaces",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "create",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "", Namespace: ""},
						},
					},
					{
						Verb: "delete",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "", Namespace: ""},
						},
					},
				},
			},
			{
				APIGroup: "",
				Resource: "pods",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "create",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "", Namespace: ""},
						},
					},
					{
						Verb: "delete",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "", Namespace: ""},
						},
					},
				},
			},
			{
				APIGroup: "projectcalico.org",
				Resource: "networkpolicies",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb: "create",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "default", Namespace: ""},
							{Tier: "tier1", Namespace: ""},
							{Tier: "tier2", Namespace: ""},
							{Tier: "tier3", Namespace: ""},
							{Tier: "tier4", Namespace: ""},
						},
					},
					{
						Verb: "delete",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "default", Namespace: ""},
							{Tier: "tier1", Namespace: ""},
							{Tier: "tier2", Namespace: ""},
							{Tier: "tier3", Namespace: ""},
							{Tier: "tier4", Namespace: ""},
						},
					},
					{
						Verb: "patch",
						ResourceGroups: []v3.AuthorizedResourceGroup{
							{Tier: "default", Namespace: ""},
							{Tier: "tier1", Namespace: ""},
							{Tier: "tier2", Namespace: ""},
							{Tier: "tier3", Namespace: ""},
							{Tier: "tier4", Namespace: ""},
						},
					},
				},
			},
		}))
	})
})
