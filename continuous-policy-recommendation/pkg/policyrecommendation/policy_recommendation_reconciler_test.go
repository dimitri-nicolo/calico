package policyrecommendation

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset/fake"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

var _ = Describe("Policy Recommendation Scope Controller", func() {
	const (
		testResourceName = "TestName"
	)

	var (
		testCtx    context.Context
		testCancel context.CancelFunc
		pr         policyRecommendationReconciler
		calicoCLI  calicoclient.ProjectcalicoV3Interface
	)

	BeforeEach(func() {
		calicoCLI = fake.NewSimpleClientset().ProjectcalicoV3()
		testCtx, testCancel = context.WithCancel(context.Background())

		pr = policyRecommendationReconciler{
			calico: calicoCLI,
		}
	})

	AfterEach(func() {
		pr.Close()
		testCancel()
	})

	It("sets the controller state if the PolicyRecScope is found in the cluster", func() {
		prScopeInCluster := &v3.PolicyRecommendationScope{
			ObjectMeta: metav1.ObjectMeta{
				Name: testResourceName,
			},
		}
		_, err := calicoCLI.PolicyRecommendationScopes().Create(
			testCtx,
			prScopeInCluster,
			metav1.CreateOptions{},
		)

		Expect(err).To(BeNil())

		err = pr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})
		Expect(err).To(BeNil())

		Expect(pr.state.object).To(Equal(*prScopeInCluster))
		// TODO: check that engine is run once it's integrated

	})

	It("cancels the engine and removes the state if the policyrec is not found", func() {
		prScopeState := &v3.PolicyRecommendationScope{
			ObjectMeta: metav1.ObjectMeta{
				Name: testResourceName,
			},
		}

		_, cancel := context.WithCancel(context.Background())
		defer cancel()
		pr.state = &policyRecommendationScopeState{
			object: *prScopeState,
			cancel: cancel,
		}

		err := pr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})

		Expect(err).To(BeNil())

		Expect(pr.state).To(BeNil())
		// TODO: check that engine is run once it's integrated
	})

	It("updates the state with the one found in the cluster", func() {
		timeDuration := metav1.Duration{Duration: 150 * time.Second}
		prScopeState := &v3.PolicyRecommendationScope{
			ObjectMeta: metav1.ObjectMeta{
				Name: testResourceName,
			},
			Spec: v3.PolicyRecommendationScopeSpec{
				Interval: &timeDuration,
			},
		}

		_, cancel := context.WithCancel(context.Background())
		defer cancel()
		pr.state = &policyRecommendationScopeState{
			object: *prScopeState,
			cancel: cancel,
		}

		updatedTimeDuration := metav1.Duration{Duration: 120 * time.Second}
		prScopeInCluster := &v3.PolicyRecommendationScope{
			ObjectMeta: metav1.ObjectMeta{
				Name: testResourceName,
			},
			Spec: v3.PolicyRecommendationScopeSpec{
				Interval: &updatedTimeDuration,
			},
		}

		_, err := calicoCLI.PolicyRecommendationScopes().Create(
			testCtx,
			prScopeInCluster,
			metav1.CreateOptions{},
		)

		Expect(err).To(BeNil())

		err = pr.Reconcile(types.NamespacedName{
			Name: testResourceName,
		})
		Expect(err).To(BeNil())

		Expect(pr.state.object).To(Equal(*prScopeInCluster))
		// TODO: check that engine is run once it's integrated
	})

})
