// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	oElastic "github.com/olivere/elastic/v7"
	. "github.com/onsi/gomega"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/calico/intrusion-detection/controller/pkg/db"
	"github.com/projectcalico/calico/intrusion-detection/controller/pkg/elastic"
	"github.com/projectcalico/calico/intrusion-detection/controller/pkg/util"
	"github.com/projectcalico/calico/lma/pkg/api"
)

func TestSuspiciousIP_Success(t *testing.T) {
	g := NewGomegaWithT(t)

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"

	logs := []FlowLogJSONOutput{
		{
			SourceIP:      util.Sptr("1.2.3.4"),
			SourceName:    "source",
			DestIP:        util.Sptr("2.3.4.5"),
			DestNamespace: "default",
			DestName:      "dest",
			DestType:      "wep",
		},
		{
			SourceIP:        util.Sptr("5.6.7.8"),
			SourceName:      "source",
			SourceNamespace: "default",
			SourceType:      "wep",
			DestIP:          util.Sptr("2.3.4.5"),
			DestName:        "dest",
		},
	}
	var hits []*oElastic.SearchHit
	for _, l := range logs {
		h := &oElastic.SearchHit{}
		src, err := json.Marshal(&l)
		g.Expect(err).ToNot(HaveOccurred())
		raw := json.RawMessage(src)
		h.Source = raw
		hits = append(hits, h)
	}
	// Add an extra malformed hit, which is ignored in results
	junk := json.RawMessage([]byte("{"))
	hits = append(hits, &oElastic.SearchHit{Source: junk})
	i := &elastic.MockIterator{
		ErrorIndex: -1,
		Hits:       hits,
		Keys:       []db.QueryKey{db.QueryKeyFlowLogSourceIP, db.QueryKeyFlowLogDestIP, db.QueryKeyUnknown}}
	q := &elastic.MockSetQuerier{Iterator: i}
	uut := NewSuspiciousIP(q)

	expected := []db.SecurityEventInterface{
		SuspiciousIPSecurityEvent{
			EventsData: api.EventsData{
				Description:   "suspicious IP 1.2.3.4 from list test connected to wep default/dest",
				Type:          SuspiciousFlow,
				Severity:      Severity,
				Origin:        testFeed.Name,
				SourceIP:      util.Sptr("1.2.3.4"),
				SourceName:    "source",
				DestIP:        util.Sptr("2.3.4.5"),
				DestName:      "dest",
				DestNamespace: "default",
				Record: SuspiciousIPEventRecord{
					Feeds: []string{"test"},
				},
			},
		},
		SuspiciousIPSecurityEvent{
			EventsData: api.EventsData{
				Description:     "wep default/source connected to suspicious IP 2.3.4.5 from list test",
				Type:            SuspiciousFlow,
				Severity:        Severity,
				Origin:          testFeed.Name,
				SourceIP:        util.Sptr("5.6.7.8"),
				SourceName:      "source",
				SourceNamespace: "default",
				DestIP:          util.Sptr("2.3.4.5"),
				DestName:        "dest",
				Record: SuspiciousIPEventRecord{
					Feeds: []string{"test"},
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	results, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(Equal(expected))
}

func TestSuspiciousIP_IterationFails(t *testing.T) {
	g := NewGomegaWithT(t)

	logs := []FlowLogJSONOutput{
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
	var hits []*oElastic.SearchHit
	for _, l := range logs {
		h := &oElastic.SearchHit{}
		src, err := json.Marshal(&l)
		g.Expect(err).ToNot(HaveOccurred())
		raw := json.RawMessage(src)
		h.Source = raw
		hits = append(hits, h)
	}
	i := &elastic.MockIterator{
		Error:      errors.New("test"),
		ErrorIndex: 1,
		Hits:       hits,
		Keys:       []db.QueryKey{db.QueryKeyFlowLogSourceIP, db.QueryKeyFlowLogDestIP}}
	q := &elastic.MockSetQuerier{Iterator: i}
	uut := NewSuspiciousIP(q)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"
	_, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).To(Equal(errors.New("test")))
}

func TestSuspiciousIP_QueryFails(t *testing.T) {
	g := NewGomegaWithT(t)

	q := &elastic.MockSetQuerier{Iterator: nil, QueryError: errors.New("query failed")}
	uut := NewSuspiciousIP(q)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"
	_, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).To(Equal(errors.New("query failed")))
}

func TestSuspiciousDomain_Success(t *testing.T) {
	g := NewGomegaWithT(t)

	hits := []*oElastic.SearchHit{
		{
			Index: "idx1",
			Id:    "id1",
		},
		{
			Index: "idx1",
			Id:    "id2",
		},
		// repeat
		{
			Index: "idx1",
			Id:    "id1",
		},
	}
	logs := []DNSLog{
		{
			QName: "xx.yy.zzz",
		},
		{
			QName: "qq.rr.sss",
		},
		{
			QName: "aa.bb.ccc",
		},
	}
	for i, l := range logs {
		h := hits[i]
		src, err := json.Marshal(&l)
		g.Expect(err).ToNot(HaveOccurred())
		raw := json.RawMessage(src)
		h.Source = raw
	}
	// Add an extra malformed hit, which is ignored in results
	junk := json.RawMessage([]byte("{"))
	hits = append(hits, &oElastic.SearchHit{Index: "junk", Id: "junk", Source: junk})
	i := &elastic.MockIterator{
		ErrorIndex: -1,
		Hits:       hits,
		Keys:       []db.QueryKey{db.QueryKeyDNSLogQName, db.QueryKeyDNSLogQName, db.QueryKeyDNSLogQName, db.QueryKeyUnknown}}
	domains := db.DomainNameSetSpec{
		"xx.yy.zzz",
		"qq.rr.sss",
		"aa.bb.ccc",
	}
	q := &elastic.MockSetQuerier{Iterator: i, Set: domains}
	uut := NewSuspiciousDomainNameSet(q)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"
	results, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(2))
	rec1, ok := results[0].GetEventsData().Record.(SuspiciousDomainEventRecord)
	g.Expect(ok).Should(BeTrue())
	rec2, ok := results[1].GetEventsData().Record.(SuspiciousDomainEventRecord)
	g.Expect(ok).Should(BeTrue())
	g.Expect(rec1.SuspiciousDomains).To(Equal([]string{"xx.yy.zzz"}))
	g.Expect(rec2.SuspiciousDomains).To(Equal([]string{"qq.rr.sss"}))
}

func TestSuspiciousDomain_IterationFails(t *testing.T) {
	g := NewGomegaWithT(t)

	hits := []*oElastic.SearchHit{
		{
			Index: "idx1",
			Id:    "id1",
		},
	}
	logs := []DNSLog{
		{
			QName: "xx.yy.zzz",
		},
	}
	for i, l := range logs {
		h := hits[i]
		src, err := json.Marshal(&l)
		g.Expect(err).ToNot(HaveOccurred())
		raw := json.RawMessage(src)
		h.Source = raw
	}
	i := &elastic.MockIterator{
		Error:      errors.New("iteration failed"),
		ErrorIndex: 0,
		Hits:       hits,
		Keys:       []db.QueryKey{db.QueryKeyDNSLogQName}}
	domains := db.DomainNameSetSpec{
		"xx.yy.zzz",
		"qq.rr.sss",
		"aa.bb.ccc",
	}
	q := &elastic.MockSetQuerier{Iterator: i, Set: domains}
	uut := NewSuspiciousDomainNameSet(q)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"
	results, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).To(Equal(errors.New("iteration failed")))
	g.Expect(results).To(HaveLen(0))
}

func TestSuspiciousDomain_GetFails(t *testing.T) {
	g := NewGomegaWithT(t)

	q := &elastic.MockSetQuerier{GetError: errors.New("get failed")}
	uut := NewSuspiciousDomainNameSet(q)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"
	results, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).To(Equal(errors.New("get failed")))
	g.Expect(results).To(HaveLen(0))
}

func TestSuspiciousDomain_QueryFails(t *testing.T) {
	g := NewGomegaWithT(t)

	domains := db.DomainNameSetSpec{
		"xx.yy.zzz",
		"qq.rr.sss",
		"aa.bb.ccc",
	}
	q := &elastic.MockSetQuerier{Set: domains, QueryError: errors.New("query failed")}
	uut := NewSuspiciousDomainNameSet(q)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"
	results, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).To(Equal(errors.New("query failed")))
	g.Expect(results).To(HaveLen(0))
}
