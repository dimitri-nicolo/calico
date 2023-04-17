// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package fv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeK8s "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	fakecalico "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/cache"
	calres "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	"github.com/projectcalico/calico/policy-recommendation/pkg/namespace"
	"github.com/projectcalico/calico/policy-recommendation/pkg/policyrecommendation"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

const (
	clusterID                     = "ClusterID"
	policyRecommendationScopeName = "default"

	timestampStep4Stabilizing = "2002-10-02T10:05:01-05:00"
	timestampStep5Stabilizing = "2002-10-02T10:07:31-05:00"
	timestampStep6Stabilizing = "2002-10-02T10:10:01-05:00"
	timestampStep7Stabilizing = "2002-10-02T10:15:01-05:00"
)

var time *string

type mockClock struct{}

func (mockClock) NowRFC3339() string { return *time }

var _ = Describe("Tests policy recommendation controller", func() {
	var (
		ctx context.Context

		err error

		caches            *syncer.CacheSet
		cacheSynchronizer client.QueryInterface
		namespaceCache    *cache.SynchronizedObjectCache[*v1.Namespace]

		fakeClient *fakecalico.Clientset
		fakeCoreV1 corev1.CoreV1Interface

		mockClientSet          *lmak8s.MockClientSet
		mockClientSetFactory   *lmak8s.MockClientSetFactory
		mockClientSetForApp    lmak8s.ClientSet
		mockConstructorTesting mockConstructorTestingTNewMockClientSet
		mockEsClient           lmaelastic.Client
		mockClock              mockClock

		namespaces []*v1.Namespace

		tier string
	)

	Context("State if StagedNetworkPolicies after sequential engine calls", func() {
		BeforeEach(func() {
			ctx = context.Background()

			fakeClient = fakecalico.NewSimpleClientset()
			fakeCoreV1 = fakeK8s.NewSimpleClientset().CoreV1()

			mockClientSetFactory = &lmak8s.MockClientSetFactory{}
			mockConstructorTesting = mockConstructorTestingTNewMockClientSet{}
			mockClientSet = lmak8s.NewMockClientSet(mockConstructorTesting)

			mockClientSet.On("ProjectcalicoV3").Return(fakeClient.ProjectcalicoV3())
			mockClientSet.On("CoreV1").Return(fakeCoreV1)
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
			cacheSynchronizer = syncer.NewCacheSynchronizer(mockClientSetForApp, *caches)

			tier = "namespace-segmentation"

			By("creating a list of namespaces")
			// Create namespaces
			namespaces = []*v1.Namespace{tigeraNamespace, namespace1, namespace2, namespace3, namespace4, namespace5}
			for _, ns := range namespaces {
				_, err = fakeCoreV1.Namespaces().Create(ctx, ns, metav1.CreateOptions{})
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
				mockClientSet.ProjectcalicoV3(), &mockEsClient, cacheSynchronizer, caches, mockClock)
			err := prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			// Step-1
			By("recommending new egress to domain flows")
			// Update the mockClock RFC3339() return value
			*time = timeAtStep1

			step1File := "../data/es_flows_sample_egress_domain1.json"
			// Run the engine to update the snps
			snps := caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step1File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				*time = "2002-10-02T10:00:00-05:00"
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			By("verifying the staged network policies for step-1")
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep1[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}

			// Step-2
			By("recommending new egress to domain flows")
			// Update the mockClock RFC3339() return value
			*time = timeAtStep2

			step2File := "../data/es_flows_sample_egress_domain2.json"
			// Run the engine to update the snps
			snps = caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step2File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			By("verifying the staged network policies for step-2")
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep2[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}

			// Step-3
			By("recommending new egress to domain flows")
			// Update the mockClock RFC3339() return value
			*time = timeAtStep3

			step3File := "../data/es_flows_sample_egress_domain3.json"
			// Run the engine to update the snps
			snps = caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step3File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			By("verifying the staged network policies for step-3")
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep3[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}

			// Step-4
			// Test status transition from 'Learning' to 'Stabilizing'

			// Update the mockClock RFC3339() return value
			*time = timestampStep4Stabilizing

			// Run the engine to update the snps
			snps = caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step3File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			By("verifying the staged network policies for step-4")
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep4[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}

			// Step-5
			// Test status transition from 'Learning' to 'Stabilizing'

			// Update the mockClock RFC3339() return value
			*time = timestampStep5Stabilizing

			// Run the engine to update the snps
			snps = caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step3File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			By("verifying the staged network policies for step-5")
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep5[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}

			// Step-6
			// Test status transition from 'Stabilizing' to 'Stable'

			// Update the mockClock RFC3339() return value
			*time = timestampStep6Stabilizing

			// Run the engine to update the snps
			snps = caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step3File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			By("verifying the staged network policies for step-6")
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep6[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}

			// Step-7
			// Test status transition from 'Stabilizing' to 'Stable'

			// Update the mockClock RFC3339() return value
			*time = timestampStep7Stabilizing

			// Run the engine to update the snps
			snps = caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step3File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			By("verifying the staged network policies for step-7")
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep7[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}
		})

		It("EgressToService 2 step updates", func() {
			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), &mockEsClient, cacheSynchronizer, caches, mockClock)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			// Step-1
			By("recommending new egress to service flows - step-1")
			// Update the mockClock RFC3339() return value
			*time = timeAtStep1

			step1File := "../data/es_flows_sample_egress_service1.json"
			// Run the engine to update the snps
			snps := caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step1File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToServiceRecommendationsStep1[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}

			// Step-2
			By("recommending new egress to service flows - step-2")
			// Update the mockClock RFC3339() return value
			*time = timeAtStep2

			step2File := "../data/es_flows_sample_egress_service2.json"
			// Run the engine to update the snps
			snps = caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step2File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToServiceRecommendationsStep2[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}
		})

		It("Namespace 2 step update", func() {
			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), &mockEsClient, cacheSynchronizer, caches, mockClock)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			// Step-1
			By("recommending new egress to service flows")
			// Update the mockClock RFC3339() return value
			*time = timeAtStep1

			step1File := "../data/es_flows_sample_namespace1.json"
			// Run the engine to update the snps
			snps := caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step1File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedNamespaceRecommendationsStep1[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}
		})

		It("PublicNetwork", func() {
		})
	})

	Context("Deleting namespaces", func() {
		BeforeEach(func() {
			ctx = context.Background()

			fakeClient = fakecalico.NewSimpleClientset()
			fakeCoreV1 = fakeK8s.NewSimpleClientset().CoreV1()

			mockClientSetFactory = &lmak8s.MockClientSetFactory{}
			mockConstructorTesting = mockConstructorTestingTNewMockClientSet{}
			mockClientSet = lmak8s.NewMockClientSet(mockConstructorTesting)

			mockClientSet.On("ProjectcalicoV3").Return(fakeClient.ProjectcalicoV3())
			mockClientSet.On("CoreV1").Return(fakeCoreV1)
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
			cacheSynchronizer = syncer.NewCacheSynchronizer(mockClientSetForApp, *caches)

			tier = "namespace-segmentation"

			By("creating a list of namespaces")
			// Create namespaces
			namespaces = []*v1.Namespace{namespace1, namespace2, namespace3, namespace4, namespace5}
			for _, ns := range namespaces {
				_, err = fakeCoreV1.Namespaces().Create(ctx, ns, metav1.CreateOptions{})
				Expect(err).To(BeNil())
			}
		})

		AfterEach(func() {
			mockClientSetFactory.AssertExpectations(GinkgoT())
		})

		It("Timestamp update after 2 steps", func() {
			// Reconcile policy recommendation scope
			prsReconciler := policyrecommendation.NewPolicyRecommendationReconciler(
				mockClientSet.ProjectcalicoV3(), &mockEsClient, cacheSynchronizer, caches, mockClock)
			err = prsReconciler.Reconcile(types.NamespacedName{Name: policyRecommendationScopeName, Namespace: ""})
			Expect(err).To(BeNil())

			// Reconcile namespaces
			nsReconciler := namespace.NewNamespaceReconciler(mockClientSet, namespaceCache, cacheSynchronizer)
			for _, ns := range namespaces {
				err = nsReconciler.Reconcile(types.NamespacedName{Name: ns.Name, Namespace: ns.Namespace})
				Expect(err).To(BeNil())
			}

			// Step-1
			// - All of the egress to domain rules have the same timestamp
			// - The snp timestamp has been set
			By("recommending new egress to domain flows")
			// Update the mockClock RFC3339() return value
			*time = timeAtStep1

			step1File := "../data/es_flows_sample_egress_domain1.json"
			// Run the engine to update the snps
			snps := caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step1File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			By("verifying the staged network policies for step-1")
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep1[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(snp.Annotations[calres.LastUpdatedKey]).To(Equal(expectedSnp.Annotations[calres.LastUpdatedKey]))
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}

			// Delete and reconcile Namespace3
			By("deleting and reconciling namespace3")
			err = fakeCoreV1.Namespaces().Delete(ctx, namespace3.Name, metav1.DeleteOptions{})
			Expect(err).To(BeNil())
			err = nsReconciler.Reconcile(types.NamespacedName{Name: namespace3.Name, Namespace: namespace3.Namespace})
			Expect(err).To(BeNil())

			By("verifying the staged network policies after deleting namespace3")
			// Run the engine to update the snps
			snps = caches.StagedNetworkPolicies.GetAll()
			for _, snp := range snps {
				mockEsResponse := getMockEsResponse(step1File)
				mockEsClient = lmaelastic.NewMockSearchClient([]interface{}{mockEsResponse})
				prsReconciler.RecommendSnp(ctx, mockClock, snp)
			}
			for _, ns := range namespaces {
				expectedNamespace := ns.Name
				expectedSnpName := fmt.Sprintf("%s.%s-%s", tier, ns.Name, calres.PolicRecSnpNameSuffix)
				snp, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(expectedNamespace).
					Get(ctx, expectedSnpName, metav1.GetOptions{})

				if expectedSnp, ok := expectedEgressToDomainRecommendationsStep1AfterDeletingNamespace3[expectedSnpName]; ok {
					Expect(err).To(BeNil())
					Expect(compareSnps(snp, expectedSnp)).To(BeTrue())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}
		})
	})
})

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
	Expect(len(left.Spec.Egress)).To(Equal(len(right.Spec.Egress)))
	length := len(left.Spec.Egress)

	for i := 0; i < length; i++ {
		compareRules(&left.Spec.Egress[i], &right.Spec.Egress[i])
	}
	length = len(left.Spec.Ingress)
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

