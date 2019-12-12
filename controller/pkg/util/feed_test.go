// Copyright 2019 Tigera Inc. All rights reserved.

package util

import (
	"testing"
	"time"

	v32 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/gomega"
)

var (
	testGlobalThreatFeed = v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "IPSet",
			GlobalNetworkSet: &v32.GlobalNetworkSetSync{
				Labels: map[string]string{
					"level": "high",
				},
			},
			Pull: &v32.Pull{
				Period: "12h",
				HTTP: &v32.HTTPPull{
					Format: v32.ThreatFeedFormat{NewlineDelimited: &v32.ThreatFeedFormatNewlineDelimited{}},
					URL:    "http://mock.feed/v1",
				},
			},
		},
	}
)

// TODO test FeedNeedsRestart

func TestParseFeedDuration(t *testing.T) {
	g := NewGomegaWithT(t)

	period := ParseFeedDuration(&testGlobalThreatFeed)

	g.Expect(period).Should(BeNumerically("==", 12*time.Hour))
}

func TestParseFeedDurationInvalidPeriod(t *testing.T) {
	g := NewGomegaWithT(t)

	f := testGlobalThreatFeed.DeepCopy()
	f.Spec.Pull.Period = "h"

	period := ParseFeedDuration(f)
	g.Expect(period).Should(BeNumerically("==", v32.DefaultPullPeriod))
}

func TestParseFeedDurationEmptyPeriod(t *testing.T) {
	g := NewGomegaWithT(t)

	f := testGlobalThreatFeed.DeepCopy()
	f.Spec.Pull.Period = ""

	period := ParseFeedDuration(f)

	g.Expect(period).Should(BeNumerically("==", v32.DefaultPullPeriod))
}

func TestParseFeedDurationNilPull(t *testing.T) {
	g := NewGomegaWithT(t)

	f := testGlobalThreatFeed.DeepCopy()
	f.Spec.Pull = nil

	g.Expect(func() { ParseFeedDuration(f) }).Should(Panic())
}

func TestFeedNeedsRestart(t *testing.T) {
	g := NewGomegaWithT(t)

	a := testGlobalThreatFeed.DeepCopy()
	a.Spec.Pull = nil
	b := testGlobalThreatFeed.DeepCopy()
	b.Spec.Pull.Period = "24h"
	c := testGlobalThreatFeed.DeepCopy()
	c.Spec.Pull.Period = "12h"
	d := testGlobalThreatFeed.DeepCopy()
	d.Spec.Pull.Period = ""

	g.Expect(FeedNeedsRestart(a, a)).Should(BeFalse())
	g.Expect(FeedNeedsRestart(a, b)).Should(BeTrue())
	g.Expect(FeedNeedsRestart(b, a)).Should(BeTrue())
	g.Expect(FeedNeedsRestart(b, b)).Should(BeFalse())
	g.Expect(FeedNeedsRestart(b, c)).Should(BeTrue())
	g.Expect(FeedNeedsRestart(c, b)).Should(BeTrue())
	g.Expect(FeedNeedsRestart(b, d)).Should(BeFalse())
	g.Expect(FeedNeedsRestart(d, b)).Should(BeFalse())
	g.Expect(FeedNeedsRestart(c, d)).Should(BeTrue())
	g.Expect(FeedNeedsRestart(d, c)).Should(BeTrue())
}
