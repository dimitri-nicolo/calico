// Copyright 2019 Tigera Inc. All rights reserved.

package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v32 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/mock"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

func TestSync(t *testing.T) {
	expected := db.IPSetSpec{
		"1.2.3.4",
		"2.3.4.5",
	}
	testSync(t, true, expected, nil, nil)
}

func TestSyncEmpty(t *testing.T) {
	expected := db.IPSetSpec{}
	testSync(t, true, expected, nil, nil)
}

func TestSyncIPSetError(t *testing.T) {
	expected := db.IPSetSpec{}
	testSync(t, false, expected, errors.New("mock error"), nil)
}

func TestSyncGNSError(t *testing.T) {
	expected := db.IPSetSpec{}
	testSync(t, false, expected, nil, errors.New("mock error"))
}

func testSync(t *testing.T, successful bool, expected db.IPSetSpec, expectedIPSetError, expectedGNSError error) {
	g := NewGomegaWithT(t)

	ipSet := &mock.IPSet{
		Error: expectedIPSetError,
	}
	gns := &mock.GlobalNetworkSetInterface{
		Error: expectedGNSError,
	}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{},
	}
	st := &mock.Statser{}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	failFunc := &FailFunc{}
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	err := syncer.sync(ctx, expected, failFunc.Fail, st, 1, 0)
	if successful {
		g.Expect(err).ShouldNot(HaveOccurred())
	} else {
		g.Expect(err).Should(HaveOccurred())
	}

	g.Expect(expected).Should(Equal(ipSet.Set), "Feed contents match")

	s := st.Status()
	if successful {
		g.Expect(s.LastSuccessfulSync).ShouldNot(Equal(time.Time{}), "Sync was marked as successful")
		g.Expect(failFunc.Called).Should(BeFalse(), "Fail function not called")
	} else {
		g.Expect(s.LastSuccessfulSync).Should(Equal(time.Time{}), "Sync was not marked as successful")
		g.Expect(failFunc.Called).Should(BeTrue(), "Fail function called")
	}
	g.Expect(s.LastSuccessfulSearch).Should(Equal(time.Time{}), "Search was not marked as successful")
	switch {
	case expectedIPSetError == nil && expectedGNSError == nil:
		g.Expect(s.ErrorConditions).Should(HaveLen(0), "No status errors reported")
	case expectedIPSetError != nil && expectedGNSError == nil:
		g.Expect(s.ErrorConditions).Should(HaveLen(1), "1 status errors reported")
		g.Expect(s.ErrorConditions[0].Type).Should(Equal(statser.ElasticSyncFailed), "ErrorConditions type matches")
	case expectedIPSetError == nil && expectedGNSError != nil:
		g.Expect(s.ErrorConditions).Should(HaveLen(1), "1 status errors reported")
		g.Expect(s.ErrorConditions[0].Type).Should(Equal(statser.GlobalNetworkSetSyncFailed), "ErrorConditions type matches")
	case expectedIPSetError != nil && expectedGNSError != nil:
		g.Expect(s.ErrorConditions).Should(HaveLen(1), "2 status errors reported")
		g.Expect(s.ErrorConditions[0].Type).Should(Equal(statser.ElasticSyncFailed), "ErrorConditions type matches")
		g.Expect(s.ErrorConditions[1].Type).Should(Equal(statser.GlobalNetworkSetSyncFailed), "ErrorConditions type matches")
	}
}

func TestSyncer_SetFeed(t *testing.T) {
	g := NewGomegaWithT(t)

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f2 := util.NewGlobalThreatFeedFromName("swap")

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	syncer.SetFeed(f2)
	g.Expect(syncer.feed).Should(Equal(f2))
	g.Expect(syncer.feed).ShouldNot(BeIdenticalTo(f2))
}

