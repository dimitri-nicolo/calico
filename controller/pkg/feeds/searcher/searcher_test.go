// Copyright 2019 Tigera Inc. All rights reserved.

package searcher

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/cacher"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
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
	runTest(t, true, expected, time.Now(), "", nil, -1)
}

// TestDoIPSetNoResults tests the case where no results are returned
func TestDoIPSetNoResults(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{}
	runTest(t, true, expected, time.Now(), "", nil, -1)
}

// TestDoIPSetSuspiciousIPFails tests the case where suspiciousIP fails after the first result
func TestDoIPSetSuspiciousIPFails(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{}
	runTest(t, false, expected, time.Time{}, "", errors.New("fail"), -1)
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
	runTest(t, false, expected, time.Time{}, "", nil, 0)
}

func runTest(t *testing.T, successful bool, expectedSecurityEvents []events.SuspiciousIPSecurityEvent,
	lastSuccessfulSearch time.Time, setHash string, err error, eventsErrorIdx int) {
	g := NewGomegaWithT(t)

	f := util.NewGlobalThreatFeedFromName("mock")
	suspiciousIP := &db.MockSuspicious{
		Error:                err,
		LastSuccessfulSearch: lastSuccessfulSearch,
		SetHash:              setHash,
	}
	for _, e := range expectedSecurityEvents {
		suspiciousIP.Events = append(suspiciousIP.Events, e)
	}
	eventsDB := &db.MockEvents{ErrorIndex: eventsErrorIdx, Events: []db.SecurityEventInterface{}}
	uut := NewSearcher(f, 0, suspiciousIP, eventsDB).(*searcher)
	feedCacher := &cacher.MockGlobalThreatFeedCache{}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	uut.doSearch(ctx, feedCacher)

	if successful {
		g.Expect(eventsDB.Events).Should(ConsistOf(expectedSecurityEvents), "Logs in DB should match expectedSecurityEvents")
	} else {
		if eventsErrorIdx >= 0 {
			g.Expect(eventsDB.Events).Should(HaveLen(len(expectedSecurityEvents)-1), "Logs in DB should have skipped 1 from input")
		} else {
			g.Expect(eventsDB.Events).Should(HaveLen(len(expectedSecurityEvents)), "DB should have all inputs")
		}
	}

	status := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed.Status
	g.Expect(status.LastSuccessfulSync).Should(BeNil(), "Sync should not be marked as successful")
	if successful {
		g.Expect(status.LastSuccessfulSearch.Time).ShouldNot(Equal(time.Time{}), "Search should be marked as successful")
		g.Expect(status.ErrorConditions).Should(HaveLen(0), "No errors should be reported")
	} else {
		g.Expect(status.LastSuccessfulSearch).Should(BeNil(), "Search should be not marked as successful")
		g.Expect(status.ErrorConditions).ShouldNot(HaveLen(0), "Errors should be reported")
	}
}

func TestFlowSearcher_SetFeed(t *testing.T) {
	g := NewGomegaWithT(t)

	f := util.NewGlobalThreatFeedFromName("mock")
	f2 := util.NewGlobalThreatFeedFromName("swap")
	suspiciousIP := &db.MockSuspicious{}
	eventsDB := &db.MockEvents{}
	searcher := NewSearcher(f, 0, suspiciousIP, eventsDB).(*searcher)

	searcher.SetFeed(f2)
	g.Expect(searcher.feed).Should(Equal(f2))
	g.Expect(searcher.feed).ShouldNot(BeIdenticalTo(f2))
}
