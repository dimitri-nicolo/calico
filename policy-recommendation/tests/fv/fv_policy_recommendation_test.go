// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package fv

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeK8s "k8s.io/client-go/kubernetes/fake"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	fakecalico "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"

	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/cache"
	calres "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/engine"
	"github.com/projectcalico/calico/policy-recommendation/pkg/namespace"
	"github.com/projectcalico/calico/policy-recommendation/pkg/policyrecommendation"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
	prtypes "github.com/projectcalico/calico/policy-recommendation/pkg/types"
	fvdata "github.com/projectcalico/calico/policy-recommendation/tests/data"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

var time *string

type MockClock struct{}

func (MockClock) NowRFC3339() string { return *time }

var _ = Describe("Tests policy recommendation controller", func() {
	const (
		clusterID                     = "ClusterID"
		policyRecommendationScopeName = "default"

		timeAtStep1               = "2002-10-02T10:00:00-05:00"
		timeAtStep2               = "2002-10-02T10:02:30-05:00"
		timeAtStep3               = "2002-10-02T10:05:00-05:00"
		timestampStep4Stabilizing = "2002-10-02T10:05:01-05:00"
		timestampStep5Stabilizing = "2002-10-02T10:07:31-05:00"
		timestampStep6Stabilizing = "2002-10-02T10:10:01-05:00"
		timestampStep7Stable      = "2002-10-02T10:15:01-05:00"
		timestampStep8Relearning  = "2002-10-02T11:02:01-05:00"
	)

	type recommendationTest struct {
		timeStep        string
		data            []rest.MockResult
		expected        map[string]*v3.StagedNetworkPolicy
		suffixGenerator func() string
	}

	var (
		ctx context.Context

		err error

		caches            *syncer.CacheSet
		cacheSynchronizer client.QueryInterface
		namespaceCache    *cache.SynchronizedObjectCache[*v1.Namespace]

		fakeClient    *fakecalico.Clientset
		fakeK8sClient *fakeK8s.Clientset

		mockClientSet          *lmak8s.MockClientSet
		mockClientSetFactory   *lmak8s.MockClientSetFactory
		mockClientSetForApp    lmak8s.ClientSet
		mockConstructorTesting mockConstructorTestingTNewMockClientSet
		mockClock              MockClock
		mockLinseedClient      linseed.MockClient

		namespaces []*v1.Namespace

		tier = prtypes.PolicyRecommendationTier
	)

	Context("State if StagedNetworkPolicies after sequential engine calls", func() {
		const serviceSuffixName = "svc.cluster.local"

		var suffixGenerator func() string

		BeforeEach(func() {
			ctx = context.Background()

			fakeClient = fakecalico.NewSimpleClientset()
			fakeK8sClient = fakeK8s.NewSimpleClientset()

			mockClientSetFactory = &lmak8s.MockClientSetFactory{}
			mockConstructorTesting = mockConstructorTestingTNewMockClientSet{}
			mockClientSet = lmak8s.NewMockClientSet(mockConstructorTesting)
			mockLinseedClient = linseed.NewMockClient("")

			mockClientSet.On("ProjectcalicoV3").Return(fakeClient.ProjectcalicoV3())
			mockClientSet.On("CoreV1").Return(fakeK8sClient.CoreV1())
			mockClientSetFactory.On("NewClientSetForApplication", clusterID, mock.Anything).Return(mockClientSet, nil)

			By("creating a policy recommendation scope resource")
			policyRecommendationScopeEnabled := emptyPolicyRecScope
			policyRecommendationScopeEnabled.Spec.NamespaceSpec.RecStatus = v3.PolicyRecommendationScopeEnabled
			Expect(emptyPolicyRecScope.Spec.NamespaceSpec.RecStatus).To(Equal(v3.PolicyRecommendationScopeEnabled))

			// Create mock policy recommendation scope resource
			_, err = mockClientSet.ProjectcalicoV3().PolicyRecommendationScopes().
				Create(ctx, policyRecommendationScopeEnabled, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			mockClientSetForApp, err = mockClientSetFactory.NewClientSetForApplication(clusterID)
			Expect(err).To(BeNil())

			By("defining caches and synchronizer")
			// Setup Caches
			// StagedNetworkPolicy cache
			snpResourceCache := cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()
			// Namespace cache
			namespaceCache = cache.NewSynchronizedObjectCache[*v1.Namespace]()
			// NetworkSets Cache
			networkSetCache := cache.NewSynchronizedObjectCache[*v3.NetworkSet]()
			// Cache set
			caches = &syncer.CacheSet{
				Namespaces:            namespaceCache,
				NetworkSets:           networkSetCache,
				StagedNetworkPolicies: snpResourceCache,
			}
			// Setup cache synchronizer
			suffixGenerator = mockSuffixGenerator
			cacheSynchronizer = syncer.NewCacheSynchronizer(mockClientSetForApp, *caches, suffixGenerator)

			By("creating a list of namespaces")
			// Create namespaces
			namespaces = []*v1.Namespace{tigeraNamespace, namespace1, namespace2, namespace3, namespace4, namespace5}
			for _, ns := range namespaces {
				_, err = fakeK8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
				Expect(err).To(BeNil())
			}

			time = new(string)
		})

		AfterEach(func() {
			mockClientSetFactory.AssertExpectations(GinkgoT())
		})

		It("EgressToDomain 7 engine runs test value, status and lastUpdated", func() {
			// The comparator function verify the timestamp annotation updates of the snp, along with the
			// rule updates after each subsequent call to the engine through RecommendSnp().

			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), fakeK8sClient, mockLinseedClient, cacheSynchronizer, caches, mockClock, serviceSuffixName, &suffixGenerator)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			tr := testRecommendation{
				ctx:           ctx,
				client:        mockClientSet,
				linseedClient: mockLinseedClient,
				caches:        caches,
				clock:         mockClock,
				namespaces:    namespaces,
				reconciler:    prsReconciler.RecommendSnp,
				tier:          tier,
			}

			testCases := []recommendationTest{
				{
					timeStep:        timeAtStep1,
					data:            fvdata.Step1Results,
					expected:        expectedEgressToDomainRecommendationsStep1,
					suffixGenerator: mockSuffixGenerator,
				},
				{
					timeStep:        timeAtStep2,
					data:            fvdata.Step2Results,
					expected:        expectedEgressToDomainRecommendationsStep2,
					suffixGenerator: mockSuffixGenerator,
				},
				{
					timeStep:        timeAtStep3,
					data:            fvdata.Step3Results,
					expected:        expectedEgressToDomainRecommendationsStep3,
					suffixGenerator: mockSuffixGenerator,
				},
				// Test status transition from 'Learning' to 'Stabilizing'
				{
					timeStep:        timestampStep4Stabilizing,
					data:            fvdata.Step3Results,
					expected:        expectedEgressToDomainRecommendationsStep4,
					suffixGenerator: mockSuffixGenerator,
				},
				// Test status transition from 'Learning' to 'Stabilizing'
				{
					timeStep:        timestampStep5Stabilizing,
					data:            fvdata.Step3Results,
					expected:        expectedEgressToDomainRecommendationsStep5,
					suffixGenerator: mockSuffixGenerator,
				},
				// Test status transition from 'Stabilizing' to 'Stable'
				{
					timeStep:        timestampStep6Stabilizing,
					data:            fvdata.Step3Results,
					expected:        expectedEgressToDomainRecommendationsStep6,
					suffixGenerator: mockSuffixGeneratorForStable,
				},
				// Test status transition from 'Stabilizing' to 'Stable'
				{
					timeStep:        timestampStep7Stable,
					data:            fvdata.Step3Results,
					expected:        expectedEgressToDomainRecommendationsStep7,
					suffixGenerator: mockSuffixGeneratorForStable,
				},
				// Test adding PVT to cluster.local domain.
				// The test data adds two suppressed flows, so expected data should stay the same
				{
					timeStep:        timestampStep8Relearning,
					data:            fvdata.Step4DomainWithNamespacesResults,
					expected:        expectedEgressToDomainRecommendationsStep8,
					suffixGenerator: mockSuffixGeneratorForStable,
				},
			}

			By("recommending new egress to domain flows")
			for i, t := range testCases {
				log.Infof("Test iteration: %d", i)

				suffixGenerator = t.suffixGenerator // Update the value so it can be picked up by the reconciler's suffixGenerator

				tr.recommendAtTimestamp(t.timeStep, t.data)
				tr.verifyRecommendations(t.expected)
			}
		})

		It("EgressToService 2 step updates", func() {
			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), fakeK8sClient, mockLinseedClient, cacheSynchronizer, caches, mockClock, serviceSuffixName, &suffixGenerator)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			tr := testRecommendation{
				ctx:           ctx,
				client:        mockClientSet,
				linseedClient: mockLinseedClient,
				caches:        caches,
				clock:         mockClock,
				namespaces:    namespaces,
				reconciler:    prsReconciler.RecommendSnp,
				tier:          tier,
			}

			testCases := []recommendationTest{
				{
					timeStep:        timeAtStep1,
					data:            fvdata.Step4Results,
					expected:        expectedEgressToServiceRecommendationsStep1,
					suffixGenerator: mockSuffixGenerator,
				},
				{
					timeStep:        timeAtStep2,
					data:            fvdata.Step5Results,
					expected:        expectedEgressToServiceRecommendationsStep2,
					suffixGenerator: mockSuffixGenerator,
				},
			}

			By("recommending new egress to service flows")
			for i, t := range testCases {
				log.Infof("Test iteration: %d", i)

				suffixGenerator = t.suffixGenerator // Update the value so it can be picked up by the suffixGenerator

				tr.recommendAtTimestamp(t.timeStep, t.data)
				tr.verifyRecommendations(t.expected)
			}
		})

		It("Namespace 2 step update", func() {
			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), fakeK8sClient, mockLinseedClient, cacheSynchronizer, caches, mockClock, serviceSuffixName, &suffixGenerator)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			tr := testRecommendation{
				ctx:           ctx,
				client:        mockClientSet,
				linseedClient: mockLinseedClient,
				caches:        caches,
				clock:         mockClock,
				namespaces:    namespaces,
				reconciler:    prsReconciler.RecommendSnp,
				tier:          tier,
			}

			testCases := []recommendationTest{
				{
					timeStep:        timeAtStep1,
					data:            fvdata.Step6Results,
					expected:        expectedNamespaceRecommendationsStep1,
					suffixGenerator: mockSuffixGenerator,
				},
			}

			By("recommending new egress to service flows")
			for i, t := range testCases {
				log.Infof("Test iteration: %d", i)

				suffixGenerator = t.suffixGenerator // Update the value so it can be picked up by the suffixGenerator

				tr.recommendAtTimestamp(t.timeStep, t.data)
				tr.verifyRecommendations(t.expected)
			}
		})

		It("NetworkSet", func() {
			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), fakeK8sClient, mockLinseedClient, cacheSynchronizer, caches, mockClock, serviceSuffixName, &suffixGenerator)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			tr := testRecommendation{
				ctx:           ctx,
				client:        mockClientSet,
				linseedClient: mockLinseedClient,
				caches:        caches,
				clock:         mockClock,
				namespaces:    namespaces,
				reconciler:    prsReconciler.RecommendSnp,
				tier:          tier,
			}

			testCases := []recommendationTest{
				{
					timeStep:        timeAtStep1,
					data:            fvdata.NetworkSetLinseedResults,
					expected:        expectedNetworkSetRecommendationsStep1,
					suffixGenerator: mockSuffixGenerator,
				},
				{
					timeStep:        timestampStep4Stabilizing,
					data:            fvdata.NetworkSetLinseedResults,
					expected:        expectedNetworkSetRecommendationsStep2,
					suffixGenerator: mockSuffixGenerator,
				},
				{
					timeStep:        timestampStep7Stable,
					data:            fvdata.NetworkSetLinseedResults,
					expected:        expectedNetworkSetRecommendationsStep3,
					suffixGenerator: mockSuffixGeneratorForStable,
				},
			}

			By("recommending new networkset flows")
			for i, t := range testCases {
				log.Infof("Test iteration: %d", i)

				suffixGenerator = t.suffixGenerator // Update the value so it can be picked up by the suffixGenerator

				tr.recommendAtTimestamp(t.timeStep, t.data)
				tr.verifyRecommendations(t.expected)
			}
		})

		It("PrivateNetwork", func() {
			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), fakeK8sClient, mockLinseedClient, cacheSynchronizer, caches, mockClock, serviceSuffixName, &suffixGenerator)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			tr := testRecommendation{
				ctx:           ctx,
				client:        mockClientSet,
				linseedClient: mockLinseedClient,
				caches:        caches,
				clock:         mockClock,
				namespaces:    namespaces,
				reconciler:    prsReconciler.RecommendSnp,
				tier:          tier,
			}

			testCases := []recommendationTest{
				{
					timeStep:        timeAtStep1,
					data:            fvdata.PrivateNetworkLinseedResults,
					expected:        expectedPrivateNetworkRecommendationsStep1,
					suffixGenerator: mockSuffixGenerator,
				},
				{
					timeStep:        timestampStep4Stabilizing,
					data:            fvdata.PrivateNetworkLinseedResults,
					expected:        expectedPrivateNetworkRecommendationsStep2,
					suffixGenerator: mockSuffixGenerator,
				},
				{
					timeStep:        timestampStep7Stable,
					data:            fvdata.PrivateNetworkLinseedResults,
					expected:        expectedPrivateNetworkRecommendationsStep3,
					suffixGenerator: mockSuffixGeneratorForStable,
				},
			}

			By("recommending new private network flows")
			for i, t := range testCases {
				log.Infof("Test iteration: %d", i)

				suffixGenerator = t.suffixGenerator // Update the value so it can be picked up by the suffixGenerator

				tr.recommendAtTimestamp(t.timeStep, t.data)
				tr.verifyRecommendations(t.expected)
			}
		})
	})

	Context("Deleting namespaces", func() {
		const serviceSuffixName = "svc.cluster.local"

		var suffixGenerator func() string

		BeforeEach(func() {
			*time = ""

			ctx = context.Background()

			fakeClient = fakecalico.NewSimpleClientset()
			fakeK8sClient = fakeK8s.NewSimpleClientset()

			mockClientSetFactory = &lmak8s.MockClientSetFactory{}
			mockConstructorTesting = mockConstructorTestingTNewMockClientSet{}
			mockClientSet = lmak8s.NewMockClientSet(mockConstructorTesting)

			mockClientSet.On("ProjectcalicoV3").Return(fakeClient.ProjectcalicoV3())
			mockClientSet.On("CoreV1").Return(fakeK8sClient.CoreV1())
			mockClientSetFactory.On("NewClientSetForApplication", clusterID, mock.Anything).Return(mockClientSet, nil)

			By("creating a policy recommendation scope resource")
			policyRecommendationScopeEnabled := emptyPolicyRecScope
			policyRecommendationScopeEnabled.Spec.NamespaceSpec.RecStatus = v3.PolicyRecommendationScopeEnabled
			Expect(emptyPolicyRecScope.Spec.NamespaceSpec.RecStatus).To(Equal(v3.PolicyRecommendationScopeEnabled))

			// Create mock policy recommendation scope resource
			_, err = mockClientSet.ProjectcalicoV3().PolicyRecommendationScopes().
				Create(ctx, policyRecommendationScopeEnabled, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			mockClientSetForApp, err = mockClientSetFactory.NewClientSetForApplication(clusterID)
			Expect(err).To(BeNil())

			By("defining caches and synchronizer")
			// Setup Caches
			// StagedNetworkPolicy cache
			snpResourceCache := cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()
			// Namespace cache
			namespaceCache = cache.NewSynchronizedObjectCache[*v1.Namespace]()
			// NetworkSets Cache
			networkSetCache := cache.NewSynchronizedObjectCache[*v3.NetworkSet]()
			// Cache set
			caches = &syncer.CacheSet{
				Namespaces:            namespaceCache,
				NetworkSets:           networkSetCache,
				StagedNetworkPolicies: snpResourceCache,
			}

			suffixGenerator = mockSuffixGenerator

			// Setup cache synchronizer
			cacheSynchronizer = syncer.NewCacheSynchronizer(mockClientSetForApp, *caches, suffixGenerator)

			By("creating a list of namespaces")
			// Create namespaces
			namespaces = []*v1.Namespace{namespace1, namespace2, namespace3, namespace4, namespace5}
			for _, ns := range namespaces {
				_, err = fakeK8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
				Expect(err).To(BeNil())
			}
		})

		AfterEach(func() {
			mockClientSetFactory.AssertExpectations(GinkgoT())
		})

		It("Timestamp update after 2 steps", func() {
			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), fakeK8sClient, mockLinseedClient, cacheSynchronizer, caches, mockClock, serviceSuffixName, &suffixGenerator)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName, Namespace: ""})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			tr := testRecommendation{
				ctx:           ctx,
				client:        mockClientSet,
				linseedClient: mockLinseedClient,
				caches:        caches,
				clock:         mockClock,
				namespaces:    namespaces,
				reconciler:    prsReconciler.RecommendSnp,
				tier:          tier,
			}

			// Step-1
			// - All of the egress to domain rules have the same timestamp
			// - The snp timestamp has been set
			By("recommending new egress to domain flows")
			testCase1 := recommendationTest{
				timeStep:        timeAtStep1,
				data:            fvdata.Step1Results,
				expected:        expectedEgressToDomainRecommendationsStep1,
				suffixGenerator: mockSuffixGenerator,
			}
			suffixGenerator = testCase1.suffixGenerator // Update the value so it can be picked up by the suffixGenerator
			tr.recommendAtTimestamp(testCase1.timeStep, testCase1.data)
			tr.verifyRecommendations(testCase1.expected)

			// Delete and reconcile Namespace3
			By("deleting and reconciling namespace3")
			err = fakeK8sClient.CoreV1().Namespaces().Delete(ctx, namespace3.Name, metav1.DeleteOptions{})
			Expect(err).To(BeNil())
			err = fakeClient.ProjectcalicoV3().StagedNetworkPolicies(namespace3.Name).Delete(ctx, "namespace-isolation.namespace3-xv5fb", metav1.DeleteOptions{})
			Expect(err).To(BeNil())

			err = nsReconciler.Reconcile(types.NamespacedName{Name: namespace3.Name, Namespace: namespace3.Namespace})
			Expect(err).To(BeNil())

			By("verifying the staged network policies after deleting namespace3")
			testCase2 := recommendationTest{
				timeStep: timeAtStep1,
				data:     fvdata.Step1Results,
				expected: expectedEgressToDomainRecommendationsStep1AfterDeletingNamespace3,
			}
			tr.recommendAtTimestamp(testCase2.timeStep, testCase2.data)
			tr.verifyRecommendations(testCase2.expected)
		})
	})

	Context("Enforcing a recommendation", func() {
		const serviceSuffixName = "svc.cluster.local"

		var suffixGenerator func() string

		BeforeEach(func() {
			ctx = context.Background()

			fakeClient = fakecalico.NewSimpleClientset()
			fakeK8sClient = fakeK8s.NewSimpleClientset()

			mockClientSetFactory = &lmak8s.MockClientSetFactory{}
			mockConstructorTesting = mockConstructorTestingTNewMockClientSet{}
			mockClientSet = lmak8s.NewMockClientSet(mockConstructorTesting)
			mockLinseedClient = linseed.NewMockClient("")

			mockClientSet.On("ProjectcalicoV3").Return(fakeClient.ProjectcalicoV3())
			mockClientSet.On("CoreV1").Return(fakeK8sClient.CoreV1())
			mockClientSetFactory.On("NewClientSetForApplication", clusterID, mock.Anything).Return(mockClientSet, nil)

			By("creating a policy recommendation scope resource")
			policyRecommendationScopeEnabled := emptyPolicyRecScope
			policyRecommendationScopeEnabled.Spec.NamespaceSpec.RecStatus = v3.PolicyRecommendationScopeEnabled
			Expect(emptyPolicyRecScope.Spec.NamespaceSpec.RecStatus).To(Equal(v3.PolicyRecommendationScopeEnabled))

			// Create mock policy recommendation scope resource
			_, err = mockClientSet.ProjectcalicoV3().PolicyRecommendationScopes().
				Create(ctx, policyRecommendationScopeEnabled, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			mockClientSetForApp, err = mockClientSetFactory.NewClientSetForApplication(clusterID)
			Expect(err).To(BeNil())

			By("defining caches and synchronizer")
			// Setup Caches
			// StagedNetworkPolicy cache
			snpResourceCache := cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()
			// Namespace cache
			namespaceCache = cache.NewSynchronizedObjectCache[*v1.Namespace]()
			// NetworkSets Cache
			networkSetCache := cache.NewSynchronizedObjectCache[*v3.NetworkSet]()
			// Cache set
			caches = &syncer.CacheSet{
				Namespaces:            namespaceCache,
				NetworkSets:           networkSetCache,
				StagedNetworkPolicies: snpResourceCache,
			}

			suffixGenerator = mockSuffixGenerator

			// Setup cache synchronizer
			cacheSynchronizer = syncer.NewCacheSynchronizer(mockClientSetForApp, *caches, suffixGenerator)

			By("creating a list of namespaces")
			// Create namespaces
			namespaces = []*v1.Namespace{tigeraNamespace, namespace1, namespace2, namespace3, namespace4, namespace5}
			for _, ns := range namespaces {
				_, err = fakeK8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
				Expect(err).To(BeNil())
			}

			time = new(string)
		})

		AfterEach(func() {
			mockClientSetFactory.AssertExpectations(GinkgoT())
		})

		It("Delete staged network policy after enforcement", func() {
			// The comparator function verify the timestamp annotation updates of the snp, along with the
			// rule updates after each subsequent call to the engine through RecommendSnp().

			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), fakeK8sClient, mockLinseedClient, cacheSynchronizer, caches, mockClock, serviceSuffixName, &suffixGenerator)
			err := prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			tr := testRecommendation{
				ctx:           ctx,
				client:        mockClientSet,
				linseedClient: mockLinseedClient,
				caches:        caches,
				clock:         mockClock,
				namespaces:    namespaces,
				reconciler:    prsReconciler.RecommendSnp,
				tier:          tier,
			}

			// Step-1
			By("recommending new egress to domain flows")
			testCase1 := recommendationTest{
				timeStep:        timeAtStep1,
				data:            fvdata.Step1Results,
				expected:        expectedEgressToDomainRecommendationsStep1,
				suffixGenerator: mockSuffixGenerator,
			}
			suffixGenerator = testCase1.suffixGenerator
			tr.recommendAtTimestamp(testCase1.timeStep, testCase1.data)
			tr.verifyRecommendations(testCase1.expected)

			snpName := "namespace-isolation.namespace3-xv5fb"
			ds, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(namespace3.Name).Get(ctx, snpName, metav1.GetOptions{})
			defaultSnp := *ds
			Expect(err).To(BeNil())
			Expect(defaultSnp).NotTo(BeNil())
			// Recommendation should have staged action 'Learn'
			Expect(defaultSnp.Spec.StagedAction).To(Equal(v3.StagedActionLearn))
			Expect(defaultSnp.Labels[calres.StagedActionKey]).To(Equal(string(v3.StagedActionLearn)))
			Expect(len(defaultSnp.OwnerReferences)).To(Equal(1))
			Expect(defaultSnp.OwnerReferences[0].Kind).To(Equal("PolicyRecommendationScope"))
			Expect(defaultSnp.OwnerReferences[0].Name).To(Equal("default"))

			owner := getRecommendationScopeOwner()
			owners := []metav1.OwnerReference{*owner}

			testCases := []struct {
				action              func(*v3.StagedNetworkPolicy)
				dsSA                v3.StagedAction
				csSA                v3.StagedAction
				ownref              []metav1.OwnerReference
				expectedCacheAction v3.StagedAction
				expectedOwnRef      []metav1.OwnerReference
				expectUpdate        bool
			}{
				// Transition learn to active
				{
					action:              activate,
					dsSA:                v3.StagedActionSet,
					csSA:                v3.StagedActionLearn,
					ownref:              nil,
					expectedCacheAction: v3.StagedActionSet,
					expectedOwnRef:      nil,
					expectUpdate:        false,
				},
				// Transition learn to ignore
				{
					action:              ignore,
					dsSA:                v3.StagedActionIgnore,
					csSA:                v3.StagedActionLearn,
					ownref:              owners,
					expectedCacheAction: v3.StagedActionIgnore,
					expectedOwnRef:      owners,
					expectUpdate:        false,
				},
				// Transition active to ignore
				{
					action:              ignore,
					dsSA:                v3.StagedActionIgnore,
					csSA:                v3.StagedActionSet,
					ownref:              owners,
					expectedCacheAction: v3.StagedActionIgnore,
					expectedOwnRef:      owners,
					expectUpdate:        false,
				},
				// Transition ignore to learn
				{
					action:              learn,
					dsSA:                v3.StagedActionLearn,
					csSA:                v3.StagedActionIgnore,
					ownref:              owners,
					expectedCacheAction: v3.StagedActionLearn,
					expectedOwnRef:      owners,
					expectUpdate:        true,
				},
			}

			defaultCacheSnp := *caches.StagedNetworkPolicies.Get(namespace3.Name)

			for i, test := range testCases {
				log.Infof("Test iteration: %d", i)

				// Re-set the datastore snp
				snp := defaultSnp
				startingCache := defaultCacheSnp
				caches.StagedNetworkPolicies.Set(snpName, &startingCache)

				// Set the snp cache state
				cacheSnp := caches.StagedNetworkPolicies.Get(snpName)
				cacheSnp.Spec.StagedAction = test.csSA
				cacheSnp.Labels[calres.StagedActionKey] = string(test.csSA)
				Expect(cacheSnp.Spec.StagedAction).To(Equal(test.csSA))
				Expect(cacheSnp.Labels[calres.StagedActionKey]).To(Equal(string(test.csSA)))

				// Transition the datastore snp
				test.action(&snp)
				updatedSnp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(namespace3.Name).Update(ctx, &snp, metav1.UpdateOptions{})
				Expect(err).To(BeNil())
				Expect(updatedSnp.Spec.StagedAction).To(Equal(test.dsSA))
				Expect(updatedSnp.Labels[calres.StagedActionKey]).To(Equal(string(test.dsSA)))
				Expect(len(updatedSnp.OwnerReferences)).To(Equal(len(test.ownref)))
				if len(updatedSnp.OwnerReferences) == 1 && len(test.ownref) == 1 {
					Expect(reflect.DeepEqual(snp.OwnerReferences, test.ownref)).To(BeTrue())
				}

				dataStoreBefore, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(namespace3.Name).Get(ctx, snpName, metav1.GetOptions{})
				Expect(err).To(BeNil())
				Expect(dataStoreBefore).NotTo(BeNil())

				// Append a rule to the cache egress rules
				cacheSnp.Spec.Egress = append(cacheSnp.Spec.Egress, v3.Rule{Action: v3.Allow, Protocol: &protocolTCP})

				// Call RecommendSnp, expecting to delete the snp from the
				mockLinseedClient.SetResults(fvdata.Step1Results...)
				prsReconciler.RecommendSnp(ctx, mockClock, cacheSnp)

				// The cache should be in the expected state
				cacheSnp = caches.StagedNetworkPolicies.Get(snpName)
				Expect(cacheSnp.Spec.StagedAction).To(Equal(test.expectedCacheAction))
				Expect(cacheSnp.Labels[calres.StagedActionKey]).To(Equal(string(test.expectedCacheAction)))

				// Verify that updates occur only when the cache is transition to learn
				dataStoreAfter, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(namespace3.Name).Get(ctx, snpName, metav1.GetOptions{})
				Expect(err).To(BeNil())
				Expect(dataStoreAfter).NotTo(BeNil())

				if test.expectUpdate {
					Expect(len(dataStoreAfter.Spec.Egress) > len(dataStoreBefore.Spec.Egress)).To(BeTrue())
				} else {
					Expect(dataStoreAfter).To(Equal(dataStoreBefore))
				}
			}
		})
	})
})

