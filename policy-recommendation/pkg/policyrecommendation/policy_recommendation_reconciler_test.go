package policyrecommendation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"

	"github.com/projectcalico/calico/policy-recommendation/pkg/cache"
	calicores "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	calres "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
)

//TODO(dimitrin): [EV-3439] Re-write tests and add back
// 								- Address empty namespace and staged network policy caches
// 								- Address empty search query results
// 								- Mock the engine reconciliation run

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
// 		Expect(scope.Spec.NamespaceSpec.TierName).To(Equal("namespace-isolation"))
// 	})
// })

const timeNowRFC3339 = "2022-11-30T09:01:38Z"

type MockClock struct{}

func (MockClock) NowRFC3339() string { return timeNowRFC3339 }

var _ = Describe("updateStatusAnnotation", func() {
	const (
		defaultInterval      = 150 * time.Second
		defaultStabilization = 10 * time.Minute
	)

	ts, err := time.Parse(time.RFC3339, timeNowRFC3339)
	Expect(err).To(BeNil())

	pr := policyRecommendationReconciler{
		state: &policyRecommendationScopeState{
			object: v3.PolicyRecommendationScope{
				Spec: v3.PolicyRecommendationScopeSpec{
					Interval:            &metav1.Duration{},
					StabilizationPeriod: &metav1.Duration{},
				},
			},
		},
		clock: MockClock{},
	}

	var _ = DescribeTable("Status update",
		func(lastUpdateDuration time.Duration, interval time.Duration, stabilization time.Duration, expectedStatus string) {
			log.Infof("interval: %f, stabilization: %f, expectedStatus: %s", interval.Seconds(), stabilization.Seconds(), expectedStatus)

			lastUpdate := map[string]string{
				calres.LastUpdatedKey: ts.Add(-(lastUpdateDuration)).Format(time.RFC3339),
			}

			pr.state.object.Spec.Interval.Duration = interval
			pr.state.object.Spec.StabilizationPeriod.Duration = stabilization

			snp := &v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-recommendation-xvjfz",
					Annotations: lastUpdate,
				},
				Spec: v3.StagedNetworkPolicySpec{
					Egress: []v3.Rule{
						{},
					},
					Types: []v3.PolicyType{
						"Egress",
					},
				},
			}

			pr.updateStatusAnnotation(snp)
			Expect(snp.Annotations[calres.StatusKey]).To(Equal(expectedStatus))
		},
		Entry("LearningStatus with default interval and stabilization periods",
			2*defaultInterval,     // Time equal to the learning period
			defaultInterval,       // Duration of the engine interval
			defaultStabilization,  // Duration of the stabilization period
			calres.LearningStatus, // Expected status annotation
		),
		Entry("StabilizingStatus with default interval and stabilization periods",
			2*defaultInterval+time.Second, // One second after the learning and still within the stable period
			defaultInterval,
			defaultStabilization,
			calres.StabilizingStatus,
		),
		Entry("StableStatus with default interval and stabilization periods",
			defaultStabilization+time.Second, // One second after the stable period
			defaultInterval,
			defaultStabilization,
			calres.StableStatus,
		),
		Entry("LearningStatus with updated interval and default stabilization periods",
			2*defaultInterval,           // Well within the updated learning period
			defaultInterval+time.Second, // The updated learning period
			defaultStabilization,
			calres.LearningStatus,
		),
		Entry("StabilizingStatus with updated interval and default stabilization periods",
			2*defaultInterval+(3*time.Second), // One second after the updated learning period and within the stable period
			defaultInterval+time.Second,       // The updated learning period
			defaultStabilization,
			calres.StabilizingStatus,
		),
		Entry("StableStatus with default interval and updated stabilization periods",
			defaultStabilization+2*time.Second, // One second after the updated stable period
			defaultInterval,
			defaultStabilization+time.Second, // The updated stable period
			calres.StableStatus,
		),
	)
})

