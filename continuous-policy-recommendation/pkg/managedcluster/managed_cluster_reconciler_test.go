package managedcluster

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeK8s "k8s.io/client-go/kubernetes/fake"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/controller"
	controller_mocks "github.com/projectcalico/calico/continuous-policy-recommendation/pkg/controller/mocks"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
)

var _ = Describe("ManagedCluster reconciler test", func() {
	const (
		testResourceName = "TestName"
	)

	var (
		testCtx    context.Context
		testCancel context.CancelFunc
		mr         managedClusterReconciler
		calicoCLI  calicoclient.ProjectcalicoV3Interface
	)

	BeforeEach(func() {
		calicoCLI = fake.NewSimpleClientset().ProjectcalicoV3()
		testCtx, testCancel = context.WithCancel(context.Background())

		mr = managedClusterReconciler{
			managementStandaloneCalico: calicoCLI,
			cache:                      make(map[string]*managedClusterState),
		}
	})

	AfterEach(func() {
		testCancel()
		mr.Close()
	})

	It("cancels the stored controllers for the cluster if the managed clusters is not found in the cluster", func() {
		_, ctrlCancel := context.WithCancel(testCtx)

		mockController := &controller_mocks.MockController{}
		mockController.On("Close").Return(nil).Once()

		mr.cache = map[string]*managedClusterState{
			testResourceName: {
				clusterName: "cluster",
				cancel:      ctrlCancel,
				controllers: []controller.Controller{mockController},
			},
		}

		err := mr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})

		Expect(err).To(BeNil())

		Expect(mr.cache[testResourceName]).To(BeNil())

	})

	It("ignores creating the manged cluster state if it's status is not connected", func() {
		_, err := calicoCLI.ManagedClusters().Create(
			testCtx,
			&v3.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: testResourceName,
				},
				Status: v3.ManagedClusterStatus{
					Conditions: []v3.ManagedClusterStatusCondition{
						{
							Status: v3.ManagedClusterStatusValueFalse,
						},
					},
				},
			},
			metav1.CreateOptions{},
		)

		Expect(err).To(BeNil())

		err = mr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})

		Expect(err).To(BeNil())

		Expect(mr.cache[testResourceName]).To(BeNil())
	})

	It("intializes controllers for the managed clusters if its status is connected", func() {
		mockClientSet := lmak8s.MockClientSet{}
		mockClientSet.On("ProjectcalicoV3").Return(
			fake.NewSimpleClientset().ProjectcalicoV3(),
		)
		mockClientSet.On("CoreV1").Return(
			fakeK8s.NewSimpleClientset().CoreV1(),
		)

		mockClientFactory := lmak8s.MockClientSetFactory{}
		mockClientFactory.On("NewClientSetForApplication", testResourceName).Return(
			&mockClientSet,
			nil,
		)

		mockESClient := lmaelastic.MockClient{}
		mockESClient.On("CreateEventsIndex", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

		mockEsClientFactory := lmaelastic.MockClusterContextClientFactory{}
		mockEsClientFactory.On("ClientForCluster", testResourceName).Return(&mockESClient, nil)

		mr.clientFactory = &mockClientFactory
		mr.elasticClientFactory = &mockEsClientFactory

		_, err := calicoCLI.ManagedClusters().Create(
			testCtx,
			&v3.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: testResourceName,
				},
				Status: v3.ManagedClusterStatus{
					Conditions: []v3.ManagedClusterStatusCondition{
						{
							Type:   v3.ManagedClusterStatusTypeConnected,
							Status: v3.ManagedClusterStatusValueTrue,
						},
					},
				},
			},
			metav1.CreateOptions{},
		)

		Expect(err).To(BeNil())

		err = mr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})

		Expect(err).To(BeNil())

		savedClusterState, ok := mr.cache[testResourceName]
		Expect(ok).To(BeTrue())
		Expect(savedClusterState.clusterName).To(Equal(testResourceName))
		Expect(len(savedClusterState.controllers)).To(Equal(3))
		Expect(savedClusterState.cancel).ToNot(BeNil())
	})
})
