// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"text/template"
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

	p := NewPuller("test", xPack, e, &filters.NilFilter{}, map[int]*template.Template{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally(">", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(0))

	g.Expect(e.Events).Should(HaveLen(1))
	event := p.generateEvent(xPack.Records[0])
	g.Expect(e.Events[0]).Should(Equal(event))
}

func TestPull_RecordsFails(t *testing.T) {
	g := NewWithT(t)

	xPack := &elastic.MockXPack{
		Err: errors.New("fail"),
	}
	e := &db.MockEvents{
		ErrorIndex: -1,
	}

	p := NewPuller("test", xPack, e, &filters.NilFilter{}, map[int]*template.Template{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).Should(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally("==", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(1))
	g.Expect(st.Status().ErrorConditions[0].Type).Should(Equal(statser.XPackRecordsFailed))

	g.Expect(e.Events).Should(HaveLen(0))
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

	p := NewPuller("test", xPack, e, flt, map[int]*template.Template{}).(*puller)
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
		event := p.generateEvent(r)
		expected = append(expected, event)
	}
	g.Expect(e.Events).Should(ConsistOf(expected))
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

	p := NewPuller("test", xPack, e, flt, map[int]*template.Template{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).Should(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally("==", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(1))
	g.Expect(st.Status().ErrorConditions[0].Type).Should(Equal(statser.FilterFailed))

	g.Expect(e.Events).Should(HaveLen(0))
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

	p := NewPuller("test", xPack, e, &filters.NilFilter{}, map[int]*template.Template{}).(*puller)
	st := statser.NewStatser("test")

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	st.Run(ctx)

	err := p.pull(ctx, st)
	g.Expect(err).Should(HaveOccurred())

	g.Expect(st.Status().LastSuccessfulSync.Time).Should(BeTemporally("==", time.Time{}))
	g.Expect(st.Status().ErrorConditions).Should(HaveLen(1))
	g.Expect(st.Status().ErrorConditions[0].Type).Should(Equal(statser.StoreEventsFailed))

	g.Expect(e.Events).Should(HaveLen(0))
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

	p := NewPuller("test", xPack, &db.MockEvents{}, &filters.NilFilter{}, map[int]*template.Template{}).(*puller)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	rs, err := p.fetch(ctx)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(rs).Should(HaveLen(1))
}

func TestGenerateEvent(t *testing.T) {
	g := NewWithT(t)

	tmpl := "detector 1"
	detectors := map[int]*template.Template{
		1: template.Must(template.New("test").Parse(tmpl)),
	}
	p := NewPuller("test", &elastic.MockXPack{}, &db.MockEvents{}, &filters.NilFilter{}, detectors).(*puller)

	r1 := elastic.RecordSpec{
		DetectorIndex: 1,
		Id:            "test",
	}
	e1 := p.generateEvent(r1)

	g.Expect(e1.Description).Should(Equal(tmpl))
	g.Expect(e1.Severity).Should(BeNumerically(">=", 0))
	g.Expect(e1.Severity).Should(BeNumerically("<=", 100))
	g.Expect(e1.Record).Should(Equal(r1))

	r2 := elastic.RecordSpec{
		DetectorIndex: 0,
		Id:            "test2",
	}
	e2 := p.generateEvent(r2)

	g.Expect(e2.Description).Should(HaveSuffix(UnknownDetector))
	g.Expect(e2.Severity).Should(BeNumerically(">=", 0))
	g.Expect(e2.Severity).Should(BeNumerically("<=", 100))
	g.Expect(e2.Record).Should(Equal(r2))
}

func TestGenerateEventDescription(t *testing.T) {
	g := NewWithT(t)

	detectors := map[int]*template.Template{
		0: template.Must(template.New("test").Parse("{{.OverFieldName}}: {{.OverFieldValue}}")),
		1: template.Must(template.New("test").Parse("{{.PartitionFieldName}}: {{.PartitionFieldValue}}")),
	}
	p := NewPuller("test", &elastic.MockXPack{}, &db.MockEvents{}, &filters.NilFilter{}, detectors).(*puller)

	r := elastic.RecordSpec{
		OverFieldName:       "foo",
		OverFieldValue:      "bar",
		PartitionFieldName:  "baz",
		PartitionFieldValue: "bop",
	}
	g.Expect(p.generateEventDescription(r)).Should(Equal(fmt.Sprintf("%s: %s", r.OverFieldName, r.OverFieldValue)))

	r.DetectorIndex = 1
	g.Expect(p.generateEventDescription(r)).Should(Equal(fmt.Sprintf("%s: %s", r.PartitionFieldName, r.PartitionFieldValue)))
}