// activate sets staged action and owner reference metadata of a staged network policy to an
// active state.
func activate(snp *v3.StagedNetworkPolicy) {
	snp.Spec.StagedAction = v3.StagedActionSet
	snp.Labels[calres.StagedActionKey] = string(v3.StagedActionSet)
	snp.OwnerReferences = nil
}

// ignore sets staged action and owner reference metadata of a staged network policy to an
// ignore state.
func ignore(snp *v3.StagedNetworkPolicy) {
	snp.Spec.StagedAction = v3.StagedActionIgnore
	snp.Labels[calres.StagedActionKey] = string(v3.StagedActionIgnore)
}

// learn sets staged action and owner reference metadata of a staged network policy to a
// learn state.
func learn(snp *v3.StagedNetworkPolicy) {
	snp.Spec.StagedAction = v3.StagedActionLearn
	snp.Labels[calres.StagedActionKey] = string(v3.StagedActionLearn)
}

// getRecommendationScopeOwner returns policy recommendation scope resource as an owner reference
// resource.
func getRecommendationScopeOwner() *metav1.OwnerReference {
	ctrl := true
	blockOwnerDelete := false

	return &metav1.OwnerReference{
		APIVersion:         "projectcalico.org/v3",
		Kind:               "PolicyRecommendationScope",
		Name:               "default",
		UID:                "",
		Controller:         &ctrl,
		BlockOwnerDeletion: &blockOwnerDelete,
	}
}

