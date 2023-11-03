package syncer_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	netV1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakek8s "k8s.io/client-go/kubernetes/fake"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	fakecalico "github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
	"github.com/projectcalico/calico/policy-recommendation/utils"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

type testStruct struct{}

var _ = Describe("Syncer", func() {
	const (
		testNamespacePrefix = "test-namespace"
		prScopeName         = "test-pr-scope"
		testTierName        = "testtier"
	)
	var (
		testCtx             context.Context
		testCancel          context.CancelFunc
		synchronizer        client.QueryInterface
		mockLmaK8sClientSet *lmak8s.MockClientSet
		fakeClient          *fakecalico.Clientset
		k8sClient           kubernetes.Interface
		cacheSet            syncer.CacheSet

		mockSuffixGenerator func() string
	)

	BeforeEach(func() {
		testCtx, testCancel = context.WithCancel(context.Background())
		// Define the kubernetes interface
		mockLmaK8sClientSet = &lmak8s.MockClientSet{}
		fakeClient = fakecalico.NewSimpleClientset()
		k8sClient = fakek8s.NewSimpleClientset()

		mockLmaK8sClientSet.On("CoreV1").Return(
			k8sClient.CoreV1(),
		)

		mockLmaK8sClientSet.On("ProjectcalicoV3").Return(
			fakeClient.ProjectcalicoV3(),
		)

		cacheSet = syncer.CacheSet{
			Namespaces:            cache.NewSynchronizedObjectCache[*v1.Namespace](),
			StagedNetworkPolicies: cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy](),
			NetworkSets:           cache.NewSynchronizedObjectCache[*v3.NetworkSet](),
		}

		mockSuffixGenerator = func() string {
			return "xv5fb"
		}

		synchronizer = syncer.NewCacheSynchronizer(mockLmaK8sClientSet, cacheSet, mockSuffixGenerator)
	})

	AfterEach(func() {
		testCancel()
	})

	Context("Unrecognized Query", func() {
		It("throws an error if the Source of the Query is not a v3.PolicyRecommendationScope", func() {
			result, err := synchronizer.RunQuery(testCtx, testStruct{})
			Expect(result).To(BeNil())
			Expect(err).ToNot(BeNil())
		})
	})

	Context("PolicyRecommendationScopeQuery", func() {
		It("throws an error if the Source of the Query is not a v3.PolicyRecommendationScope", func() {
			query := syncer.PolicyRecommendationScopeQuery{
				MetaSelectors: syncer.MetaSelectors{
					Source: &api.Update{
						UpdateType: api.UpdateTypeKVNew,
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Name: "test-name",
								Kind: "test",
							},
							Value: netV1.NetworkPolicy{},
						},
					},
				},
			}

			result, err := synchronizer.RunQuery(testCtx, query)
			Expect(result).To(BeNil())
			Expect(err).ToNot(BeNil())
		})

		It("creates a StagedNetworkPolicy in cache but not in the datastore for each existing namespace v3.PolicyRecommendationScope", func() {
			policyRec := v3.PolicyRecommendationScope{
				ObjectMeta: metav1.ObjectMeta{
					Name: prScopeName,
				},
				Spec: v3.PolicyRecommendationScopeSpec{
					NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
						TierName:  testTierName,
						RecStatus: v3.PolicyRecommendationScopeEnabled,
					},
				},
			}

			testNamespacePrefix := "test-namespace"

			for i := 0; i < 2; i++ {
				_, err := k8sClient.CoreV1().Namespaces().Create(
					testCtx,
					&v1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: fmt.Sprintf("%s-%d", testNamespacePrefix, i),
						},
					}, metav1.CreateOptions{})

				Expect(err).ShouldNot(HaveOccurred())
			}

			query := syncer.PolicyRecommendationScopeQuery{
				MetaSelectors: syncer.MetaSelectors{
					Source: &api.Update{
						UpdateType: api.UpdateTypeKVNew,
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Name: prScopeName,
								Kind: v3.KindPolicyRecommendationScope,
							},
							Value: &policyRec,
						},
					},
				},
			}

			result, err := synchronizer.RunQuery(testCtx, query)

			Expect(err).To(BeNil())
			policyRecResult, ok := result.(*syncer.PolicyReqScopeQueryResult)
			Expect(ok).To(BeTrue())
			Expect(policyRecResult.StagedNetworkPolicies).To(HaveLen(2))

			for i := 0; i < 2; i++ {
				ns := fmt.Sprintf("%s-%d", testNamespacePrefix, i)
				storeName := utils.GetPolicyName(testTierName, ns, mockSuffixGenerator)
				snpsOnCluster, err := fakeClient.ProjectcalicoV3().StagedNetworkPolicies(ns).Get(
					testCtx,
					storeName,
					metav1.GetOptions{})
				// The namespace sych query is not responsible for creating staged network polices
				Expect(err).NotTo(BeNil())
				Expect(snpsOnCluster).To(BeNil())

				// check caches are also updated
				snpInCache := cacheSet.StagedNetworkPolicies.Get(ns)
				Expect(snpInCache).ToNot(BeNil())
				Expect(snpInCache.Spec.StagedAction).To(Equal(v3.StagedActionLearn))
				Expect(snpInCache.Spec.Tier).To(Equal(testTierName))
				// Assert no rules are created yet
				Expect(snpInCache.Spec.Ingress).To(BeEmpty())
				Expect(snpInCache.Spec.Egress).To(BeEmpty())

				nsInCache := cacheSet.Namespaces.Get(ns)
				Expect(nsInCache).ToNot(BeNil())
			}
		})
	})

	It("deletes all StagedNetworkPolicy in each namespace", func() {
		snpCache := cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()
		for i := 0; i < 2; i++ {
			ns := fmt.Sprintf("%s-%d", testNamespacePrefix, i)
			snp := v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.GetPolicyName(testTierName, ns, mockSuffixGenerator),
					Namespace: ns,
				},
			}
			_, err := fakeClient.ProjectcalicoV3().StagedNetworkPolicies(ns).Create(
				testCtx,
				&snp,
				metav1.CreateOptions{})

			Expect(err).ShouldNot(HaveOccurred())
			snpCache.Set(snp.Name, &snp)
		}

		cacheSet = syncer.CacheSet{
			StagedNetworkPolicies: snpCache,
		}
		synchronizer = syncer.NewCacheSynchronizer(mockLmaK8sClientSet, cacheSet, mockSuffixGenerator)

		// pre-req setup policyrec that will be deleted
		policyRec := v3.PolicyRecommendationScope{
			ObjectMeta: metav1.ObjectMeta{
				Name: prScopeName,
			},
			Spec: v3.PolicyRecommendationScopeSpec{
				NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
					TierName:  testTierName,
					RecStatus: v3.PolicyRecommendationScopeEnabled,
				},
			},
		}

		query := syncer.PolicyRecommendationScopeQuery{
			MetaSelectors: syncer.MetaSelectors{
				Source: &api.Update{
					UpdateType: api.UpdateTypeKVNew,
					KVPair: model.KVPair{
						Key: model.ResourceKey{
							Name: prScopeName,
							Kind: v3.KindPolicyRecommendationScope,
						},
						Value: &policyRec,
					},
				},
			},
		}

		_, err := synchronizer.RunQuery(testCtx, query)

		Expect(err).To(BeNil())

		query = syncer.PolicyRecommendationScopeQuery{
			MetaSelectors: syncer.MetaSelectors{
				Source: &api.Update{
					UpdateType: api.UpdateTypeKVDeleted,
					KVPair: model.KVPair{
						Key: model.ResourceKey{
							Name: prScopeName,
							Kind: v3.KindPolicyRecommendationScope,
						},
					},
				},
			},
		}

		result, err := synchronizer.RunQuery(testCtx, query)
		Expect(result).To(BeNil())
		Expect(err).To(BeNil())

		for i := 0; i < 2; i++ {
			ns := fmt.Sprintf("%s-%d", testNamespacePrefix, i)
			snpName := utils.GetPolicyName(testTierName, ns, mockSuffixGenerator)
			_, err := fakeClient.ProjectcalicoV3().StagedNetworkPolicies(ns).Get(
				testCtx,
				snpName,
				metav1.GetOptions{})
			Expect(err).ToNot(BeNil())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())

			snpInCache := cacheSet.StagedNetworkPolicies.Get(ns)
			Expect(snpInCache).To(BeNil())
		}
	})

	Context("NamespaceQuery", func() {
		It("throws an error if the Source of the Query is not a v1.Namespace", func() {
			query := syncer.PolicyRecommendationScopeQuery{
				MetaSelectors: syncer.MetaSelectors{
					Source: &api.Update{
						UpdateType: api.UpdateTypeKVNew,
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Name: "test-name",
								Kind: "test",
							},
							Value: netV1.NetworkPolicy{},
						},
					},
				},
			}

			result, err := synchronizer.RunQuery(testCtx, query)
			Expect(result).To(BeNil())
			Expect(err).ToNot(BeNil())
		})

		It("creates an empty SNP in the cache, but not in the datastore for the newly added namespace", func() {
			// setup policyrec to enable namespace listening
			policyRec := v3.PolicyRecommendationScope{
				ObjectMeta: metav1.ObjectMeta{
					Name: prScopeName,
				},
				Spec: v3.PolicyRecommendationScopeSpec{
					NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
						TierName:  testTierName,
						RecStatus: v3.PolicyRecommendationScopeEnabled,
					},
				},
			}

			preReqQuery := syncer.PolicyRecommendationScopeQuery{
				MetaSelectors: syncer.MetaSelectors{
					Source: &api.Update{
						UpdateType: api.UpdateTypeKVNew,
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Name: prScopeName,
								Kind: v3.KindPolicyRecommendationScope,
							},
							Value: &policyRec,
						},
					},
				},
			}

			_, err := synchronizer.RunQuery(testCtx, preReqQuery)
			Expect(err).To(BeNil())

			ns := fmt.Sprintf("%s-%d", testNamespacePrefix, 1)
			namespace := v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			}
			query := syncer.NamespaceQuery{
				MetaSelectors: syncer.MetaSelectors{
					Source: &api.Update{
						UpdateType: api.UpdateTypeKVNew,
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Name: ns,
								Kind: "Namespace",
							},
							Value: &namespace,
						},
					},
				},
			}

			result, err := synchronizer.RunQuery(testCtx, query)

			Expect(err).To(BeNil())
			policyRecResult, ok := result.(*syncer.NamespaceQueryResult)
			Expect(ok).To(BeTrue())
			Expect(policyRecResult.StagedNetworkPolicies).To(HaveLen(1))

			snpsOnCluster, err := fakeClient.ProjectcalicoV3().StagedNetworkPolicies(ns).Get(
				testCtx,
				utils.GetPolicyName(testTierName, ns, mockSuffixGenerator),
				metav1.GetOptions{})
			Expect(err).NotTo(BeNil())
			Expect(snpsOnCluster).To(BeNil())
		})

		It("deletes all SNP from the removed namespace", func() {
			snpCache := cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()

			ns1 := "namespace1"
			ns2 := "namespace2"
			ns3 := "namespace3"

			snp1 := v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ns1 + "-snp1",
					Namespace: ns1,
				},
				Spec: v3.StagedNetworkPolicySpec{
					Ingress: []v3.Rule{
						{
							Destination: v3.EntityRule{
								NamespaceSelector: ns2,
							},
						},
					},
					Egress: []v3.Rule{
						{
							Destination: v3.EntityRule{
								NamespaceSelector: ns3,
							},
						},
					},
				},
			}

			snp2 := v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ns2 + "-snp2",
					Namespace: ns2,
				},
				Spec: v3.StagedNetworkPolicySpec{
					Ingress: []v3.Rule{
						{
							Destination: v3.EntityRule{
								NamespaceSelector: ns3,
							},
						},
					},
					Egress: []v3.Rule{
						{
							Destination: v3.EntityRule{
								NamespaceSelector: ns1,
							},
						},
					},
				},
			}

			snp3 := v3.StagedNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ns3 + "-snp3",
					Namespace: ns3,
				},
				Spec: v3.StagedNetworkPolicySpec{
					Ingress: []v3.Rule{
						{
							Destination: v3.EntityRule{
								NamespaceSelector: ns2,
							},
						},
					},
					Egress: []v3.Rule{
						{
							Destination: v3.EntityRule{
								NamespaceSelector: ns3,
							},
						},
					},
				},
			}

			snpCache.Set(ns1, &snp1)
			_, err := fakeClient.ProjectcalicoV3().StagedNetworkPolicies(ns1).Create(testCtx, &snp1, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			snpCache.Set(ns2, &snp2)
			_, err = fakeClient.ProjectcalicoV3().StagedNetworkPolicies(ns2).Create(testCtx, &snp2, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			snpCache.Set(ns3, &snp3)
			_, err = fakeClient.ProjectcalicoV3().StagedNetworkPolicies(ns3).Create(testCtx, &snp3, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			cacheSet = syncer.CacheSet{
				StagedNetworkPolicies: snpCache,
			}
			synchronizer = syncer.NewCacheSynchronizer(mockLmaK8sClientSet, cacheSet, mockSuffixGenerator)

			// Setup policyrec to enable namespace listening
			policyRec := v3.PolicyRecommendationScope{
				ObjectMeta: metav1.ObjectMeta{
					Name: prScopeName,
				},
				Spec: v3.PolicyRecommendationScopeSpec{
					NamespaceSpec: v3.PolicyRecommendationScopeNamespaceSpec{
						TierName:  testTierName,
						RecStatus: v3.PolicyRecommendationScopeEnabled,
					},
				},
			}

			preReqQuery := syncer.PolicyRecommendationScopeQuery{
				MetaSelectors: syncer.MetaSelectors{
					Source: &api.Update{
						UpdateType: api.UpdateTypeKVNew,
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Name: prScopeName,
								Kind: v3.KindPolicyRecommendationScope,
							},
							Value: &policyRec,
						},
					},
				},
			}

			_, err = synchronizer.RunQuery(testCtx, preReqQuery)
			Expect(err).To(BeNil())

			// Delete namesoace1
			namespace := v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns1,
				},
			}
			query := syncer.NamespaceQuery{
				MetaSelectors: syncer.MetaSelectors{
					Source: &api.Update{
						UpdateType: api.UpdateTypeKVDeleted,
						KVPair: model.KVPair{
							Key: model.ResourceKey{
								Name: ns1,
								Kind: "Namespace",
							},
							Value: &namespace,
						},
					},
				},
			}

			result, err := synchronizer.RunQuery(testCtx, query)
			Expect(err).To(BeNil())

			namespaceResult, ok := result.(*syncer.NamespaceQueryResult)
			Expect(ok).To(BeTrue())
			Expect(namespaceResult.StagedNetworkPolicies).ToNot(BeNil())

			// Deleting namespace1 will delete the key from the cache, and by the value from the datastore
			// by default
			snp1CacheItem := cacheSet.StagedNetworkPolicies.Get(snp1.Namespace)
			Expect(snp1CacheItem).To(BeNil())

			snp2CacheItem := cacheSet.StagedNetworkPolicies.Get(snp2.Namespace)
			Expect(snp2CacheItem).ToNot(BeNil())
			Expect(snp2CacheItem.Spec.Ingress).To(Equal([]v3.Rule{
				{
					Destination: v3.EntityRule{
						NamespaceSelector: ns3,
					},
				},
			}))
			Expect(snp2CacheItem.Spec.Egress).To(BeEmpty())

			snp3CacheItem := cacheSet.StagedNetworkPolicies.Get(snp3.Namespace)
			Expect(snp3CacheItem).ToNot(BeNil())
			Expect(snp3CacheItem.Spec.Ingress).To(Equal([]v3.Rule{
				{
					Destination: v3.EntityRule{
						NamespaceSelector: ns2,
					},
				},
			}))
			Expect(snp3CacheItem.Spec.Egress).To(Equal([]v3.Rule{
				{
					Destination: v3.EntityRule{
						NamespaceSelector: ns3,
					},
				},
			}))
		})
	})
})
