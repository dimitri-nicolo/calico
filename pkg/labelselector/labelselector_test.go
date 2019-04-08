// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package labelselector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/tigera/compliance/pkg/labelselector"
	"github.com/tigera/compliance/pkg/resources"
)

var (
	podID = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypePods,
		NameNamespace: resources.NameNamespace{
			Name:      "test",
			Namespace: "namespace",
		},
	}
	policyID = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeNetworkPolicies,
		NameNamespace: resources.NameNamespace{
			Name:      "testNP",
			Namespace: "namespace",
		},
	}
	podSet    = set.From(podID)
	policySet = set.From(policyID)
)

type tester struct {
	l        labelselector.Interface
	policies set.Set
	pods     set.Set
}

func newTester() *tester {
	return &tester{
		l:        labelselector.NewLabelSelection(),
		policies: set.New(),
		pods:     set.New(),
	}
}

func (t *tester) onMatchPodStart(policy, pod resources.ResourceID) {
	t.pods.Add(pod)
}

func (t *tester) onMatchPodStopped(policy, pod resources.ResourceID) {
	t.pods.Discard(pod)
}

func (t *tester) onMatchPolicyStart(policy, pod resources.ResourceID) {
	t.policies.Add(policy)
}

func (t *tester) onMatchPolicyStopped(policy, pod resources.ResourceID) {
	t.policies.Discard(policy)
}

func (t *tester) registerPodCallbacks() {
	t.l.RegisterCallbacks(
		[]schema.GroupVersionKind{resources.ResourceTypePods},
		t.onMatchPodStart,
		t.onMatchPodStopped,
	)
}

func (t *tester) registerPolicyCallbacks() {
	t.l.RegisterCallbacks(
		[]schema.GroupVersionKind{resources.ResourceTypeNetworkPolicies},
		t.onMatchPolicyStart,
		t.onMatchPolicyStopped,
	)
}

var _ = Describe("label selector checks", func() {
	It("should get pod and policy callbacks if both registered", func() {
		t := newTester()

		By("Registering for pod and policy callbacks")
		t.registerPodCallbacks()
		t.registerPolicyCallbacks()

		By("Adding a matching selector/label")
		t.l.UpdateSelector(policyID, "thing == 'yes'")
		t.l.UpdateLabels(podID, map[string]string{
			"thing": "yes",
		}, nil)
		Expect(t.policies.Equals(policySet)).To(BeTrue())
		Expect(t.pods.Equals(podSet)).To(BeTrue())

		By("Removing the match")
		t.l.UpdateSelector(policyID, "thing == 'no'")
		Expect(t.policies.Len()).To(BeZero())
		Expect(t.pods.Len()).To(BeZero())
	})

	It("should get pod callbacks if registered, but not policy", func() {
		t := newTester()

		By("Registering for pod callbacks")
		t.registerPodCallbacks()

		By("Adding a matching selector/label")
		t.l.UpdateSelector(policyID, "thing == 'boo'")
		t.l.UpdateLabels(podID, map[string]string{
			"thing": "boo",
		}, nil)
		Expect(t.policies.Len()).To(BeZero())
		Expect(t.pods.Equals(podSet)).To(BeTrue())

		By("Removing the match")
		t.l.UpdateLabels(podID, map[string]string{
			"thing": "foo",
		}, nil)
		Expect(t.policies.Len()).To(BeZero())
		Expect(t.pods.Len()).To(BeZero())
	})

	It("should get policy callbacks if registered, but not pod", func() {
		t := newTester()

		By("Registering for policy callbacks")
		t.registerPolicyCallbacks()

		By("Adding a matching selector/label")
		t.l.UpdateSelector(policyID, "thing == 'boo'")
		t.l.UpdateLabels(podID, map[string]string{
			"thing": "boo",
		}, nil)
		Expect(t.policies.Equals(policySet)).To(BeTrue())
		Expect(t.pods.Len()).To(BeZero())

		By("Removing the match")
		t.l.UpdateLabels(podID, map[string]string{
			"thing": "foo",
		}, nil)
		Expect(t.policies.Len()).To(BeZero())
		Expect(t.pods.Len()).To(BeZero())
	})

	It("should handle parent inheritance of labels", func() {
		t := newTester()

		By("Registering for policy callbacks")
		t.registerPolicyCallbacks()
		t.registerPodCallbacks()

		By("Adding a matching selector/label via parent")
		t.l.UpdateSelector(policyID, "thing == 'boo'")
		t.l.UpdateParentLabels("parent", map[string]string{
			"thing": "boo",
		})
		t.l.UpdateLabels(podID, nil, []string{"parent"})
		Expect(t.policies.Equals(policySet)).To(BeTrue())
		Expect(t.pods.Equals(podSet)).To(BeTrue())

		By("Removing the parent")
		t.l.UpdateLabels(podID, nil, []string{"afakeparent"})
		Expect(t.policies.Len()).To(BeZero())
		Expect(t.pods.Len()).To(BeZero())
	})
})