// compareSnps is a helper function used to determine equality between two staged network policies.
func compareSnps(left, right *v3.StagedNetworkPolicy) bool {
	log.Infof("comparing left and right rules: %s - %s", left.Name, right.Name)
	Expect(left.ObjectMeta.Name).To(Equal(right.ObjectMeta.Name), "Snp name: '%s' is not the same as '%s'", left.ObjectMeta.Name, right.ObjectMeta.Name)
	Expect(left.ObjectMeta.Namespace).To(Equal(right.ObjectMeta.Namespace), "Snp namespace: '%s' is not the same as '%s'", left.ObjectMeta.Namespace, right.ObjectMeta.Namespace)
	Expect(left.ObjectMeta.Labels).To(Equal(right.ObjectMeta.Labels))
	Expect(left.ObjectMeta.Annotations[calres.LastUpdatedKey]).To(Equal(right.ObjectMeta.Annotations[calres.LastUpdatedKey]))
	Expect(left.ObjectMeta.Annotations[calres.StatusKey]).To(Equal(right.ObjectMeta.Annotations[calres.StatusKey]))

	Expect(len(left.OwnerReferences)).To(Equal(len(right.OwnerReferences)), "The length of owner references should be equal for snp: %s", left.ObjectMeta.Name)
	for i := 0; i < len(left.OwnerReferences); i++ {
		Expect(left.OwnerReferences[i].APIVersion).To(Equal(right.OwnerReferences[i].APIVersion))
		Expect(left.OwnerReferences[i].Kind).To(Equal(right.OwnerReferences[i].Kind))
		Expect(left.OwnerReferences[i].Name).To(Equal(right.OwnerReferences[i].Name))
	}

	Expect(left.Spec.StagedAction).To(Equal(right.Spec.StagedAction))
	Expect(left.Spec.Tier).To(Equal(right.Spec.Tier))
	Expect(left.Spec.Order).To(Equal(right.Spec.Order))
	Expect(left.Spec.Selector).To(Equal(right.Spec.Selector))
	Expect(left.Spec.Types).To(Equal(right.Spec.Types))

	length := len(left.Spec.Egress)
	Expect(length).To(Equal(len(right.Spec.Egress)))
	for i := 0; i < length; i++ {
		compareRules(&left.Spec.Egress[i], &right.Spec.Egress[i])
	}

	length = len(left.Spec.Ingress)
	Expect(length).To(Equal(len(right.Spec.Ingress)))
	for i := 0; i < length; i++ {
		compareRules(&left.Spec.Ingress[i], &right.Spec.Ingress[i])
	}

	return true
}

