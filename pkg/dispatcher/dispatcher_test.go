// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package dispatcher_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/set"

	"github.com/tigera/compliance/pkg/dispatcher"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

var (
	pod1ID = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypePods,
		NameNamespace: resources.NameNamespace{
			Name:      "test",
			Namespace: "namespace",
		},
	}
	pod1Add = syncer.Update{
		Type:       syncer.UpdateTypeNew,
		ResourceID: pod1ID,
		Resource:   resources.NewResource(resources.ResourceTypePods),
	}
	pod2ID = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypePods,
		NameNamespace: resources.NameNamespace{
			Name:      "test2",
			Namespace: "namespace2",
		},
	}
	pod2Delete = syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: pod2ID,
	}
	policy1ID = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeNetworkPolicies,
		NameNamespace: resources.NameNamespace{
			Name:      "testNP",
			Namespace: "namespace",
		},
	}
	policy1Update = syncer.Update{
		Type:       syncer.UpdateTypeUpdated,
		ResourceID: policy1ID,
		Resource:   resources.NewResource(resources.ResourceTypeNetworkPolicies),
	}
	policy2ID = resources.ResourceID{
		GroupVersionKind: resources.ResourceTypeNetworkPolicies,
		NameNamespace: resources.NameNamespace{
			Name:      "testNP2",
			Namespace: "namespace",
		},
	}
	policy2Delete = syncer.Update{
		Type:       syncer.UpdateTypeDeleted,
		ResourceID: policy2ID,
	}
)

type tester struct {
	d         dispatcher.Dispatcher
	policies  set.Set
	pods      set.Set
	status    set.Set
	resources int
}

func newTester() *tester {
	return &tester{
		d:        dispatcher.NewDispatcher(),
		policies: set.New(),
		pods:     set.New(),
		status:   set.New(),
	}
}

func (t *tester) onPodUpdate(update syncer.Update) {
	log.WithField("type", update.Type).Info("Pod update")
	t.pods.Add(update.ResourceID)
	if update.Resource != nil {
		t.resources++
	}
}

func (t *tester) onPolicyUpdate(update syncer.Update) {
	log.WithField("type", update.Type).Info("Policy update")
	t.policies.Add(update.ResourceID)
	if update.Resource != nil {
		t.resources++
	}
}

func (t *tester) onStatusUpdate(status syncer.StatusType) {
	log.WithField("status", status).Info("Status update")
	t.status.Add(status)
}

func (t *tester) registerPodUpdates(types syncer.UpdateType) {
	t.d.RegisterOnUpdateHandler(
		resources.ResourceTypePods,
		types,
		t.onPodUpdate,
	)
}

func (t *tester) registerPolicyUpdates(types syncer.UpdateType) {
	t.d.RegisterOnUpdateHandler(
		resources.ResourceTypeNetworkPolicies,
		types,
		t.onPolicyUpdate,
	)
}

func (t *tester) registerOnStatusCallbacks() {
	t.d.RegisterOnStatusUpdateHandler(t.onStatusUpdate)
}

var _ = Describe("label selector checks", func() {
	It("should get pod callbacks when registered", func() {
		t := newTester()

		By("Registering for pod deleted and new callbacks, and no policy or status callbacks")
		t.registerPodUpdates(syncer.UpdateTypeDeleted | syncer.UpdateTypeNew)

		By("Sending new and deleted pod updates")
		t.d.OnUpdate(pod1Add)
		t.d.OnUpdate(pod2Delete)

		By("Sending update and deleted policy updates")
		t.d.OnUpdate(policy1Update)
		t.d.OnUpdate(policy2Delete)

		By("Sending an onStatusUpdate")
		t.d.OnStatusUpdate(syncer.StatusTypeInSync)

		By("Checking we get updates for both pod resources")
		Expect(t.pods.Len()).To(Equal(2))
		Expect(t.pods.Equals(set.From(pod1ID, pod2ID))).To(BeTrue())

		By("Checking we get no updates for policy resources")
		Expect(t.policies.Len()).To(BeZero())

		By("Checking we got one new or modified resource")
		Expect(t.resources).To(Equal(1))

		By("Checking we get no status updates")
		Expect(t.status.Len()).To(BeZero())
	})

	It("should get pod, policy and status callbacks when registered", func() {
		t := newTester()

		By("Registering for pod deleted, policy updated and status callbacks")
		t.registerPodUpdates(syncer.UpdateTypeDeleted)
		t.registerPolicyUpdates(syncer.UpdateTypeUpdated)
		t.registerOnStatusCallbacks()

		By("Sending new and deleted pod updates")
		t.d.OnUpdate(pod1Add)
		t.d.OnUpdate(pod2Delete)

		By("Sending update and deleted policy updates")
		t.d.OnUpdate(policy1Update)
		t.d.OnUpdate(policy2Delete)

		By("Sending an onStatusUpdate")
		t.d.OnStatusUpdate(syncer.StatusTypeInSync)

		By("Checking we get updates for the pod delete")
		Expect(t.pods.Len()).To(Equal(1))
		Expect(t.pods.Equals(set.From(pod2ID))).To(BeTrue())

		By("Checking we get updates for the policy update")
		Expect(t.policies.Len()).To(Equal(1))
		Expect(t.policies.Equals(set.From(policy1ID))).To(BeTrue())

		By("Checking we got one new or modified resource")
		Expect(t.resources).To(Equal(1))
		Expect(t.status.Len()).To(Equal(1))

		By("Checking we the in-sync status update")
		Expect(t.status.Equals(set.From(syncer.StatusTypeInSync))).To(BeTrue())
	})

})
