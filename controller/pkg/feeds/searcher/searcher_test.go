// Copyright 2019 Tigera Inc. All rights reserved.

package searcher

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

// TestDoIPSet tests the case where everything is working
func TestDoIPSet(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{
		{
			SourceIP:   util.Sptr("1.2.3.4"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
		{
			SourceIP:   util.Sptr("5.6.7.8"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
	}
	runTest(t, true, expected, nil, -1, -1)
}

// TestDoIPSetNoResults tests the case where no results are returned
func TestDoIPSetNoResults(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{}
	runTest(t, true, expected, nil, -1, -1)
}

// TestDoIPSetSuspiciousIPFails tests the case where suspiciousIP fails after the first result
func TestDoIPSetSuspiciousIPFails(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{}
	runTest(t, false, expected, errors.New("fail"), -1, -1)
}

func TestDoIPSetSuspiciousIPIterationFails(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{
		{
			SourceIP:   util.Sptr("1.2.3.4"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
		{
			SourceIP:   util.Sptr("5.6.7.8"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
	}
	runTest(t, false, expected, nil, 1, -1)
}

// TestDoIPSetEventsFails tests the case where the first call to events.PutSecurityEvent fails but the second does not
func TestDoIPSetEventsFails(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{
		{
			SourceIP:   util.Sptr("1.2.3.4"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
		{
			SourceIP:   util.Sptr("5.6.7.8"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
	}
	runTest(t, false, expected, nil, -1, 0)
}

func runTest(t *testing.T, successful bool, expected []events.SuspiciousIPSecurityEvent, err error, suspiciousErrorIdx, eventsErrorIdx int) {
	g := NewGomegaWithT(t)

	f := util.NewGlobalThreatFeedFromName("mock")
	suspiciousIP := &db.MockSuspiciousIP{Error: err, ErrorIndex: suspiciousErrorIdx}
	for _, e := range expected {
		suspiciousIP.FlowLogs = append(suspiciousIP.FlowLogs, e)
	}
	eventsDB := &db.MockEvents{ErrorIndex: eventsErrorIdx, FlowLogs: []db.SecurityEventInterface{}}
	searcher := NewFlowSearcher(f, 0, suspiciousIP, eventsDB).(*flowSearcher)
	s := &statser.MockStatser{}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	searcher.doIPSet(ctx, s)

	if successful {
		g.Expect(eventsDB.FlowLogs).Should(ConsistOf(expected), "Logs in DB should match expected")
		g.Expect(suspiciousIP.FlowLogs).Should(HaveLen(0), "All flowLogs from suspiciousIP were consumed")
	} else {
		if eventsErrorIdx >= 0 {
			g.Expect(eventsDB.FlowLogs).Should(HaveLen(len(expected)-1), "Logs in DB should have skipped 1 from input")
		}
		if suspiciousErrorIdx >= 0 {
			g.Expect(eventsDB.FlowLogs).Should(HaveLen(suspiciousErrorIdx), "Logs in DB should stop at the first error")
		}
	}

	status := s.Status()
	g.Expect(status.LastSuccessfulSync.Time).Should(Equal(time.Time{}), "Sync should not be marked as successful")
	if successful {
		g.Expect(status.LastSuccessfulSearch.Time).ShouldNot(Equal(time.Time{}), "Search should be marked as successful")
		g.Expect(status.ErrorConditions).Should(HaveLen(0), "No errors should be reported")
	} else {
		g.Expect(status.LastSuccessfulSearch.Time).Should(Equal(time.Time{}), "Search should be not marked as successful")
		g.Expect(status.ErrorConditions).ShouldNot(HaveLen(0), "Errors should be reported")
	}
}

func TestFlowSearcher_SetFeed(t *testing.T) {
	g := NewGomegaWithT(t)

	f := util.NewGlobalThreatFeedFromName("mock")
	f2 := util.NewGlobalThreatFeedFromName("swap")
	suspiciousIP := &db.MockSuspiciousIP{}
	eventsDB := &db.MockEvents{}
	searcher := NewFlowSearcher(f, 0, suspiciousIP, eventsDB).(*flowSearcher)

	searcher.SetFeed(f2)
	g.Expect(searcher.feed).Should(Equal(f2))
	g.Expect(searcher.feed).ShouldNot(BeIdenticalTo(f2))
}
