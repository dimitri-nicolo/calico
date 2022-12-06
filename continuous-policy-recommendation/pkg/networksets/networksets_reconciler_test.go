package networksets

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

var _ = Describe("Network Set Controller", func() {
	const (
		testResourceName      = "TestName"
		testResourceNamespace = "TestNamespace"
	)

	var (
		testCtx        context.Context
		testCancel     context.CancelFunc
		nsr            networksetReconciler
		testNetworkSet *v3.NetworkSet
		resourceCache  cache.ObjectCache[*v3.NetworkSet]
		calicoCLI      calicoclient.ProjectcalicoV3Interface
		sampleNets     = []string{
			"198.51.100.0/28",
			"203.0.113.0/24",
		}
	)

	BeforeEach(func() {
		calicoCLI = fake.NewSimpleClientset().ProjectcalicoV3()
		resourceCache = cache.NewSynchronizedObjectCache[*v3.NetworkSet]()
		testNetworkSet = &v3.NetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testResourceName,
				Namespace: testResourceNamespace,
			},
			Spec: v3.NetworkSetSpec{
				Nets: sampleNets,
			},
		}

		testCtx, testCancel = context.WithCancel(context.Background())
		nsr = networksetReconciler{
			calico:        calicoCLI,
			resourceCache: resourceCache,
		}
	})

	AfterEach(func() {
		nsr.Close()
		testCancel()
	})

	It("adds untracked NetworkSets to the resourceCache", func() {
		_, err := calicoCLI.NetworkSets(testResourceNamespace).Create(
			testCtx,
			testNetworkSet,
			metav1.CreateOptions{})

		Expect(err).To(BeNil())

		err = nsr.Reconcile(types.NamespacedName{
			Name:      testResourceName,
			Namespace: testResourceNamespace,
		})

		Expect(err).To(BeNil())
		snp := resourceCache.Get(testResourceName)
		Expect(snp).ToNot(BeNil())
		Expect(*snp).To(Equal(*testNetworkSet))
	})

	It("deletes the NetworkSet found in the cluster from the resourceCache", func() {
		resourceCache.Set(testResourceName, testNetworkSet)

		err := nsr.Reconcile(types.NamespacedName{
			Name:      testResourceName,
			Namespace: testResourceNamespace,
		})

		Expect(err).To(BeNil())
		networkset := resourceCache.Get(testResourceName)
		Expect(networkset).To(BeNil())
	})

	It("updates the NetworkSet tracked by the resourceCache on a cluster update", func() {
		resourceCache.Set(testResourceName, testNetworkSet)

		updatedNS := &v3.NetworkSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testResourceName,
				Namespace: testResourceNamespace,
			},
			Spec: v3.NetworkSetSpec{
				Nets: []string{
					"198.51.101.0/28",
					"203.0.114.0/24",
				},
			},
		}
		_, err := calicoCLI.NetworkSets(testResourceNamespace).Create(
			testCtx,
			updatedNS,
			metav1.CreateOptions{},
		)
		Expect(err).ShouldNot(HaveOccurred())

		err = nsr.Reconcile(types.NamespacedName{
			Name:      testResourceName,
			Namespace: testResourceNamespace,
		})

		Expect(err).ShouldNot(HaveOccurred())
		result := resourceCache.Get(testResourceName)
		Expect(result).ToNot(BeNil())
		Expect(result.Spec).To(Equal(updatedNS.Spec))
	})

})
