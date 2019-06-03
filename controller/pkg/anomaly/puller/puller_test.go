// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/anomaly/events"
	"github.com/tigera/intrusion-detection/controller/pkg/anomaly/filters"
	"github.com/tigera/intrusion-detection/controller/pkg/anomaly/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestPull(t *testing.T) {
	g := NewWithT(t)

	xPack := &elastic.MockXPack{
		Records: []elastic.RecordSpec{
			{
				Timestamp:   elastic.Time{time.Now()},
				RecordScore: 100,
			},
		},
	}
	e := &db.MockEvents{
		ErrorIndex: -1,
	}

	p := NewPuller("test", xPack, e, &filters.NilFilter{}, "", map[int]string{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally(">", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(0))

	g.Expect(e.FlowLogs).Should(HaveLen(1))
	g.Expect(e.FlowLogs[0]).Should(Equal(p.generateEvent(xPack.Records[0])))
}

func TestPull_RecordsFails(t *testing.T) {
	g := NewWithT(t)

	xPack := &elastic.MockXPack{
		Err: errors.New("fail"),
	}
	e := &db.MockEvents{
		ErrorIndex: -1,
	}

	p := NewPuller("test", xPack, e, &filters.NilFilter{}, "", map[int]string{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).Should(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally("==", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(1))
	g.Expect(st.Status().ErrorConditions[0].Type).Should(Equal(statser.XPackRecordsFailed))

	g.Expect(e.FlowLogs).Should(HaveLen(0))
}

func TestPull_Filter(t *testing.T) {
	g := NewWithT(t)

	xPack := &elastic.MockXPack{
		Records: []elastic.RecordSpec{
			{
				Timestamp:   elastic.Time{time.Now()},
				RecordScore: 100,
			},
		},
	}
	e := &db.MockEvents{
		ErrorIndex: -1,
	}
	flt := filters.MockFilter{
		RS: []elastic.RecordSpec{
			{RecordScore: 1},
			{RecordScore: 2},
		},
	}

	p := NewPuller("test", xPack, e, flt, "", map[int]string{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally(">", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(0))

	expected := []events.XPackSecurityEvent{}
	for _, r := range flt.RS {
		expected = append(expected, p.generateEvent(r))
	}
	g.Expect(e.FlowLogs).Should(ConsistOf(expected))
}

func TestPull_FilterFails(t *testing.T) {
	g := NewWithT(t)

	xPack := &elastic.MockXPack{
		Records: []elastic.RecordSpec{
			{
				Timestamp:   elastic.Time{time.Now()},
				RecordScore: 100,
			},
		},
	}
	e := &db.MockEvents{}
	flt := filters.MockFilter{
		Err: errors.New("fail"),
	}

	p := NewPuller("test", xPack, e, flt, "", map[int]string{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).Should(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally("==", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(1))
	g.Expect(st.Status().ErrorConditions[0].Type).Should(Equal(statser.FilterFailed))

	g.Expect(e.FlowLogs).Should(HaveLen(0))
}

func TestPull_PutFails(t *testing.T) {
	g := NewWithT(t)

	xPack := &elastic.MockXPack{
		Records: []elastic.RecordSpec{
			{
				Timestamp:   elastic.Time{time.Now()},
				RecordScore: 100,
			},
		},
	}
	e := &db.MockEvents{}

	p := NewPuller("test", xPack, e, &filters.NilFilter{}, "", map[int]string{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).Should(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally("==", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(1))
	g.Expect(st.Status().ErrorConditions[0].Type).Should(Equal(statser.StoreEventsFailed))

	g.Expect(e.FlowLogs).Should(HaveLen(0))
}

func TestFetch(t *testing.T) {
	g := NewWithT(t)

	xPack := &elastic.MockXPack{
		Records: []elastic.RecordSpec{
			{
				RecordScore: 100,
			},
		},
	}

	p := NewPuller("test", xPack, &db.MockEvents{}, &filters.NilFilter{}, "", map[int]string{}).(*puller)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	rs, err := p.fetch(ctx)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(rs).Should(HaveLen(1))
}

func TestGenerateEvent(t *testing.T) {
	g := NewWithT(t)

	description := "test anomaly detector"
	detectors := map[int]string{
		1: "detector 1",
	}
	p := NewPuller("test", &elastic.MockXPack{}, &db.MockEvents{}, &filters.NilFilter{}, description, detectors).(*puller)

	r1 := elastic.RecordSpec{
		DetectorIndex: 1,
		Id:            "test",
	}
	e1 := p.generateEvent(r1)

	g.Expect(e1.Description).Should(HavePrefix(description))
	g.Expect(e1.Description).Should(HaveSuffix(detectors[1]))
	g.Expect(e1.Severity).Should(BeNumerically(">=", 0))
	g.Expect(e1.Severity).Should(BeNumerically("<=", 100))
	g.Expect(e1.Record).Should(Equal(r1))

	r2 := elastic.RecordSpec{
		DetectorIndex: 0,
		Id:            "test2",
	}
	e2 := p.generateEvent(r2)

	g.Expect(e2.Description).Should(HavePrefix(description))
	g.Expect(e2.Description).Should(HaveSuffix(UnknownDetector))
	g.Expect(e2.Severity).Should(BeNumerically(">=", 0))
	g.Expect(e2.Severity).Should(BeNumerically("<=", 100))
	g.Expect(e2.Record).Should(Equal(r2))
}
