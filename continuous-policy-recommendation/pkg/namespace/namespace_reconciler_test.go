package namespace

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
)

var _ = Describe("Namespace Controller", func() {
	const (
		testResourceName = "TestName"
	)

	var (
		nr            namespaceReconciler
		resourceCache cache.ObjectCache[*v1.Namespace]
		k8sClient     kubernetes.Interface
	)

	BeforeEach(func() {
		k8sClient = fake.NewSimpleClientset()
		resourceCache = cache.NewSynchronizedObjectCache[*v1.Namespace]()

		nr = namespaceReconciler{
			k8sClient,
			resourceCache,
		}
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

		err = nr.Reconcile(types.NamespacedName{
			Name: ns.Name,
		})
		Expect(err).To(BeNil())
		namespace := resourceCache.Get(testResourceName)

		Expect(namespace).ToNot(BeNil())
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

		err := nr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})
		Expect(err).To(BeNil())
		namespace := resourceCache.Get(testResourceName)
		Expect(namespace).To(BeNil())
	})
})
