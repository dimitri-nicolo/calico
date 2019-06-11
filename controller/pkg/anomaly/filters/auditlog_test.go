// Copyright 2019 Tigera Inc. All rights reserved.

package filters

import (
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestAuditKey_String(t *testing.T) {
	g := NewWithT(t)

	k1 := auditKey{time.Now(), 0, "a", "b", "c/d"}
	k2 := auditKey{k1.timestamp, 0, "a", "b/c", "d"}
	k3 := auditKey{k1.timestamp, 1, k1.kind, k1.namespace, k1.name}
	k4 := auditKey{k1.timestamp.Add(time.Second * 1), 1, k1.kind, k1.namespace, k1.name}

	g.Expect(k1.String()).Should(Equal(k1.String()))
	g.Expect(k2.String()).Should(Equal(k2.String()))
	g.Expect(k1.String()).ShouldNot(Equal(k2.String()))
	g.Expect(k1.String()).ShouldNot(Equal(k3.String()))
	g.Expect(k3.String()).ShouldNot(Equal(k4.String()))
	g.Expect(k1.String()).ShouldNot(Equal(k4.String()))
}

func TestGetInfluencer(t *testing.T) {
	g := NewWithT(t)

	r := elastic.RecordSpec{
		Influencers: []elastic.InfluencerSpec{
			{"a", []interface{}{}},
			{"b", []interface{}{"x"}},
			{"c", []interface{}{1, "y"}},
			{"d", []interface{}{}},
			{"d", []interface{}{"z"}},
		},
	}

	g.Expect(getInfluencer(r, "a")).Should(Equal(""))
	g.Expect(getInfluencer(r, "b")).Should(Equal("x"))
	g.Expect(getInfluencer(r, "c")).Should(Equal("y"))
	g.Expect(getInfluencer(r, "d")).Should(Equal("z"))
}

func TestGetNamespace(t *testing.T) {
	g := NewWithT(t)

	r := elastic.RecordSpec{
		Influencers: []elastic.InfluencerSpec{
			{"source_namespace", []interface{}{"x"}},
			{"source_name", []interface{}{"a"}},
			{"dest_namespace", []interface{}{"y"}},
			{"dest_name", []interface{}{"b"}},
		},
	}

	g.Expect(getNamespace(r, "source_name")).Should(Equal("x"))
	g.Expect(getNamespace(r, "source_name_aggr")).Should(Equal("x"))
	g.Expect(getNamespace(r, "dest_name")).Should(Equal("y"))
	g.Expect(getNamespace(r, "dest_name_aggr")).Should(Equal("y"))
	g.Expect(getNamespace(r, "abc")).Should(Equal(AnyNamespace))

	r.Influencers = r.Influencers[:2]

	g.Expect(getNamespace(r, "source_name")).Should(Equal("x"))
	g.Expect(getNamespace(r, "source_name_aggr")).Should(Equal("x"))
	g.Expect(getNamespace(r, "dest_name")).Should(Equal(AnyNamespace))
	g.Expect(getNamespace(r, "dest_name_aggr")).Should(Equal(AnyNamespace))
}

func TestGetAuditKeys(t *testing.T) {
	g := NewWithT(t)

	r := elastic.RecordSpec{
		PartitionFieldName:  "source_name",
		PartitionFieldValue: "a",
		OverFieldName:       "dest_name",
		OverFieldValue:      "b",
		ByFieldName:         "source_name_aggr",
		ByFieldValue:        "c",
		Influencers: []elastic.InfluencerSpec{
			{"source_namespace", []interface{}{"x"}},
			{"dest_namespace", []interface{}{"y"}},
		},
		Timestamp:  elastic.Time{time.Now()},
		BucketSpan: 60,
	}

	g.Expect(getAuditKeys(r)).Should(ConsistOf(
		auditKey{r.Timestamp.Time, r.BucketSpan, Pod, "x", "a"},
		auditKey{r.Timestamp.Time, r.BucketSpan, Pod, "y", "b"},
		auditKey{r.Timestamp.Time, r.BucketSpan, ReplicaSet, "x", "c"},
	))

	r.PartitionFieldName = "dest_name_aggr"
	r.OverFieldName = ""
	r.ByFieldName = ""
	g.Expect(getAuditKeys(r)).Should(ConsistOf(
		auditKey{r.Timestamp.Time, r.BucketSpan, ReplicaSet, "y", "a"},
	))

	r.PartitionFieldName = ""
	r.OverFieldName = "source_ip"
	r.ByFieldName = "dest_ip"
	g.Expect(getAuditKeys(r)).Should(HaveLen(0))
}

type filterMockAuditLog struct {
	db.MockAuditLog
	returnTrueAfter  bool
	trueAfter        int
	returnTrueBefore bool
	trueBefore       int
}

func (al *filterMockAuditLog) ObjectCreatedBetween(kind, namespace, name string, before, after time.Time) (bool, error) {
	if al.returnTrueAfter {
		al.trueAfter--
		if al.trueAfter < 0 {
			return true, nil
		}
	}
	if al.returnTrueBefore {
		al.trueBefore--
		if al.trueBefore >= 0 {
			return true, nil
		}
	}
	return al.MockAuditLog.ObjectCreatedBetween(kind, namespace, name, before, after)
}

func (al *filterMockAuditLog) ObjectDeletedBetween(kind, namespace, name string, before, after time.Time) (bool, error) {
	return al.MockAuditLog.ObjectDeletedBetween(kind, namespace, name, before, after)
}

func TestAuditLog_Filter(t *testing.T) {
	g := NewWithT(t)

	al := &filterMockAuditLog{MockAuditLog: db.MockAuditLog{}}
	f := NewAuditLog(al).(*auditLogFilter)

	in := []elastic.RecordSpec{
		{
			ByFieldName:  "source_name",
			ByFieldValue: "a",
			Influencers: []elastic.InfluencerSpec{
				{"source_namespace", []interface{}{"x"}},
			},
		},
		{
			ByFieldName:  "dest_name_aggr",
			ByFieldValue: "b",
			Influencers: []elastic.InfluencerSpec{
				{"source_namespace", []interface{}{"y"}},
			},
		},
		{
			ByFieldName:  "dest_name",
			ByFieldValue: "c",
			Influencers: []elastic.InfluencerSpec{
				{"source_namespace", []interface{}{"z"}},
			},
		},
	}

	// Test that errors are returned correctly
	al.CreatedErr = errors.New("error")
	_, err := f.Filter(in)
	g.Expect(err).Should(HaveOccurred())
	al.CreatedErr = nil

	// Test with a filter that passes everything
	out, err := f.Filter(in)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(out).Should(ConsistOf(in))

	// Test with a filter that passes only the first two
	al.returnTrueAfter = true
	al.trueAfter = 2
	out, err = f.Filter(in)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(out).Should(ConsistOf(in[:2]))

	// Test with a filter that passes only the last
	al.returnTrueAfter = false
	al.returnTrueBefore = true
	al.trueBefore = 2
	out, err = f.Filter(in)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(out).Should(ConsistOf(in[2:]))

}
