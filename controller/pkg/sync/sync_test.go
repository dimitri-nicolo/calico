// Copyright 2019 Tigera Inc. All rights reserved.

package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/statser"

	. "github.com/onsi/gomega"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/mock"
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

type FailFunc struct {
	Called bool
}

func (m *FailFunc) Fail() {
	m.Called = true
}
