package policyrecommendation

//TODO(dimitrin): [EV-3439] Re-write tests and add back
// 								- Address empty namespace and staged network policy caches
// 								- Address empty elastic search query results
// 								- Mock the engine reconcilation run

// import (
// 	"context"
// 	"time"

// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"

// 	"github.com/stretchr/testify/mock"

// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/types"

// 	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
// 	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
// 	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

// 	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
// 	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
// 	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
// 	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
// )

// var _ = Describe("Policy Recommendation Scope Controller", func() {
// 	const (
// 		testResourceName = "TestName"
// 	)

// 	var (
// 		testCtx               context.Context
// 		testCancel            context.CancelFunc
// 		pr                    policyRecommendationReconciler
// 		calicoCLI             calicoclient.ProjectcalicoV3Interface
// 		mockSynchronizer      client.MockQueryInterface
// 		mockedPolicyRecStatus v3.PolicyRecommendationScopeStatus
// 	)

// 	BeforeEach(func() {
// 		calicoCLI = fake.NewSimpleClientset().ProjectcalicoV3()
// 		testCtx, testCancel = context.WithCancel(context.Background())

// 		mockSynchronizer = client.MockQueryInterface{}
// 		mockedPolicyRecStatus = v3.PolicyRecommendationScopeStatus{
// 			Conditions: []v3.PolicyRecommendationScopeStatusCondition{
// 				{
// 					Message: "Ran at" + time.Now().String(),
// 					Status:  "enabled",
// 					Type:    "OK",
// 				},
// 			},
// 		}

// 		pr = policyRecommendationReconciler{
// 			calico:       calicoCLI,
// 			synchronizer: &mockSynchronizer,
// 		}
// 	})

// 	AfterEach(func() {
// 		pr.Close()
// 		testCancel()
// 	})

// 	It("sets the controller state if the PolicyRecScope is found in the cluster", func() {

// 		prScopeInCluster := &v3.PolicyRecommendationScope{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: testResourceName,
// 			},
// 		}
// 		_, err := calicoCLI.PolicyRecommendationScopes().Create(
// 			testCtx,
// 			prScopeInCluster,
// 			metav1.CreateOptions{},
// 		)

// 		Expect(err).To(BeNil())

// 		var policyRecQueryArg syncer.PolicyRecommendationScopeQuery
// 		mockSynchronizer.On("RunQuery", testCtx, mock.Anything).Run(
// 			func(args mock.Arguments) {
// 				policyRecQueryArg = args[1].(syncer.PolicyRecommendationScopeQuery)
// 			},
// 		).Return(mockedPolicyRecStatus, nil)

// 		err = pr.Reconcile(types.NamespacedName{
// 			Name: testResourceName,
// 		})

// 		Expect(err).To(BeNil())

// 		Expect(pr.state.object).To(Equal(*prScopeInCluster))
// 		// TODO: check that engine is run once it's integrated

// 		Expect(policyRecQueryArg.MetaSelectors.Source.KVPair.Key).To(Equal(
// 			model.ResourceKey{
// 				Name: prScopeInCluster.Name,
// 				Kind: prScopeInCluster.Kind,
// 			},
// 		))

// 		Expect(policyRecQueryArg.MetaSelectors.Source.UpdateType).To(Equal(api.UpdateTypeKVNew))
// 	})

// 	It("cancels the engine and removes the state if the policyrec is not found", func() {
// 		prScopeState := &v3.PolicyRecommendationScope{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: testResourceName,
// 			},
// 		}

// 		_, cancel := context.WithCancel(context.Background())
// 		defer cancel()
// 		pr.state = &policyRecommendationScopeState{
// 			object: *prScopeState,
// 			cancel: cancel,
// 		}

// 		var policyRecQueryArg syncer.PolicyRecommendationScopeQuery
// 		mockSynchronizer.On("RunQuery", mock.Anything, mock.Anything).Run(
// 			func(args mock.Arguments) {
// 				policyRecQueryArg = args[1].(syncer.PolicyRecommendationScopeQuery)
// 			},
// 		).Return(mockedPolicyRecStatus, nil)