// compareRules is a helper function used to compare the policy recommendation relevant parameters
// between two v3 rules.
func compareRules(left, right *v3.Rule) bool {
	Expect(left.Metadata.Annotations[calres.ScopeKey]).To(Equal(right.Metadata.Annotations[calres.ScopeKey]), "%v should equal\n %v", left.Metadata.Annotations, right.Metadata.Annotations)
	Expect(left.Metadata.Annotations[calres.LastUpdatedKey]).To(Equal(right.Metadata.Annotations[calres.LastUpdatedKey]), "%v should equal\n %v", left.Metadata.Annotations, right.Metadata.Annotations)
	Expect(left.Metadata.Annotations[calres.NamespaceKey]).To(Equal(right.Metadata.Annotations[calres.NamespaceKey]), "%v should equal\n %v", left.Metadata.Annotations, right.Metadata.Annotations)
	Expect(left.Metadata.Annotations[calres.NameKey]).To(Equal(right.Metadata.Annotations[calres.NameKey]), "%v should equal\n %v", left.Metadata.Annotations, right.Metadata.Annotations)
	Expect(left.Action).To(Equal(right.Action))
	Expect(left.Protocol).To(Equal(right.Protocol))
	Expect(reflect.DeepEqual(left.Destination.Services, right.Destination.Services)).To(BeTrue())
	Expect(reflect.DeepEqual(left.Destination.Domains, right.Destination.Domains)).To(BeTrue())
	Expect(reflect.DeepEqual(left.Destination.Ports, right.Destination.Ports)).To(BeTrue())

	return true
}

