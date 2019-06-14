// Copyright 2019 Tigera Inc. All rights reserved.

package filters

import (
	"context"
	"errors"
	"testing"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"

	. "github.com/onsi/gomega"
)

func TestFilters_NilFilter(t *testing.T) {
	g := NewWithT(t)

	in := []elastic.RecordSpec{
		{RecordScore: 100},
		{RecordScore: 90},
	}

	out, err := Filters{NilFilter{}}.Filter(context.TODO(), in)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(out).Should(Equal(in))
}

func TestFilters(t *testing.T) {
	g := NewWithT(t)

	in := []elastic.RecordSpec{
		{RecordScore: 100},
		{RecordScore: 90},
	}

	f1 := MockFilter{
		RS: []elastic.RecordSpec{
			{RecordScore: 90},
		},
	}

	f2 := MockFilter{
		RS: []elastic.RecordSpec{
			{RecordScore: 100},
		},
	}

	f3 := MockFilter{
		Err: errors.New("test"),
	}

	out, err := Filters{f1}.Filter(context.TODO(), in)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(out).Should(Equal(f1.RS))

	out, err = Filters{f2}.Filter(context.TODO(), in)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(out).Should(Equal(f2.RS))

	_, err = Filters{f3}.Filter(context.TODO(), in)
	g.Expect(err).Should(HaveOccurred())

	out, err = Filters{f1, f2}.Filter(context.TODO(), in)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(out).Should(Equal(f2.RS))

	out, err = Filters{f2, f1}.Filter(context.TODO(), in)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(out).Should(Equal(f1.RS))

	_, err = Filters{f1, f2, f3}.Filter(context.TODO(), in)
	g.Expect(err).Should(HaveOccurred())
	_, err = Filters{f1, f3, f2}.Filter(context.TODO(), in)
	g.Expect(err).Should(HaveOccurred())
	_, err = Filters{f3, f2, f1}.Filter(context.TODO(), in)
	g.Expect(err).Should(HaveOccurred())
}