// 		err := pr.Reconcile(types.NamespacedName{
// 			Name: testResourceName,
// 		})

// 		Expect(err).To(BeNil())
// 		Expect(pr.state).To(BeNil())
// 		// TODO: check that engine is run once it's integrated

// 		Expect(policyRecQueryArg.MetaSelectors.Source.KVPair.Key).To(Equal(
// 			model.ResourceKey{
// 				Name: testResourceName,
// 				Kind: v3.KindPolicyRecommendationScope,
// 			},
// 		))
// 		Expect(policyRecQueryArg.MetaSelectors.Source.UpdateType).To(Equal(api.UpdateTypeKVDeleted))
// 	})

// 	It("updates the state with the one found in the cluster", func() {
// 		timeDuration := metav1.Duration{Duration: 150 * time.Second}
// 		prScopeState := &v3.PolicyRecommendationScope{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: testResourceName,
// 			},
// 			Spec: v3.PolicyRecommendationScopeSpec{
// 				Interval: &timeDuration,
// 			},
// 		}

// 		_, cancel := context.WithCancel(context.Background())
// 		defer cancel()
// 		pr.state = &policyRecommendationScopeState{
// 			object: *prScopeState,
// 			cancel: cancel,
// 		}

// 		updatedTimeDuration := metav1.Duration{Duration: 120 * time.Second}
// 		prScopeInCluster := &v3.PolicyRecommendationScope{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: testResourceName,
// 			},
// 			Spec: v3.PolicyRecommendationScopeSpec{
// 				Interval: &updatedTimeDuration,
// 			},
// 		}

// 		_, err := calicoCLI.PolicyRecommendationScopes().Create(
// 			testCtx,
// 			prScopeInCluster,
// 			metav1.CreateOptions{},
// 		)

// 		Expect(err).To(BeNil())

// 		err = pr.Reconcile(types.NamespacedName{
// 			Name: testResourceName,
// 		})
// 		Expect(err).To(BeNil())

// 		Expect(pr.state.object).To(Equal(*prScopeInCluster))
// 		// TODO: check that engine is run once it's integrated
// 	})

// })

// var _ = Describe("Policy Recommendation Scope Reconciler Utilities", func() {
// 	It("define default values for the policy recommendation scope", func() {
// 		scope := &v3.PolicyRecommendationScope{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: "test-scope",
// 			},
// 			Spec: v3.PolicyRecommendationScopeSpec{
// 				NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
// 					RecStatus: v3.PolicyRecommendationScopeEnabled,
// 				},
// 			},
// 		}

// 		expectedMaxRules := 20
// 		expectedPoliciesLearningCutOff := 20

// 		// Initialize the scope with its default values
// 		setPolicyRecommendationScopeDefaults(scope)

// 		Expect(scope.Name).To(Equal("test-scope"))
// 		Expect(scope.Spec.InitialLookback).To(Equal(&metav1.Duration{Duration: 24 * time.Hour}))
// 		Expect(scope.Spec.Interval).To(Equal(&metav1.Duration{Duration: 150 * time.Second}))
// 		Expect(scope.Spec.StabilizationPeriod).To(Equal(&metav1.Duration{Duration: 10 * time.Minute}))
// 		Expect(scope.Spec.MaxRules).To(Equal(&expectedMaxRules))
// 		Expect(scope.Spec.PoliciesLearningCutOff).To(Equal(&expectedPoliciesLearningCutOff))
// 		Expect(scope.Spec.NamespaceSpec.RecStatus).To(Equal(v3.PolicyRecommendationScopeEnabled))
// 		Expect(scope.Spec.NamespaceSpec.Selector).To(Equal(""))
// 		Expect(scope.Spec.NamespaceSpec.TierName).To(Equal("namespace-segmentation"))
// 	})
// })