func TestSyncer_RunSyncGlobalNetworkSet(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ips := db.IPSetSpec{
		"1.2.3.4",
		"5.6.7.8/32",
	}
	ipSet := &mock.IPSet{
		Set: ips,
	}
	gns := &mock.GlobalNetworkSetInterface{}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{
			"test": "label",
		},
	}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	syncer.runSyncGlobalNetworkSet(ctx, st)

	g.Expect(gns.GlobalNetworkSet).ShouldNot(BeNil())
	g.Expect(gns.GlobalNetworkSet.Spec.Nets).Should(ConsistOf(ips))
	g.Expect(gns.GlobalNetworkSet.Labels).Should(gstruct.MatchAllKeys(gstruct.Keys{
		"test": Equal(f.Spec.GlobalNetworkSet.Labels["test"]),
	}))

	status := st.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}))
	g.Expect(status.ErrorConditions).Should(HaveLen(0))
}

func TestSyncer_RunSyncGlobalNetworkSet_NoSync(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	syncer.runSyncGlobalNetworkSet(ctx, st)

	g.Expect(gns.GlobalNetworkSet).Should(BeNil())

	status := st.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}))
	g.Expect(status.ErrorConditions).Should(HaveLen(0))
}

func TestSyncer_RunSyncGlobalNetworkSet_IPSetFails(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ipSet := &mock.IPSet{
		Error: errors.New("test"),
	}
	gns := &mock.GlobalNetworkSetInterface{}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{
			"test": "label",
		},
	}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	syncer.runSyncGlobalNetworkSet(ctx, st)

	g.Expect(gns.GlobalNetworkSet).Should(BeNil())

	status := st.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}))
	g.Expect(status.ErrorConditions).Should(HaveLen(1))
}

func TestSyncer_RunSyncGlobalNetworkSet_SyncFails(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ips := db.IPSetSpec{
		"1.2.3.4",
		"5.6.7.8/32",
	}
	ipSet := &mock.IPSet{
		Set: ips,
	}
	gns := &mock.GlobalNetworkSetInterface{
		CreateError: errors.New("test"),
	}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{
			"test": "label",
		},
	}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	syncer.runSyncGlobalNetworkSet(ctx, st)

	g.Expect(gns.GlobalNetworkSet).Should(BeNil())

	status := st.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}))
	g.Expect(status.ErrorConditions).Should(HaveLen(1))
}

func TestSyncer_SyncGlobalNetworkSet_Create(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ips := db.IPSetSpec{
		"1.2.3.4",
		"5.6.7.8/32",
	}
	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{
			"test": "label",
		},
	}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	err := syncer.syncGlobalNetworkSet(ctx, f.Name, ips, f.Spec.GlobalNetworkSet, st)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(gns.GlobalNetworkSet).ShouldNot(BeNil())
	g.Expect(gns.GlobalNetworkSet.Name).Should(Equal("threatfeed.mock"))
	g.Expect(gns.GlobalNetworkSet.Annotations).Should(gstruct.MatchAllKeys(gstruct.Keys{
		"tigera.io/creator": Equal("intrusion-detection-controller"),
	}))
	g.Expect(gns.GlobalNetworkSet.Labels).Should(gstruct.MatchAllKeys(gstruct.Keys{
		"test": Equal(f.Spec.GlobalNetworkSet.Labels["test"]),
	}))
	g.Expect(gns.GlobalNetworkSet.Spec.Nets).Should(ConsistOf(ips))

	status := st.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}))
	g.Expect(status.ErrorConditions).Should(HaveLen(0))
}

func TestSyncer_SyncGlobalNetworkSet_CreateFail(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ips := db.IPSetSpec{
		"1.2.3.4",
		"5.6.7.8/32",
	}
	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{
		CreateError: errors.New("test"),
	}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{
			"test": "label",
		},
	}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	err := syncer.syncGlobalNetworkSet(ctx, f.Name, ips, f.Spec.GlobalNetworkSet, st)
	g.Expect(err).Should(HaveOccurred())
	g.Expect(gns.GlobalNetworkSet).Should(BeNil())
}