var _ = Describe("policyRecommendationReconciler", func() {
	var (
		ctx    context.Context
		pr     *policyRecommendationReconciler
		client calicoclient.ProjectcalicoV3Interface
	)

	var (
		labelSelector = strings.Join(
			[]string{
				fmt.Sprintf("%s=%s", calicores.TierKey, "namespace-isolation"),
				fmt.Sprintf("%s=%s", calicores.OwnerReferenceKindKey, v3.KindPolicyRecommendationScope),
			}, ",",
		)
	)

	BeforeEach(func() {
		ctx = context.Background()
		client = fake.NewSimpleClientset().ProjectcalicoV3()
		pr = &policyRecommendationReconciler{
			calico: client,
			caches: &syncer.CacheSet{
				StagedNetworkPolicies: cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy](),
			},
			state: &policyRecommendationScopeState{
				object: v3.PolicyRecommendationScope{
					Spec: v3.PolicyRecommendationScopeSpec{
						NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
							TierName: "namespace-isolation",
						},
					},
				},
			},
		}
	})

	DescribeTable("getRecommendation",
		func(name, namespace string, store []v3.StagedNetworkPolicy, expected *v3.StagedNetworkPolicy, expectedErr error) {

			// Add the objects to the datastore
			addObjects(ctx, client, pr, store)

			actual, err := pr.getRecFromStore(ctx, name, namespace)
			if expectedErr != nil {
				Expect(err).To(Equal(expectedErr))
			} else {
				Expect(*actual).To(Equal(*expected))
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("Recommendation exists",
			"test-snp",
			"test-ns",
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Egress:  []v3.Rule{},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{},
					},
				},
			},
			&v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-snp",
					Namespace: "test-ns",
					Labels: map[string]string{
						calicores.TierKey:                       "namespace-isolation",
						"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					},
					Annotations: map[string]string{},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "projectcalico.org/v3",
							Kind:               "PolicyRecommendationScope",
							Controller:         &[]bool{true}[0],
							BlockOwnerDeletion: &[]bool{false}[0],
						},
					},
				},
				Spec: v3.StagedNetworkPolicySpec{
					Egress:  []v3.Rule{},
					Ingress: []v3.Rule{},
					Types:   []v3.PolicyType{},
				},
			},
			nil,
		),
		Entry("Recommendation does not exist",
			"test-snp",
			"test-ns",
			[]v3.StagedNetworkPolicy{},
			&v3.StagedNetworkPolicy{},
			k8serrors.NewNotFound(v3.Resource("stagednetworkpolicies"), "test-snp"),
		),
		Entry("Another recommendation exists",
			"test-sn1",
			"test-ns",
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp-2",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Egress:  []v3.Rule{},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{},
					},
				},
			},
			&v3.StagedNetworkPolicy{},
			fmt.Errorf("recommendation name test-snp-2 differs from test-sn1, in namespace: test-ns"),
		),
		Entry("Multiple recommendations exist",
			"test-snp1",
			"test-ns",
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp-1",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Egress:  []v3.Rule{},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp-2",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Egress:  []v3.Rule{},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{},
					},
				},
			},
			&v3.StagedNetworkPolicy{},
			fmt.Errorf("more than one recommendation in namespace: test-ns"),
		),
	)

	DescribeTable("syncToDatastore",
		func(key, namespace string, cache *v3.StagedNetworkPolicy, store, expected []v3.StagedNetworkPolicy, expectedErr error) {
			// Add the objects to the datastore
			addObjects(ctx, client, pr, store)

			// Call the syncToDatastore function
			err := pr.syncToDatastore(ctx, key, namespace, cache)

			// Check the results
			if expectedErr != nil {
				Expect(err).To(MatchError(expectedErr))
			} else {

				actual, err := pr.calico.StagedNetworkPolicies(namespace).List(ctx, metav1.ListOptions{
					LabelSelector: labelSelector,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(actual.Items).To(Equal(expected))
			}
		},
		Entry("Cache is nil, datastore policy exists and is learning",
			"test-snp",
			"test-ns",
			nil,
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionLearn,
					},
				},
			},
			nil,
			nil,
		),
		Entry("Cache is nil, datastore policy exists and is not learning",
			"test-snp",
			"test-ns",
			nil,
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionDelete,
					},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionDelete,
					},
				},
			},
			errors.New("can only delete a learning policy"),
		),
		Entry("Cache is nil, datastore policy does not exist",
			"test-snp",
			"test-ns",
			nil,
			[]v3.StagedNetworkPolicy{},
			[]v3.StagedNetworkPolicy{},
			k8serrors.NewNotFound(v3.Resource("stagednetworkpolicies"), "test-snp"),
		),
		Entry("Cache is not nil, datastore policy exists and is learning",
			"test-snp",
			"test-ns",
			&v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-snp",
					Namespace: "test-ns",
					Labels: map[string]string{
						calicores.TierKey:                       "namespace-isolation",
						"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					},
				},
				Spec: v3.StagedNetworkPolicySpec{
					Tier:         "namespace-isolation",
					StagedAction: v3.StagedActionLearn,
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionLearn,
					},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionLearn,
					},
				},
			},
			nil,
		),
		Entry("Cache is not nil, and its rules are empty",
			"test-snp",
			"test-ns",
			&v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-snp",
					Namespace: "test-ns",
					Labels: map[string]string{
						calicores.TierKey:                       "namespace-isolation",
						"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					},
					Annotations: map[string]string{},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "projectcalico.org/v3",
							Kind:               "PolicyRecommendationScope",
							Controller:         &[]bool{true}[0],
							BlockOwnerDeletion: &[]bool{false}[0],
						},
					},
				},
				Spec: v3.StagedNetworkPolicySpec{
					Tier:    "namespace-isolation",
					Egress:  []v3.Rule{},
					Ingress: []v3.Rule{},
					Types:   []v3.PolicyType{},
				},
			},
			[]v3.StagedNetworkPolicy{},
			nil,
			nil,
		),
		Entry("Cache is not nil, datastore contains a different policy",
			"test-snp1",
			"test-ns",
			&v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-snp1",
					Namespace: "test-ns",
					Labels: map[string]string{
						calicores.TierKey:                       "namespace-isolation",
						"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					},
					Annotations: map[string]string{},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "projectcalico.org/v3",
							Kind:               "PolicyRecommendationScope",
							Controller:         &[]bool{true}[0],
							BlockOwnerDeletion: &[]bool{false}[0],
						},
					},
				},
				Spec: v3.StagedNetworkPolicySpec{
					Tier: "namespace-isolation",
					Egress: []v3.Rule{
						{
							Action: "Allow",
						},
					},
					Ingress: []v3.Rule{},
					Types:   []v3.PolicyType{"Egress"},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp2",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier: "namespace-isolation",
						Egress: []v3.Rule{
							{
								Action: "Allow",
							},
						},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{"Egress"},
					},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp2",
						Namespace: "test-ns2",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier: "namespace-isolation",
						Egress: []v3.Rule{
							{
								Action: "Allow",
							},
						},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{"Egress"},
					},
				},
			},
			fmt.Errorf("recommendation name test-snp2 differs from test-snp1, in namespace: %s", "test-ns"),
		),

		Entry("Cache is not nil, datastore policy exists and is learning, cache and store items are equal",
			"test-snp",
			"test-ns",
			&v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-snp",
					Namespace: "test-ns",
					Labels: map[string]string{
						calicores.TierKey:                       "namespace-isolation",
						"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					},
					Annotations: map[string]string{},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "projectcalico.org/v3",
							Kind:               "PolicyRecommendationScope",
							Controller:         &[]bool{true}[0],
							BlockOwnerDeletion: &[]bool{false}[0],
						},
					},
				},
				Spec: v3.StagedNetworkPolicySpec{
					Tier: "namespace-isolation",
					Egress: []v3.Rule{
						{
							Action: "Allow",
						},
					},
					Ingress: []v3.Rule{},
					Types:   []v3.PolicyType{"Egress"},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionSet,
						Egress: []v3.Rule{
							{
								Action: "Allow",
							},
						},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{"Egress"},
					},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionSet,
						Egress: []v3.Rule{
							{
								Action: "Allow",
							},
						},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{"Egress"},
					},
				},
			},
			nil,
		),

		Entry("Cache is not nil, datastore policy exists and is not learning",
			"test-snp",
			"test-ns",
			&v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-snp",
					Namespace: "test-ns",
					Labels: map[string]string{
						calicores.TierKey:                       "namespace-isolation",
						"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					},
					Annotations: map[string]string{},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "projectcalico.org/v3",
							Kind:               "PolicyRecommendationScope",
							Controller:         &[]bool{true}[0],
							BlockOwnerDeletion: &[]bool{false}[0],
						},
					},
				},
				Spec: v3.StagedNetworkPolicySpec{
					Tier: "namespace-isolation",
					Egress: []v3.Rule{
						{
							Action: "Allow",
						},
					},
					Ingress: []v3.Rule{},
					Types:   []v3.PolicyType{"Egress"},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionSet,
						Egress:       []v3.Rule{},
						Ingress: []v3.Rule{
							{
								Action: "Allow",
							},
						},
						Types: []v3.PolicyType{"Ingress"},
					},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp",
						Namespace: "test-ns",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionSet,
						Egress:       []v3.Rule{},
						Ingress: []v3.Rule{
							{
								Action: "Allow",
							},
						},
						Types: []v3.PolicyType{"Ingress"},
					},
				},
			},
			nil,
		),
		Entry("Cache is not nil, datastore policy does not exist",
			"test-snp1",
			"test-ns1",
			&v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-snp1",
					Namespace: "test-ns1",
					Labels: map[string]string{
						calicores.TierKey:                       "namespace-isolation",
						"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					},
					Annotations: map[string]string{},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "projectcalico.org/v3",
							Kind:               "PolicyRecommendationScope",
							Controller:         &[]bool{true}[0],
							BlockOwnerDeletion: &[]bool{false}[0],
						},
					},
				},
				Spec: v3.StagedNetworkPolicySpec{
					Tier:         "namespace-isolation",
					StagedAction: v3.StagedActionLearn,
					Egress: []v3.Rule{
						{
							Action: "Allow",
						},
					},
					Ingress: []v3.Rule{},
					Types:   []v3.PolicyType{"Egress"},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp2",
						Namespace: "test-ns2",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionLearn,
						Egress: []v3.Rule{
							{
								Action: "Allow",
							},
						},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{"Egress"},
					},
				},
			},
			[]v3.StagedNetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snp1",
						Namespace: "test-ns1",
						Labels: map[string]string{
							calicores.TierKey:                       "namespace-isolation",
							"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
						},
						Annotations: map[string]string{},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "projectcalico.org/v3",
								Kind:               "PolicyRecommendationScope",
								Controller:         &[]bool{true}[0],
								BlockOwnerDeletion: &[]bool{false}[0],
							},
						},
					},
					Spec: v3.StagedNetworkPolicySpec{
						Tier:         "namespace-isolation",
						StagedAction: v3.StagedActionLearn,
						Egress: []v3.Rule{
							{
								Action: "Allow",
							},
						},
						Ingress: []v3.Rule{},
						Types:   []v3.PolicyType{"Egress"},
					},
				},
			},
			nil,
		),
	)
})

// addObjects adds the given objects to the cache and datastore. It attaches a recommendation owner
// reference to each policy.
func addObjects(ctx context.Context, client calicoclient.ProjectcalicoV3Interface, pr *policyRecommendationReconciler, snps []v3.StagedNetworkPolicy) {
	// Add owner references to each store entry
	for i := range snps {
		snps[i].OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion:         "projectcalico.org/v3",
				Kind:               "PolicyRecommendationScope",
				Controller:         &[]bool{true}[0],
				BlockOwnerDeletion: &[]bool{false}[0],
			},
		}
	}

	for _, snp := range snps {
		// Add the object to the cache
		pr.caches.StagedNetworkPolicies.Set(snp.Namespace, &snp)
		// Add the object to the datastore
		_, err := client.StagedNetworkPolicies(snp.Namespace).Create(ctx, &snp, metav1.CreateOptions{})
		Expect(err).To(BeNil())
	}
}
