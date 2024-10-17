package auth

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"

	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	clientsetfake "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
)

var _ = Describe("queryserver resource authorizer tests", func() {
	var authz Authorizer
	var mockClientSetFactory *lmak8s.MockClientSetFactory

	BeforeEach(func() {
		mockClientSet := &lmak8s.MockClientSet{}
		mockClientSetFactory = lmak8s.NewMockClientSetFactory(GinkgoT())
		mockClientSetFactory.On("NewClientSetForApplication", mock.Anything, mock.Anything).Return(mockClientSet, nil)
		mockClientSet.On("ProjectcalicoV3").Return(clientsetfake.NewSimpleClientset().ProjectcalicoV3())

		authz = NewAuthorizer(mockClientSetFactory)
	})

	Context("Test authorizer.PerformUserAuthorizationReview", func() {
		It("return user unauthorized when user is not set in context", func() {
			_, err := authz.PerformUserAuthorizationReview(context.TODO(), nil)
			Expect(err).Should(HaveOccurred())
		})
		It("return permissions for the set user", func() {
			ctx := request.WithUser(context.TODO(), &user.DefaultInfo{
				Name:   "qs-authz-test",
				UID:    "id-12345",
				Groups: nil,
				Extra:  nil,
			})
			permissions, err := authz.PerformUserAuthorizationReview(ctx, nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(permissions).ToNot(BeNil())
		})
	})

	Context("Test permissions", func() {
		authorizedResourceVerbs := []v3.AuthorizedResourceVerbs{
			{
				APIGroup: "projectcalico.org",
				Resource: "networkpolicies",
				Verbs: []v3.AuthorizedResourceVerb{
					{
						Verb:           "get",
						ResourceGroups: []v3.AuthorizedResourceGroup{{Namespace: "ns-a"}},
					},
					{
						Verb:           "list",
						ResourceGroups: []v3.AuthorizedResourceGroup{{Namespace: "ns-b"}},
					},
				},
			},
		}
		It("test convertAuthorizationReviewStatusToPermissions", func() {
			permissions, err := convertAuthorizationReviewStatusToPermissions(authorizedResourceVerbs)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(permissions).ToNot(BeNil())

		})

		It("test Permissions.IsAuthorized", func() {
			permissions, err := convertAuthorizationReviewStatusToPermissions(authorizedResourceVerbs)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(permissions).ToNot(BeNil())

			networkpolicy1 := v3.NewNetworkPolicy()
			networkpolicy1.ObjectMeta = metav1.ObjectMeta{
				Name:      "netpolicy1",
				Namespace: "ns-a",
			}
			networkpolicy1.Spec = v3.NetworkPolicySpec{
				Tier: "default",
			}

			Expect(permissions.IsAuthorized(networkpolicy1, nil, []string{"get"})).To(BeTrue())

			networkpolicy1 = v3.NewNetworkPolicy()
			networkpolicy1.ObjectMeta = metav1.ObjectMeta{
				Name:      "netpolicy1",
				Namespace: "ns-b",
			}
			networkpolicy1.Spec = v3.NetworkPolicySpec{
				Tier: "default",
			}
			Expect(permissions.IsAuthorized(networkpolicy1, "default", []string{"get"})).To(BeFalse())
		})
	})

})