func TestSyncer_SyncGlobalNetworkSet_Update(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ips := db.IPSetSpec{
		"1.2.3.4",
		"5.6.7.8/32",
	}
	ipSet := &mock.IPSet{}
	now := time.Now()
	gns := &mock.GlobalNetworkSetInterface{
		GlobalNetworkSet: &v32.GlobalNetworkSet{
			ObjectMeta: v1.ObjectMeta{
				Name: "threatfeed.mock",
				CreationTimestamp: v1.Time{
					Time: now,
				},
				Labels: nil,
				Annotations: map[string]string{
					"test": "annotation",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: nil,
			},
		},
		CreateError: &kerrors.StatusError{
			ErrStatus: v1.Status{
				Reason: v1.StatusReasonAlreadyExists,
			},
		},
	}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{
			"test": "label",
		},
	}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	err := syncer.syncGlobalNetworkSet(ctx, f.Name, ips, f.Spec.GlobalNetworkSet, st)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(gns.GlobalNetworkSet).ShouldNot(BeNil())
	g.Expect(gns.GlobalNetworkSet.Name).Should(Equal("threatfeed.mock"))
	g.Expect(gns.GlobalNetworkSet.CreationTimestamp.Time).Should(Equal(now))
	g.Expect(gns.GlobalNetworkSet.Annotations).Should(gstruct.MatchAllKeys(gstruct.Keys{
		"test": Equal("annotation"),
	}))
	g.Expect(gns.GlobalNetworkSet.Labels).Should(gstruct.MatchAllKeys(gstruct.Keys{
		"test": Equal(f.Spec.GlobalNetworkSet.Labels["test"]),
	}))
	g.Expect(gns.GlobalNetworkSet.Spec.Nets).Should(ConsistOf(ips))
}

func TestSyncer_SyncGlobalNetworkSet_Update_GetFails(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ips := db.IPSetSpec{
		"1.2.3.4",
		"5.6.7.8/32",
	}
	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{
		CreateError: &kerrors.StatusError{
			ErrStatus: v1.Status{
				Reason: v1.StatusReasonAlreadyExists,
			},
		},
		GetError: errors.New("fail"),
	}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{
			"test": "label",
		},
	}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	err := syncer.syncGlobalNetworkSet(ctx, f.Name, ips, f.Spec.GlobalNetworkSet, st)
	g.Expect(err).Should(HaveOccurred())
	g.Expect(gns.GlobalNetworkSet).Should(BeNil())
}

func TestSyncer_SyncGlobalNetworkSet_Update_Fails(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ips := db.IPSetSpec{
		"1.2.3.4",
		"5.6.7.8/32",
	}
	ipSet := &mock.IPSet{}
	now := time.Now()
	gns := &mock.GlobalNetworkSetInterface{
		GlobalNetworkSet: &v32.GlobalNetworkSet{
			ObjectMeta: v1.ObjectMeta{
				Name: "threatfeed.mock",
				CreationTimestamp: v1.Time{
					Time: now,
				},
				Labels: nil,
				Annotations: map[string]string{
					"test": "annotation",
				},
			},
			Spec: v3.GlobalNetworkSetSpec{
				Nets: nil,
			},
		},
		CreateError: &kerrors.StatusError{
			ErrStatus: v1.Status{
				Reason: v1.StatusReasonAlreadyExists,
			},
		},
		UpdateError: errors.New("test"),
	}
	st := &mock.Statser{}
	f := util.NewGlobalThreatFeedFromName("mock")
	f.Spec.GlobalNetworkSet = &v3.GlobalNetworkSetSync{
		Labels: map[string]string{
			"test": "label",
		},
	}

	syncer := NewSyncer(f, ipSet, gns).(*syncer)

	err := syncer.syncGlobalNetworkSet(ctx, f.Name, ips, f.Spec.GlobalNetworkSet, st)
	g.Expect(err).Should(HaveOccurred())
}

type FailFunc struct {
	Called bool
}

func (m *FailFunc) Fail() {
	m.Called = true
}
