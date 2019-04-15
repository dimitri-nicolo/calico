// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package replay_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/tigera/compliance/internal/testutils"
	"github.com/tigera/compliance/pkg/list"
	. "github.com/tigera/compliance/pkg/replay"
	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type mockCallbacks struct {
	updates       []syncer.Update
	statusUpdates []syncer.StatusUpdate
}

func (cb *mockCallbacks) OnUpdate(u syncer.Update) {
	cb.updates = append(cb.updates, u)
}

func (cb *mockCallbacks) OnStatusUpdate(su syncer.StatusUpdate) {
	cb.statusUpdates = append(cb.statusUpdates, su)
}

//
// These tests are functional verification (fv) tests
// meaning that a standing elasticsearch database is
// required to run them.
//
// To run locally, you can spin one up quickly using
// `make run-elastic`
//
// TODO: hook up to ci properly with GINKGO_FOCUS filtering
//
var _ = Describe("Replay", func() {
	//
	// The mock data was generated using the testdata/demo.sh script
	// with each kubectl command separated by 10 second intervals.
	// It was exported using the cmd/testdata-exporter binary.
	// using data generated from 4/3/2019 2001 - 2006 UTC
	var (
		//ns  = "compliance-testing"
		ctx = context.Background()

		baseTime     = time.Date(2019, 4, 3, 20, 01, 0, 0, time.UTC)
		replayTester *ReplayTester
		cb           *mockCallbacks
	)

	It("should send both an insync and a complete status update in a complete run through", func() {
		By("initializing the replayer with a replay tester than implements the required interfaces")
		replayTester = NewReplayTester(baseTime)
		cb = new(mockCallbacks)
		replayer := New(baseTime.Add(time.Minute), baseTime.Add(2*time.Minute), replayTester, replayTester, cb)

		By("storing a mock list of the tested network policy")

		// make the initial network policy without a typemeta
		np := &apiv3.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Namespace: "some-namespace", Name: "some-netpol"},
			Spec:       apiv3.NetworkPolicySpec{Selector: `foo == "bar"`},
		}

		npList := apiv3.NewNetworkPolicyList()
		npList.Items = append(npList.Items, *np)
		npResList := &list.TimestampedResourceList{
			ResourceList:              npList,
			RequestStartedTimestamp:   metav1.Time{baseTime.Add(15 * time.Second)},
			RequestCompletedTimestamp: metav1.Time{baseTime.Add(16 * time.Second)},
		}
		replayTester.StoreList(resources.TypeCalicoNetworkPolicies, npResList)

		By("setting a network policy audit event before the start time")
		np.TypeMeta = resources.TypeCalicoNetworkPolicies
		np.Spec.Selector = `foo == "baz"`
		replayTester.SetResourceAuditEvent(VerbUpdate, np, baseTime.Add(30*time.Second))

		By("setting a network policy audit event after the start time")
		np.Spec.Selector = `foo == "barbaz"`
		replayTester.SetResourceAuditEvent(VerbUpdate, np, baseTime.Add(75*time.Second))

		// Make the replay call.
		replayer.Start(ctx)

		By("ensuring that only one update was received since the first one occured before the start time")
		Expect(len(cb.updates)).To(Equal(2))
		Expect(cb.updates[0].ResourceID.String()).To(Equal("NetworkPolicy(some-namespace/some-netpol)"))
		Expect(cb.updates[1].ResourceID.String()).To(Equal("NetworkPolicy(some-namespace/some-netpol)"))

		Expect(cb.updates[0].Resource.(*apiv3.NetworkPolicy).Spec.Selector).To(Equal(`foo == "baz"`))
		Expect(cb.updates[1].Resource.(*apiv3.NetworkPolicy).Spec.Selector).To(Equal(`foo == "barbaz"`))

		By("ensuring that the in-sync and complete status update was received")
		Expect(cb.statusUpdates).To(ContainElement(syncer.NewStatusUpdateInSync()))
		Expect(cb.statusUpdates).To(ContainElement(syncer.NewStatusUpdateComplete()))
	})
})
