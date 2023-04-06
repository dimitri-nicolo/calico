package namespace

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/syncer"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

var _ = Describe("Namespace Controller", func() {
	const (
		testResourceName = "TestName"
	)

	var (
		nr                    namespaceReconciler
		resourceCache         cache.ObjectCache[*v1.Namespace]
		k8sClient             kubernetes.Interface
		mockSynchronizer      client.MockQueryInterface
		mockedPolicyRecStatus v3.PolicyRecommendationScopeStatus
	)

	BeforeEach(func() {
		k8sClient = fake.NewSimpleClientset()
		resourceCache = cache.NewSynchronizedObjectCache[*v1.Namespace]()

		mockSynchronizer = client.MockQueryInterface{}
		mockedPolicyRecStatus = v3.PolicyRecommendationScopeStatus{
			Conditions: []v3.PolicyRecommendationScopeStatusCondition{
				{
					Message: "Ran at" + time.Now().String(),
					Status:  "enabled",
					Type:    "OK",
				},
			},
		}

		nr = *NewNamespaceReconciler(k8sClient, resourceCache, &mockSynchronizer)
	})

	AfterEach(func() {
		nr.Close()
	})

	It("adds an unseen namespace to the resourceCache by the resource's name", func() {
		ns, err := k8sClient.CoreV1().Namespaces().Create(
			context.Background(),
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testResourceName,
				},
			}, metav1.CreateOptions{})

		Expect(err).ShouldNot(HaveOccurred())

		var namespaceQueryArg syncer.NamespaceQuery
		mockSynchronizer.On("RunQuery", mock.Anything, mock.Anything).Run(
			func(args mock.Arguments) {
				namespaceQueryArg = args[1].(syncer.NamespaceQuery)
			},
		).Return(mockedPolicyRecStatus, nil)

		err = nr.Reconcile(types.NamespacedName{
			Name: ns.Name,
		})
		Expect(err).To(BeNil())

		namespace := resourceCache.Get(testResourceName)
		Expect(namespace).ToNot(BeNil())

		Expect(namespaceQueryArg.MetaSelectors.Source.KVPair.Key).To(Equal(
			model.ResourceKey{
				Name: namespace.Name,
				Kind: namespace.Kind,
			},
		))
		Expect(namespaceQueryArg.MetaSelectors.Source.UpdateType).To(Equal(api.UpdateTypeKVNew))
	})

	It("deletes an already stored namespace in the cache if it's not found in the cluster", func() {
		resourceCache.Set(
			testResourceName,
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testResourceName,
				},
			},
		)

		var namespaceQueryArg syncer.NamespaceQuery
		mockSynchronizer.On("RunQuery", mock.Anything, mock.Anything).Run(
			func(args mock.Arguments) {
				namespaceQueryArg = args[1].(syncer.NamespaceQuery)
			},
		).Return(mockedPolicyRecStatus, nil)

		err := nr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})
		Expect(err).To(BeNil())

		namespace := resourceCache.Get(testResourceName)
		Expect(namespace).To(BeNil())

		Expect(namespaceQueryArg.MetaSelectors.Source.KVPair.Key).To(Equal(
			model.ResourceKey{
				Name: testResourceName,
				Kind: KindNamespace,
			},
		))
		Expect(namespaceQueryArg.MetaSelectors.Source.UpdateType).To(Equal(api.UpdateTypeKVDeleted))
	})
})
