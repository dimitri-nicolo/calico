// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package recommendation_scope_controller

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	fakecalico "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeK8s "k8s.io/client-go/kubernetes/fake"

	rcache "github.com/projectcalico/calico/kube-controllers/pkg/cache"
	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/api"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	recengine "github.com/projectcalico/calico/policy-recommendation/pkg/engine"
	querymocks "github.com/projectcalico/calico/policy-recommendation/pkg/flows/mocks"
	rectypes "github.com/projectcalico/calico/policy-recommendation/pkg/types"
)

var mt *string

type MockClock struct{}

func (MockClock) NowRFC3339() string { return *mt }

var _ = Describe("RecommendationScopeReconciler", func() {
	const (
		// kindRecommendations is the kind of the recommendations resource.
		kindRecommendations = "recommendations"
	)

	var (
		ctx            context.Context
		buffer         *bytes.Buffer
		engine         *recengine.RecommendationEngine
		logEntry       *log.Entry
		mockNamespaced types.NamespacedName
		mockCtrl       *mockRecommendationScopeController
		r              *recommendationScopeReconciler
	)

	BeforeEach(func() {
		buffer = &bytes.Buffer{}
		// Create a new Logrus logger instance
		logger := log.New()
		// Set the logger's output to the buffer
		logger.SetOutput(buffer)
		// Create a new managed cluster logger entry
		logEntry = logger.WithField("RecommendationScope", "controller")

		ctx = context.TODO()

		mockClientSet := lmak8s.NewMockClientSet(GinkgoT())
		mockClientSet.On("ProjectcalicoV3").Return(fakecalico.NewSimpleClientset().ProjectcalicoV3())
		mockClientSet.On("CoreV1").Return(fakeK8s.NewSimpleClientset().CoreV1())

		_, err := mockClientSet.ProjectcalicoV3().ManagedClusters().Create(ctx, &v3.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "managed-cluster-2",
			},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		mockClientSetFactory := lmak8s.NewMockClientSetFactory(GinkgoT())
		mockClientSetFactory.On("NewClientSetForApplication", "managed-cluster-1").Return(mockClientSet, nil)
		mockClientSetFactory.On("NewClientSetForApplication", "managed-cluster-2").Return(mockClientSet, nil)

		mockLinseedClient := lsclient.NewMockClient("")

		namespaces := []string{"default", "kube-system"}

		mockClock := &MockClock{}

		query := &querymocks.PolicyRecommendationQuery{}
		flows := []*api.Flow{
			{
				Source: api.FlowEndpointData{
					Type:      api.EndpointTypeWep,
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
				Destination: api.FlowEndpointData{
					Type:    api.FlowLogEndpointTypeNetwork,
					Name:    api.FlowLogNetworkPublic,
					Domains: "www.test-domain.com",
					Port:    &[]uint16{80}[0],
				},
				Proto:      &[]uint8{6}[0],
				ActionFlag: api.ActionFlagAllow,
				Reporter:   api.ReporterTypeSource,
			},
		}
		query.On("QueryFlows", mock.Anything).Return(flows, nil)

		// Define the list of items handled by the policy recommendation cache.
		listFunc := func() (map[string]interface{}, error) {
			snps, err := mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(v1.NamespaceAll).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", v3.LabelTier, rectypes.PolicyRecommendationTierName),
			})
			if err != nil {
				return nil, err
			}

			snpMap := make(map[string]interface{})
			for _, snp := range snps.Items {
				snpMap[snp.Namespace] = snp
			}

			return snpMap, nil
		}

		// Create a cache to store recommendations in.
		cacheArgs := rcache.ResourceCacheArgs{
			ListFunc:    listFunc,
			ObjectType:  reflect.TypeOf(v3.StagedNetworkPolicy{}),
			LogTypeDesc: kindRecommendations,
			ReconcilerConfig: rcache.ReconcilerConfig{
				DisableUpdateOnChange: true,
				DisableMissingInCache: true,
			},
		}
		cache := rcache.NewResourceCache(cacheArgs)

		engine = recengine.NewRecommendationEngine(
			ctx,
			"managed-cluster-1",
			mockClientSet.ProjectcalicoV3(),
			mockLinseedClient,
			query,
			cache,
			&v3.PolicyRecommendationScope{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: v3.PolicyRecommendationScopeSpec{
					Interval: &metav1.Duration{Duration: 1 * time.Second},
					InitialLookback: &metav1.Duration{
						Duration: 1 * time.Second,
					},
					StabilizationPeriod: &metav1.Duration{
						Duration: 1 * time.Second,
					},
					NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
						RecStatus: v3.PolicyRecommendationScopeEnabled,
					},
				},
			},
			mockClock,
		)

		for _, ns := range namespaces {
			engine.AddNamespace(ns)
		}

		mockCtrl = newMockRecommendationScopeController(engine)

		r = &recommendationScopeReconciler{
			ctx:       ctx,
			clientSet: mockClientSet,
			linseed:   mockLinseedClient,
			engine:    engine,
			enabled:   v3.PolicyRecommendationScopeDisabled,
			ctrl:      mockCtrl,
			mutex:     sync.Mutex{},
			stopChan:  make(chan struct{}),
			clog:      logEntry,
		}
	})

	Describe("Reconcile", func() {
		Context("When the key name is not PolicyRecommendationScopeName", func() {
			It("should return nil", func() {
				mockNamespaced.Name = "other"
				err := r.Reconcile(mockNamespaced)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("Ignoring PolicyRecommendationScope other"))
			})
		})

		Context("When the key name is PolicyRecommendationScopeName", func() {
			Context("When the incoming scope is enabled and the existing reconciler engine is disabled", func() {
				// thirtySecondInterval is the interval for the recommendation scope.
				const thirtySecondInterval = 30 * time.Second

				BeforeEach(func() {
					mockNamespaced.Name = "default"

				})

				It("should start the controller, and transition the engine status from 'Disabled' to 'Enabled'", func() {
					r.enabled = v3.PolicyRecommendationScopeDisabled
					_, err := r.clientSet.ProjectcalicoV3().PolicyRecommendationScopes().Create(ctx, &v3.PolicyRecommendationScope{
						ObjectMeta: metav1.ObjectMeta{
							Name: "default",
						},
						Spec: v3.PolicyRecommendationScopeSpec{
							Interval: &metav1.Duration{Duration: thirtySecondInterval},
							NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
								RecStatus: v3.PolicyRecommendationScopeEnabled,
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					expected := time.Duration(thirtySecondInterval)

					err = r.Reconcile(mockNamespaced)
					Expect(err).To(BeNil())

					Eventually(func() bool {
						return time.Duration(engine.GetScope().GetInterval().Seconds()) == time.Duration(expected.Seconds())
					}, 10*time.Second).Should(BeTrue())

					Expect(r.enabled).To(Equal(v3.PolicyRecommendationScopeEnabled))
					Eventually(func() bool {
						return mockCtrl.event == "Running"
					}, 10*time.Second).Should(BeTrue())
				})

				It("should handle another update, the engine is enabled and the scope is still enabled", func() {
					// ninetySecondInterval is the interval for the recommendation scope.
					const ninetySecondInterval = 90 * time.Second

					r.enabled = v3.PolicyRecommendationScopeDisabled
					_, err := r.clientSet.ProjectcalicoV3().PolicyRecommendationScopes().Create(ctx, &v3.PolicyRecommendationScope{
						ObjectMeta: metav1.ObjectMeta{
							Name: "default",
						},
						Spec: v3.PolicyRecommendationScopeSpec{
							Interval: &metav1.Duration{Duration: thirtySecondInterval},
							NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
								RecStatus: v3.PolicyRecommendationScopeEnabled,
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					expected := time.Duration(thirtySecondInterval)

					err = r.Reconcile(mockNamespaced)
					Expect(err).To(BeNil())

					Eventually(func() bool {
						return time.Duration(engine.GetScope().GetInterval().Seconds()) == time.Duration(expected.Seconds())
					}, 10*time.Second).Should(BeTrue())

					Expect(r.enabled).To(Equal(v3.PolicyRecommendationScopeEnabled))
					Eventually(func() bool {
						return mockCtrl.event == "Running"
					}, 10*time.Second).Should(BeTrue())

					_, err = r.clientSet.ProjectcalicoV3().PolicyRecommendationScopes().Update(ctx, &v3.PolicyRecommendationScope{
						ObjectMeta: metav1.ObjectMeta{
							Name: "default",
						},
						Spec: v3.PolicyRecommendationScopeSpec{
							Interval: &metav1.Duration{Duration: ninetySecondInterval},
							NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
								RecStatus: v3.PolicyRecommendationScopeEnabled,
							},
						},
					}, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())

					expected = time.Duration(ninetySecondInterval)

					err = r.Reconcile(mockNamespaced)
					Expect(err).To(BeNil())

					Eventually(func() bool {
						return time.Duration(engine.GetScope().GetInterval().Seconds()) == time.Duration(expected.Seconds())
					}, 10*time.Second).Should(BeTrue())

					Expect(r.enabled).To(Equal(v3.PolicyRecommendationScopeEnabled))
					Eventually(func() bool {
						return mockCtrl.event == "Running"
					}, 10*time.Second).Should(BeTrue())
				})
			})

			Context("When the incoming scope is disabled and the existing reconciler engine is enabled", func() {
				BeforeEach(func() {
					mockNamespaced.Name = "default"

				})

				It("update the enabled status and the recommendation scope with a new value", func() {
					r.enabled = v3.PolicyRecommendationScopeEnabled
					_, err := r.clientSet.ProjectcalicoV3().PolicyRecommendationScopes().Create(ctx, &v3.PolicyRecommendationScope{
						ObjectMeta: metav1.ObjectMeta{
							Name: "default",
						},
						Spec: v3.PolicyRecommendationScopeSpec{
							Interval: &metav1.Duration{Duration: 3 * time.Second},
							NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
								RecStatus: v3.PolicyRecommendationScopeDisabled,
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(r.stopChan).NotTo(BeClosed())
					err = r.Reconcile(mockNamespaced)
					Expect(err).To(BeNil())
					Expect(r.enabled).To(Equal(v3.PolicyRecommendationScopeDisabled))
					Expect(r.stopChan).To(BeClosed())
				})

				It("should not update the scope when the engine is already disabled and the scope is disabled", func() {
					r.enabled = v3.PolicyRecommendationScopeEnabled
					_, err := r.clientSet.ProjectcalicoV3().PolicyRecommendationScopes().Create(ctx, &v3.PolicyRecommendationScope{
						ObjectMeta: metav1.ObjectMeta{
							Name: "default",
						},
						Spec: v3.PolicyRecommendationScopeSpec{
							Interval: &metav1.Duration{Duration: 3 * time.Second},
							NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
								RecStatus: v3.PolicyRecommendationScopeDisabled,
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(r.stopChan).NotTo(BeClosed())
					err = r.Reconcile(mockNamespaced)
					Expect(err).To(BeNil())
					Expect(r.enabled).To(Equal(v3.PolicyRecommendationScopeDisabled))
					Expect(r.stopChan).To(BeClosed())

					_, err = r.clientSet.ProjectcalicoV3().PolicyRecommendationScopes().Update(ctx, &v3.PolicyRecommendationScope{
						ObjectMeta: metav1.ObjectMeta{
							Name: "default",
						},
						Spec: v3.PolicyRecommendationScopeSpec{
							Interval: &metav1.Duration{Duration: 9 * time.Second},
							NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
								RecStatus: v3.PolicyRecommendationScopeDisabled,
							},
						},
					}, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(r.stopChan).To(BeClosed())

					err = r.Reconcile(mockNamespaced)
					Expect(err).To(BeNil())
					Expect(r.enabled).To(Equal(v3.PolicyRecommendationScopeDisabled))
					Expect(r.stopChan).To(BeClosed())
				})
			})
		})
	})
})

type mockRecommendationScopeController struct {
	event  string
	engine *recengine.RecommendationEngine
}

func (m *mockRecommendationScopeController) Run(stopCh chan struct{}) {
	m.event = "Running"
	m.engine.Run(stopCh)
}

func newMockRecommendationScopeController(engine *recengine.RecommendationEngine) *mockRecommendationScopeController {
	return &mockRecommendationScopeController{
		event:  "Not Running",
		engine: engine,
	}
}
