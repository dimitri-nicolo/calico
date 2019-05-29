// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/anomaly/filters"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestFetch(t *testing.T) {
	g := NewWithT(t)

	xPack := &elastic.MockXPack{
		Records: []elastic.RecordSpec{
			{
				RecordScore: 100,
			},
		},
	}

	p := NewPuller("test", xPack, &db.MockEvents{}, &filters.NilFilter{}).(*puller)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	rs, err := p.fetch(ctx)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(rs).Should(HaveLen(1))
}