type testRecommendation struct {
	ctx           context.Context
	client        *lmak8s.MockClientSet
	linseedClient linseed.MockClient
	caches        *syncer.CacheSet
	clock         MockClock
	namespaces    []*v1.Namespace
	tier          string
	reconciler    func(context.Context, engine.Clock, *v3.StagedNetworkPolicy)
}

func (t *testRecommendation) recommendAtTimestamp(
	timestamp string,
	results []rest.MockResult,
) {
	// Update the mockClock RFC3339() return value
	*time = timestamp

	// Run the engine to update the snps
	snps := t.caches.StagedNetworkPolicies.GetAll()
	for _, snp := range snps {
		t.linseedClient.SetResults(results...)
		t.reconciler(t.ctx, t.clock, snp)
	}
}

func (t *testRecommendation) verifyRecommendations(expectedRecommendation map[string]*v3.StagedNetworkPolicy) {
	recs, _ := t.client.ProjectcalicoV3().StagedNetworkPolicies("").List(t.ctx, metav1.ListOptions{})
	Expect(len(recs.Items) == len(expectedRecommendation)).To(BeTrue())

	for key, expected := range expectedRecommendation {
		snp, err := t.client.ProjectcalicoV3().StagedNetworkPolicies(expected.Namespace).
			Get(t.ctx, key, metav1.GetOptions{})

		Expect(err).To(BeNil())
		Expect(compareSnps(snp, expected)).To(BeTrue())
	}
}

