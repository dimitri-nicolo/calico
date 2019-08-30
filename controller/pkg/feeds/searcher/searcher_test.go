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
	runTest(t, true, expected, nil, -1)
}

// TestDoIPSetNoResults tests the case where no results are returned
func TestDoIPSetNoResults(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{}
	runTest(t, true, expected, nil, -1)
}

// TestDoIPSetSuspiciousIPFails tests the case where suspiciousIP fails after the first result
func TestDoIPSetSuspiciousIPFails(t *testing.T) {
	expected := []events.SuspiciousIPSecurityEvent{}
	runTest(t, false, expected, errors.New("fail"), -1)
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
	runTest(t, false, expected, nil, 0)
}

func runTest(t *testing.T, successful bool, expected []events.SuspiciousIPSecurityEvent, err error, eventsErrorIdx int) {
	g := NewGomegaWithT(t)

	f := util.NewGlobalThreatFeedFromName("mock")
	suspiciousIP := &db.MockSuspicious{Error: err}
	for _, e := range expected {
		suspiciousIP.Hits = append(suspiciousIP.Hits, e)
	}
	eventsDB := &db.MockEvents{ErrorIndex: eventsErrorIdx, Events: []db.SecurityEventInterface{}}
	uut := NewSearcher(f, 0, suspiciousIP, eventsDB).(*searcher)
	s := &statser.MockStatser{}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	uut.doSearch(ctx, s)

	if successful {
		g.Expect(eventsDB.Events).Should(ConsistOf(expected), "Logs in DB should match expected")
	} else {
		if eventsErrorIdx >= 0 {
			g.Expect(eventsDB.Events).Should(HaveLen(len(expected)-1), "Logs in DB should have skipped 1 from input")
		} else {
			g.Expect(eventsDB.Events).Should(HaveLen(len(expected)), "DB should have all inputs")
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
	suspiciousIP := &db.MockSuspicious{}
	eventsDB := &db.MockEvents{}
	searcher := NewSearcher(f, 0, suspiciousIP, eventsDB).(*searcher)

	searcher.SetFeed(f2)
	g.Expect(searcher.feed).Should(Equal(f2))
	g.Expect(searcher.feed).ShouldNot(BeIdenticalTo(f2))
}

//// TestDomainNameSetFilter tests filtering out repeated events with the same
//// DNS Index and ID
//func TestDomainNameSetFilter(t *testing.T) {
//	found := []db.SecurityEventInterface{
//		events.SuspiciousDomainSecurityEvent{
//			DNSLogIndex:      "tigera_secure_ee_dns.cluster.20190827",
//			DNSLogID:         "1",
//			SourceIP:         util.Sptr("1.2.3.4"),
//			SourceName:       "source",
//			SourceNamespace:  "default",
//			SuspiciousDomain: util.Sptr("xx.yy.zzz"),
//		},
//		events.SuspiciousDomainSecurityEvent{
//			DNSLogIndex:      "tigera_secure_ee_dns.cluster.20190827",
//			DNSLogID:         "2",
//			SourceIP:         util.Sptr("5.6.7.8"),
//			SourceName:       "source",
//			SourceNamespace:  "default",
//			SuspiciousDomain: util.Sptr("qq.ww.tt"),
//		},
//		// Repeated Index/ID
//		events.SuspiciousDomainSecurityEvent{
//			DNSLogIndex:      "tigera_secure_ee_dns.cluster.20190827",
//			DNSLogID:         "2",
//			SourceIP:         util.Sptr("5.6.7.8"),
//			SourceName:       "source",
//			SourceNamespace:  "default",
//			SuspiciousDomain: util.Sptr("tt.rr.sss"),
//		},
//	}
//	expected := []db.SecurityEventInterface{
//		events.SuspiciousDomainSecurityEvent{
//			DNSLogIndex:      "tigera_secure_ee_dns.cluster.20190827",
//			DNSLogID:         "1",
//			SourceIP:         util.Sptr("1.2.3.4"),
//			SourceName:       "source",
//			SourceNamespace:  "default",
//			SuspiciousDomain: util.Sptr("xx.yy.zzz"),
//		},
//		events.SuspiciousDomainSecurityEvent{
//			DNSLogIndex:      "tigera_secure_ee_dns.cluster.20190827",
//			DNSLogID:         "2",
//			SourceIP:         util.Sptr("5.6.7.8"),
//			SourceName:       "source",
//			SourceNamespace:  "default",
//			SuspiciousDomain: util.Sptr("qq.ww.tt"),
//		},
//	}
//	g := NewGomegaWithT(t)
//
//	f := util.NewGlobalThreatFeedFromName("mock")
//	suspicious := &db.MockSuspicious{ErrorIndex: -1}
//	suspicious.Hits = append(suspicious.Hits, found...)
//	eventsDB := &db.MockEvents{ErrorIndex: -1}
//	srchr := NewDNSSearcher(f, 0, suspicious, eventsDB).(*searcher)
//	s := &statser.MockStatser{}
//
//	ctx, cancel := context.WithCancel(context.TODO())
//	defer cancel()
//
//	srchr.doSearch(ctx, s)
//
//	g.Expect(eventsDB.Events).Should(ConsistOf(expected), "Logs in DB should match expected")
//	g.Expect(suspicious.Hits).Should(HaveLen(0), "All flowLogs from suspicious were consumed")
//
//	status := s.Status()
//	g.Expect(status.LastSuccessfulSync.Time).Should(Equal(time.Time{}), "Sync should not be marked as successful")
//	g.Expect(status.LastSuccessfulSearch.Time).ShouldNot(Equal(time.Time{}), "Search should be marked as successful")
//	g.Expect(status.ErrorConditions).Should(HaveLen(0), "No errors should be reported")
//
//	// Repeated search should *not* filter logs from last time
//	suspicious.Hits = append(suspicious.Hits, found...)
//	srchr.doSearch(ctx, s)
//	// Should get two copies of expected pushed to db.  (In the real Elasticsearch, the fact
//	// that they have the same document ID will make repeated puts idempotent.)
//	g.Expect(eventsDB.Events).Should(ConsistOf(append(expected, expected...)))
//	g.Expect(suspicious.Hits).Should(HaveLen(0), "All flowLogs from suspicious were consumed")
//}
