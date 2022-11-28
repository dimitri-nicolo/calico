package stagednetworkpolicies

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
)

var _ = Describe("Staged Network Policies Controller", func() {
	const (
		testResourceName      = "TestName"
		testResourceNamespace = "TestNamespace"
	)

	var (
		testCtx       context.Context
		testCancel    context.CancelFunc
		snpr          stagednetworkpoliciesReconciler
		resourceCache cache.ObjectCache[*v3.StagedNetworkPolicy]
		calicoCLI     calicoclient.ProjectcalicoV3Interface
	)

	BeforeEach(func() {
		calicoCLI = fake.NewSimpleClientset().ProjectcalicoV3()
		resourceCache = cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()

		testCtx, testCancel = context.WithCancel(context.Background())
		snpr = stagednetworkpoliciesReconciler{
			calico:        calicoCLI,
			resourceCache: resourceCache,
		}
	})

	AfterEach(func() {
		snpr.Close()
		testCancel()
	})

	It("ignores SNPs untracked by the resourceCache", func() {
		err := snpr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})
		Expect(err).To(BeNil())
		snp := resourceCache.Get(testResourceName)
		Expect(snp).To(BeNil())
	})

	It("restores the deleted SNPs tracked by the resourceCache if not found in the cluster", func() {
		resourceCache.Set(testResourceName, &v3.StagedNetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testResourceName,
				Namespace: testResourceNamespace,
			},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: "Set",
			},
		})

		err := snpr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})
		Expect(err).To(BeNil())
		snp := resourceCache.Get(testResourceName)
		Expect(snp).ToNot(BeNil())
		Expect(snp.Spec.StagedAction).To(Equal(v3.StagedActionSet))
	})

	It("restores the altered SNPs found in the cluster with the one tracked by the resourceCache", func() {
		expectedSNP := &v3.StagedNetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testResourceName,
				Namespace: testResourceNamespace,
			},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: "Set",
			},
		}

		resourceCache.Set(testResourceName, expectedSNP)

		_, err := calicoCLI.StagedNetworkPolicies(testResourceNamespace).Create(
			testCtx,
			&v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testResourceNamespace,
				},
				Spec: v3.StagedNetworkPolicySpec{
					StagedAction: "Delete",
				},
			},
			metav1.CreateOptions{},
		)
		Expect(err).ShouldNot(HaveOccurred())

		err = snpr.Reconcile(types.NamespacedName{
			Name:      testResourceName,
			Namespace: testResourceNamespace,
		})

		Expect(err).To(BeNil())

		snp, err := calicoCLI.StagedNetworkPolicies(testResourceNamespace).Get(
			testCtx,
			testResourceName,
			metav1.GetOptions{},
		)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(snp).ToNot(BeNil())
		Expect(*snp).To(Equal(*expectedSNP))
	})

})