// readJsonFile reads in a file, given a fname and returns a json file as an array of bytes. Returns
// an error if ReadFile returns an error.
func readJsonFile(fname string) ([]byte, error) {
	// Read the file
	file, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// getMockEsResponse given a file path, returns a mock elastic response as a string.
func getMockEsResponse(file string) string {
	Expect(file).NotTo(BeEmpty())

	// Read json file containing a sample es response
	data, err := readJsonFile(file)
	Expect(err).To(BeNil())
	var esResponseMap map[string]interface{}
	err = json.Unmarshal(data, &esResponseMap)
	Expect(err).To(BeNil())

	var esResponseBuckets bytes.Buffer
	for _, val := range esResponseMap {
		jsonVal, err := json.MarshalIndent(val, "", "  ")
		Expect(err).To(BeNil())
		esResponseBuckets.WriteString(fmt.Sprintf(esBucket, string(jsonVal)))
	}
	// The response buckets contains a trailing comma, which must be removed
	buckets := esResponseBuckets.String()
	buckets = buckets[:len(buckets)-2]

	response := fmt.Sprintf(esResp, len(esResponseMap), buckets)

	return response
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

	esResp = `{
  "took": 78,
  "timed_out": false,
  "_shards": {
    "total": 40,
    "successful": 40,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": {
      "relation": "eq",
      "value": %d
    },
    "max_score": 0,
    "hits": []
  },
  "aggregations": {
    "flog_buckets": {
      "buckets": [
        %s
      ]
    }
  }
}
`

	esBucket = `{
  "key": %s,
  "doc_count": 14,
  "sum_bytes_out": {
    "value": 5
  },
  "policies": {
    "doc_count": 4,
    "by_tiered_policy": {
      "doc_count_error_upper_bound": 0,
      "sum_other_doc_count": 0,
      "buckets": [
        {
          "key": "0|allow-tigera|__/allow-tigera.cluster-dns|pass|1",
          "doc_count": 5
        }
      ]
    }
  },
  "source_labels": {
    "doc_count": 60,
    "by_kvpair": {
      "doc_count_error_upper_bound": 0,
      "sum_other_doc_count": 0,
      "buckets": [
        {
          "key": "aaaa",
          "doc_count": 70
        },
        {
          "key": "bbbb",
          "doc_count": 80
        },
        {
          "key": 1,
          "doc_count": 1
        }
      ]
    }
  },
  "dest_labels": {
    "doc_count": 21,
    "by_kvpair": {
      "doc_count_error_upper_bound": 0,
      "sum_other_doc_count": 0,
      "buckets": [
        {
          "key": 123,
          "doc_count": 22
        },
        {
          "key": 124,
          "doc_count": 23
        }
      ]
    }
  }
},
`
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