var (
	tigeraNamespace = &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tigera-value",
		},
	}

	namespace1 = &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace1",
		},
	}
	namespace2 = &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace2",
		},
	}
	namespace3 = &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace3",
		},
	}
	namespace4 = &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace4",
		},
	}
	namespace5 = &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace5",
		},
	}

	emptyPolicyRecScope = &v3.PolicyRecommendationScope{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: v3.PolicyRecommendationScopeSpec{
			NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
				RecStatus: v3.PolicyRecommendationScopeDisabled,
				Selector: "!(projectcalico.org/name starts with 'tigera-') && !(projectcalico.org/name " +
					"starts with 'calico-') && !(projectcalico.org/name starts with 'kube-')",
			},
		},
	}
)

// Mock for testing.
type mockConstructorTestingTNewMockClientSet struct {
}

func (m mockConstructorTestingTNewMockClientSet) Cleanup(func()) {
}

func (m mockConstructorTestingTNewMockClientSet) Logf(format string, args ...interface{}) {
}

func (m mockConstructorTestingTNewMockClientSet) Errorf(format string, args ...interface{}) {
}

func (m mockConstructorTestingTNewMockClientSet) FailNow() {
}

func mockSuffixGenerator() string {
	return "xv5fb"
}

func mockSuffixGeneratorForStable() string {
	return "76kle"
}
