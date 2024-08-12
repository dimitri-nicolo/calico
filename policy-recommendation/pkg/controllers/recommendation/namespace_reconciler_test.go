// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package recommendation_controller

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeK8s "k8s.io/client-go/kubernetes/fake"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	fakecalico "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"

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

var _ = Describe("NamespaceReconciler", func() {
	const (
		// kindRecommendations is the kind of the recommendations resource.
		kindRecommendations = "recommendations"

		// retryInterval is the interval between retries.
		retryInterval = time.Second * 2
	)

	var (
		key           types.NamespacedName
		mockClientSet *lmak8s.MockClientSet
		r             *namespaceReconciler
	)

	BeforeEach(func() {
		ctx := context.TODO()
		buffer := &bytes.Buffer{}
		// Create a new Logrus logger instance
		logger := log.New()
		// Set the logger's output to the buffer
		logger.SetOutput(buffer)
		// Create a new managed cluster logger entry
		logEntry := logger.WithField("ManagedCluster", "controller")

		mockClientSet = lmak8s.NewMockClientSet(GinkgoT())
		mockClientSet.On("ProjectcalicoV3").Return(fakecalico.NewSimpleClientset().ProjectcalicoV3())
		mockClientSet.On("CoreV1").Return(fakeK8s.NewSimpleClientset().CoreV1())

		// Get the list of recommendations from the datastore with retries.
		listRecommendations := func(ret int) ([]v3.StagedNetworkPolicy, error) {
			var err error
			var snps *v3.StagedNetworkPolicyList
			for i := 0; i < ret; i++ {
				snps, err = mockClientSet.ProjectcalicoV3().StagedNetworkPolicies(v1.NamespaceAll).List(ctx, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s", v3.LabelTier, rectypes.PolicyRecommendationTierName),
				})
				if err == nil {
					break
				}
				time.Sleep(retryInterval) // Wait before retrying
			}

			if err != nil {
				return nil, err
			}

			return snps.Items, nil
		}
		// Define the list of items handled by the policy recommendation cache.
		listFunc := func() (map[string]interface{}, error) {
			snps, err := listRecommendations(retries)
			if err != nil {
				return nil, err
			}

			snpMap := make(map[string]interface{})
			for _, snp := range snps {
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

		mockLinseedClient := lsclient.NewMockClient("")

		namespaces := []string{"default"}

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

		engine := recengine.NewRecommendationEngine(
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
					Interval: &metav1.Duration{
						Duration: time.Second * 2,
					},
					NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
						Selector: "!(projectcalico.org/name starts with 'tigera-') && !(projectcalico.org/name starts with 'calico-') && " +
							"!(projectcalico.org/name starts with 'kube-') && !(projectcalico.org/name starts with 'openshift-')",
					},
				},
			},
			mockClock,
		)

		for _, ns := range namespaces {
			engine.AddNamespace(ns)
		}

		// Create a new namespaceReconciler instance with the fake clientSet
		r = &namespaceReconciler{
			clientSet: mockClientSet,
			ctx:       ctx,
			engine:    engine,
			clog:      logEntry,
			cache:     cache,
		}
	})

	Context("When the namespace is created", func() {
		It("should add the namespace to the engine for processing if the selector is validated", func() {
			_, err := mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			key = types.NamespacedName{
				Name:      "test-namespace",
				Namespace: "test-namespace",
			}

			err = r.Reconcile(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.engine.GetNamespaces().Contains(key.Name)).To(BeTrue())
			Expect(r.engine.GetFilteredNamespaces().Contains(key.Name)).To(BeTrue())
		})

		It("should not add the namespace to the engine for processing if the selector is not validated", func() {
			_, err := mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tigera-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			key = types.NamespacedName{
				Name:      "tigera-namespace",
				Namespace: "tigera-namespace",
			}

			err = r.Reconcile(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.engine.GetNamespaces().Contains(key.Name)).To(BeTrue())
			Expect(r.engine.GetFilteredNamespaces().Contains(key.Name)).To(BeFalse())
		})

		It("should add multiple namespaces to the engine for processing if the selector is validated", func() {
			_, err := mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace-2",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			key1 := types.NamespacedName{
				Name:      "test-namespace-2",
				Namespace: "test-namespace-2",
			}

			err = r.Reconcile(key1)
			Expect(err).ToNot(HaveOccurred())

			_, err = mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace-3",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			key2 := types.NamespacedName{
				Name:      "test-namespace-3",
				Namespace: "test-namespace-3",
			}

			err = r.Reconcile(key2)
			Expect(err).ToNot(HaveOccurred())

			Expect(r.engine.GetNamespaces().Contains(key1.Name)).To(BeTrue())
			Expect(r.engine.GetNamespaces().Contains(key2.Name)).To(BeTrue())

			Expect(r.engine.GetFilteredNamespaces().Contains(key1.Name)).To(BeTrue())
			Expect(r.engine.GetFilteredNamespaces().Contains(key2.Name)).To(BeTrue())
		})

		It("should keep track of every namespace even if the selector is not validated, but not add it to the filtered items for processing", func() {
			// We test against the default selector, which excludes namespaces starting with "calico-",
			// "kube-", "tigera-", and  added "openshift-".

			// Try to create a namespace with a name starting with "calico-".
			_, err := mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "calico-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			key = types.NamespacedName{
				Name:      "calico-namespace",
				Namespace: "calico-namespace",
			}

			err = r.Reconcile(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.engine.GetNamespaces().Contains(key.Name)).To(BeTrue())

			// Try to create a namespace with a name starting with "kube-".
			_, err = mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			key = types.NamespacedName{
				Name:      "kube-namespace",
				Namespace: "kube-namespace",
			}

			err = r.Reconcile(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.engine.GetNamespaces().Contains(key.Name)).To(BeTrue())
			Expect(r.engine.GetFilteredNamespaces().Contains(key.Name)).To(BeFalse())

			// Try to create a namespace with a name starting with "tigera-".
			_, err = mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tigera-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			key = types.NamespacedName{
				Name:      "tigera-namespace",
				Namespace: "tigera-namespace",
			}

			err = r.Reconcile(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.engine.GetNamespaces().Contains(key.Name)).To(BeTrue())
			Expect(r.engine.GetFilteredNamespaces().Contains(key.Name)).To(BeFalse())

			// Try to create a namespace with a name starting with "openshift-".
			_, err = mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "openshift-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			key = types.NamespacedName{
				Name:      "openshift-namespace",
				Namespace: "openshift-namespace",
			}

			err = r.Reconcile(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.engine.GetNamespaces().Contains(key.Name)).To(BeTrue())
			Expect(r.engine.GetFilteredNamespaces().Contains(key.Name)).To(BeFalse())
		})
	})

	Context("When the namespace is deleted", func() {
		BeforeEach(func() {
			// Create a namespace to delete
			_, err := mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-delete-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should remove namespaces from the engine's processing items and the cache", func() {
			r.engine.GetNamespaces().Add("test-keep-namespace")
			r.engine.GetFilteredNamespaces().Add("test-keep-namespace")
			r.cache.Set("test-keep-namespace", v3.StagedNetworkPolicy{})
			_, err := mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-keep-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Add two namespaces to delete, and attempt to do so concurrently. test-delete-namespace has
			// already been added to the store in BeforeEach.
			r.engine.GetNamespaces().Add("test-delete-namespace")
			r.engine.GetFilteredNamespaces().Add("test-delete-namespace")
			r.cache.Set("test-delete-namespace", v3.StagedNetworkPolicy{})

			r.engine.GetNamespaces().Add("test-delete-namespace2")
			r.engine.GetFilteredNamespaces().Add("test-delete-namespace2")
			r.cache.Set("test-delete-namespace2", v3.StagedNetworkPolicy{})
			_, err = mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-delete-namespace2",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			go func() {
				err := mockClientSet.CoreV1().Namespaces().Delete(context.TODO(), "test-delete-namespace", metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = r.Reconcile(types.NamespacedName{
					Name:      "test-delete-namespace",
					Namespace: "test-delete-namespace",
				})
				Expect(err).ToNot(HaveOccurred())
			}()

			go func() {
				err := mockClientSet.CoreV1().Namespaces().Delete(context.TODO(), "test-delete-namespace2", metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = r.Reconcile(types.NamespacedName{
					Name:      "test-delete-namespace2",
					Namespace: "test-delete-namespace2",
				})
				Expect(err).ToNot(HaveOccurred())
			}()

			Eventually(func() bool {
				v, ok := r.cache.Get("test-keep-namespace")
				if ok && v != nil && r.engine.GetNamespaces().Contains("test-keep-namespace") && r.engine.GetFilteredNamespaces().Contains("test-keep-namespace") {
					return true
				}
				return false
			}).Should(BeTrue())
			Eventually(func() bool {
				_, ok := r.cache.Get("test-delete-namespace")
				if !ok && !r.engine.GetNamespaces().Contains("test-delete-namespace") && !r.engine.GetFilteredNamespaces().Contains("test-delete-namespace") {
					return true
				}
				return false
			}).Should(BeTrue())
			Eventually(func() bool {
				_, ok := r.cache.Get("test-delete-namespace2")
				if !ok && !r.engine.GetNamespaces().Contains("test-delete-namespace2") && !r.engine.GetFilteredNamespaces().Contains("test-delete-namespace2") {
					return true
				}
				return false
			}).Should(BeTrue())
		})

		It("should handle concurrent additions and deletions of namespaces to/from the engine's processing items and the cache", func() {
			r.engine.GetNamespaces().Add("test-remove-namespace")
			Expect(r.engine.GetNamespaces().Contains("test-remove-namespace")).To(BeTrue())
			r.engine.GetFilteredNamespaces().Add("test-remove-namespace")
			Expect(r.engine.GetFilteredNamespaces().Contains("test-remove-namespace")).To(BeTrue())
			r.cache.Set("test-remove-namespace", v3.StagedNetworkPolicy{})
			v, ok := r.cache.Get("test-remove-namespace")
			Expect(v).ToNot(BeNil())
			Expect(ok).To(BeTrue())
			// Create a namespace to delete, and verify that it was created.
			_, err := mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-remove-namespace",
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				val, err := mockClientSet.CoreV1().Namespaces().Get(context.TODO(), "test-remove-namespace", metav1.GetOptions{})
				if err != nil {
					return false
				}
				return val != nil && val.Name == "test-remove-namespace"
			}, 10*time.Second).Should(BeTrue())

			go func() {
				_, err := mockClientSet.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-new-namespace",
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					v, err := mockClientSet.CoreV1().Namespaces().Get(context.TODO(), "test-new-namespace", metav1.GetOptions{})
					if err != nil && kerrors.IsNotFound(err) {
						return false
					}
					return v != nil && v.Name == "test-new-namespace"
				}, 10*time.Second).Should(BeTrue())

				err = r.Reconcile(types.NamespacedName{
					Name:      "test-new-namespace",
					Namespace: "test-new-namespace",
				})
				Expect(err).ToNot(HaveOccurred())
			}()

			go func() {
				err := mockClientSet.CoreV1().Namespaces().Delete(context.TODO(), "test-remove-namespace", metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					v, err := mockClientSet.CoreV1().Namespaces().Get(context.TODO(), "test-remove-namespace", metav1.GetOptions{})
					if err != nil && kerrors.IsNotFound(err) {
						return true
					}
					return v == nil
				}, 10*time.Second).Should(BeTrue())

				err = r.Reconcile(types.NamespacedName{
					Name:      "test-remove-namespace",
					Namespace: "test-remove-namespace",
				})
				Expect(err).ToNot(HaveOccurred())
			}()

			Eventually(func() bool {
				v, ok := r.cache.Get("test-remove-namespace")
				if ok && v != nil {
					if !r.engine.GetNamespaces().Contains("test-remove-namespace") && !r.engine.GetFilteredNamespaces().Contains("test-remove-namespace") {
						return true
					}
				}
				return false
			}).Should(BeFalse())
			Eventually(func() bool {
				return r.engine.GetNamespaces().Contains("test-new-namespace") && r.engine.GetFilteredNamespaces().Contains("test-new-namespace")
			}, 10*time.Second).Should(BeTrue())
		})

		It("should remove the namespace reference from the rules of all other cache items", func() {
			r.cache.Set("test-namespace", v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-namespace",
					Namespace: "test-namespace",
				},
				Spec: v3.StagedNetworkPolicySpec{
					Egress: []v3.Rule{
						{
							Action: "Allow",
							Destination: v3.EntityRule{
								NamespaceSelector: "test-delete-namespace",
							},
						},
						{
							Action: "Allow",
							Destination: v3.EntityRule{
								NamespaceSelector: "test-namespace-2",
							},
						},
					},
					Ingress: []v3.Rule{
						{
							Action: "Allow",
							Source: v3.EntityRule{
								NamespaceSelector: "test-delete-namespace",
							},
						},
						{
							Action: "Allow",
							Source: v3.EntityRule{
								NamespaceSelector: "test-namespace-2",
							},
						},
					},
				},
			})

			r.engine.GetNamespaces().Add("test-delete-namespace")
			r.cache.Set("test-delete-namespace", v3.StagedNetworkPolicy{})

			err := mockClientSet.CoreV1().Namespaces().Delete(context.TODO(), "test-delete-namespace", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			key = types.NamespacedName{
				Name:      "test-delete-namespace",
				Namespace: "test-delete-namespace",
			}

			err = r.Reconcile(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.cache.Get(key.Name)).To(BeNil())
			Expect(r.engine.GetNamespaces().Contains(key.Name)).To(BeFalse())

			// Check that the recommendation was updated.
			item, ok := r.cache.Get("test-namespace")
			Expect(ok).To(BeTrue())
			snp := item.(v3.StagedNetworkPolicy)
			Expect(len(snp.Spec.Egress)).To(Equal(1))
			Expect(snp.Spec.Egress[0].Destination.NamespaceSelector).To(Equal("test-namespace-2"))
			Expect(len(snp.Spec.Ingress)).To(Equal(1))
			Expect(snp.Spec.Ingress[0].Source.NamespaceSelector).To(Equal("test-namespace-2"))
		})
	})
})
