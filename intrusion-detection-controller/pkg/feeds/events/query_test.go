// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/storage"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func TestSuspiciousIP_Success(t *testing.T) {
	g := NewGomegaWithT(t)

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"

	logs := []v1.FlowLog{
		{
			SourceIP:      util.Sptr("1.2.3.4"),
			SourcePort:    util.I64ptr(333),
			SourceName:    "source",
			DestIP:        util.Sptr("2.3.4.5"),
			DestPort:      util.I64ptr(333),
			DestNamespace: "default",
			DestName:      "dest",
			DestType:      "wep",
		},
		{
			SourceIP:        util.Sptr("5.6.7.8"),
			SourcePort:      util.I64ptr(333),
			SourceName:      "source",
			SourceNamespace: "default",
			SourceType:      "wep",
			DestIP:          util.Sptr("2.3.4.5"),
			DestPort:        util.I64ptr(333),
			DestName:        "dest",
		},
	}
	i := &storage.MockIterator[v1.FlowLog]{
		ErrorIndex: -1,
		Values:     logs,
		Keys:       []storage.QueryKey{storage.QueryKeyFlowLogSourceIP, storage.QueryKeyFlowLogDestIP},
	}
	q := &storage.MockSetQuerier{IteratorFlow: i}
	uut := NewSuspiciousIP(q)

	expected := []v1.Event{
		{
			ID:            "test_0__1.2.3.4_333_2.3.4.5_333",
			Time:          v1.NewEventTimestamp(0),
			Description:   "suspicious IP 1.2.3.4 from list test connected to wep default/dest",
			Type:          SuspiciousFlow,
			Severity:      Severity,
			Origin:        testFeed.Name,
			SourceIP:      util.Sptr("1.2.3.4"),
			SourceName:    "source",
			DestIP:        util.Sptr("2.3.4.5"),
			DestName:      "dest",
			DestNamespace: "default",
			Record: v1.SuspiciousIPEventRecord{
				Feeds: []string{"test"},
			},
			DestPort:     util.I64ptr(333),
			SourcePort:   util.I64ptr(333),
			Name:         "test",
			AttackVector: "Network",
			MitreIDs:     &[]string{"T1190"},
			Mitigations:  &[]string{"Network policies working as expected"},
			MitreTactic:  "Initial Access",
		},
		{
			ID:              "test_0__5.6.7.8_333_2.3.4.5_333",
			Time:            v1.NewEventTimestamp(0),
			Description:     "wep default/source connected to suspicious IP 2.3.4.5 from list test",
			Type:            SuspiciousFlow,
			Severity:        Severity,
			Origin:          testFeed.Name,
			SourceIP:        util.Sptr("5.6.7.8"),
			SourceName:      "source",
			SourceNamespace: "default",
			SourcePort:      util.I64ptr(333),
			DestIP:          util.Sptr("2.3.4.5"),
			DestPort:        util.I64ptr(333),
			DestName:        "dest",
			Record: v1.SuspiciousIPEventRecord{
				Feeds: []string{"test"},
			},
			Name:         "test",
			AttackVector: "Network",
			MitreIDs:     &[]string{"T1190"},
			Mitigations:  &[]string{"Network policies working as expected"},
			MitreTactic:  "Initial Access",
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

	logs := []v1.FlowLog{
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
	i := &storage.MockIterator[v1.FlowLog]{
		Error:      errors.New("test"),
		ErrorIndex: 1,
		Values:     logs,
		Keys:       []storage.QueryKey{storage.QueryKeyFlowLogSourceIP, storage.QueryKeyFlowLogDestIP},
	}
	q := &storage.MockSetQuerier{IteratorFlow: i}
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

	q := &storage.MockSetQuerier{IteratorDNS: nil, QueryError: errors.New("query failed")}
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

	logs := []v1.DNSLog{
		{
			ID:    "id1",
			QName: "xx.yy.zzz",
		},
		{
			ID:    "id2",
			QName: "qq.rr.sss",
		},
		{
			ID:    "id1",
			QName: "aa.bb.ccc",
		},
	}
	i := &storage.MockIterator[v1.DNSLog]{
		ErrorIndex: -1,
		Values:     logs,
		Keys:       []storage.QueryKey{storage.QueryKeyDNSLogQName, storage.QueryKeyDNSLogQName, storage.QueryKeyDNSLogQName},
	}
	domains := storage.DomainNameSetSpec{
		"xx.yy.zzz",
		"qq.rr.sss",
		"aa.bb.ccc",
	}
	q := &storage.MockSetQuerier{IteratorDNS: i, Set: domains}
	uut := NewSuspiciousDomainNameSet(q)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"
	results, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(2))
	rec1, ok := results[0].Record.(v1.SuspiciousDomainEventRecord)
	g.Expect(ok).Should(BeTrue())
	rec2, ok := results[1].Record.(v1.SuspiciousDomainEventRecord)
	g.Expect(ok).Should(BeTrue())
	g.Expect(rec1.SuspiciousDomains).To(Equal([]string{"xx.yy.zzz"}))
	g.Expect(rec2.SuspiciousDomains).To(Equal([]string{"qq.rr.sss"}))
}

func TestSuspiciousDomain_IterationFails(t *testing.T) {
	g := NewGomegaWithT(t)

	logs := []v1.DNSLog{
		{
			ID:    "id1",
			QName: "xx.yy.zzz",
		},
	}
	i := &storage.MockIterator[v1.DNSLog]{
		Error:      errors.New("iteration failed"),
		ErrorIndex: 0,
		Values:     logs,
		Keys:       []storage.QueryKey{storage.QueryKeyDNSLogQName},
	}
	domains := storage.DomainNameSetSpec{
		"xx.yy.zzz",
		"qq.rr.sss",
		"aa.bb.ccc",
	}
	q := &storage.MockSetQuerier{IteratorDNS: i, Set: domains}
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

	q := &storage.MockSetQuerier{GetError: errors.New("get failed")}
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

	domains := storage.DomainNameSetSpec{
		"xx.yy.zzz",
		"qq.rr.sss",
		"aa.bb.ccc",
	}
	q := &storage.MockSetQuerier{Set: domains, QueryError: errors.New("query failed")}
	uut := NewSuspiciousDomainNameSet(q)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testFeed := &apiV3.GlobalThreatFeed{}
	testFeed.Name = "test"
	results, _, _, err := uut.QuerySet(ctx, testFeed)
	g.Expect(err).To(Equal(errors.New("query failed")))
	g.Expect(results).To(HaveLen(0))
}
