// Copyright 2019 Tigera Inc. All rights reserved.

package util

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testGlobalThreatFeed = &v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: FeedsNamespace,
		},
		Spec: v3.GlobalThreatFeedSpec{
			Content: "IPSet",
			GlobalNetworkSet: &v3.GlobalNetworkSetSync{
				Labels: map[string]string{
					"level": "high",
				},
			},
			Pull: &v3.Pull{
				Period: "12h",
				HTTP: &v3.HTTPPull{
					Format: v3.ThreatFeedFormat{NewlineDelimited: &v3.ThreatFeedFormatNewlineDelimited{}},
					URL:    "http://mock.feed/v1",
				},
			},
		},
	}
)

// TODO test FeedNeedsRestart

func TestParseFeedDuration(t *testing.T) {
	g := NewGomegaWithT(t)

	period := ParseFeedDuration(testGlobalThreatFeed)

	g.Expect(period).Should(BeNumerically("==", 12*time.Hour))
}

func TestParseFeedDurationInvalidPeriod(t *testing.T) {
	g := NewGomegaWithT(t)

	f := testGlobalThreatFeed.DeepCopy()
	f.Spec.Pull.Period = "h"

	period := ParseFeedDuration(f)
	g.Expect(period).Should(BeNumerically("==", v3.DefaultPullPeriod))
}

func TestParseFeedDurationEmptyPeriod(t *testing.T) {
	g := NewGomegaWithT(t)

	f := testGlobalThreatFeed.DeepCopy()
	f.Spec.Pull.Period = ""

	period := ParseFeedDuration(f)

	g.Expect(period).Should(BeNumerically("==", v3.DefaultPullPeriod))
}

func TestParseFeedDurationNilPull(t *testing.T) {
	g := NewGomegaWithT(t)

	f := testGlobalThreatFeed.DeepCopy()
	f.Spec.Pull = nil

	g.Expect(func() { ParseFeedDuration(f) }).Should(Panic())
}

func TestFeedNeedsRestart(t *testing.T) {
	g := NewGomegaWithT(t)

	noPull := testGlobalThreatFeed.DeepCopy()
	noPull.Spec.Pull = nil
	period24h := testGlobalThreatFeed.DeepCopy()
	period24h.Spec.Pull.Period = "24h"
	period12h := testGlobalThreatFeed.DeepCopy()
	period12h.Spec.Pull.Period = "12h"
	periodEmpty := testGlobalThreatFeed.DeepCopy()
	periodEmpty.Spec.Pull.Period = ""
	noHTTP := testGlobalThreatFeed.DeepCopy()
	noHTTP.Spec.Pull.HTTP = nil
	emptyFormat := testGlobalThreatFeed.DeepCopy()
	emptyFormat.Spec.Pull.HTTP.Format.NewlineDelimited = nil
	jsonFormat := emptyFormat.DeepCopy()
	jsonFormat.Spec.Pull.HTTP.Format.JSON = &v3.ThreatFeedFormatJSON{Path: "$."}
	jsonFormat2 := emptyFormat.DeepCopy()
	jsonFormat2.Spec.Pull.HTTP.Format.JSON = &v3.ThreatFeedFormatJSON{Path: "$.foo"}
	csvFormat := emptyFormat.DeepCopy()
	csvFormat.Spec.Pull.HTTP.Format.CSV = &v3.ThreatFeedFormatCSV{FieldNum: UintPtr(1)}
	csvFormat2 := emptyFormat.DeepCopy()
	csvFormat2.Spec.Pull.HTTP.Format.CSV = &v3.ThreatFeedFormatCSV{FieldNum: UintPtr(2)}
	noGlobalNetworkSet := testGlobalThreatFeed.DeepCopy()
	noGlobalNetworkSet.Spec.GlobalNetworkSet = nil

	g.Expect(FeedNeedsRestart(noPull, noPull)).Should(BeFalse(), "two push feeds")
	g.Expect(FeedNeedsRestart(noPull, period24h)).Should(BeTrue(), "24h period vs no pull")
	g.Expect(FeedNeedsRestart(period24h, noPull)).Should(BeTrue(), "No pull section vs 24h period")
	g.Expect(FeedNeedsRestart(period24h, period24h)).Should(BeFalse(), "24h period with its duplicate")
	g.Expect(FeedNeedsRestart(period24h, period12h)).Should(BeTrue(), "Differing periods")
	g.Expect(FeedNeedsRestart(period12h, period24h)).Should(BeTrue(), "Differing periods")
	g.Expect(FeedNeedsRestart(period24h, periodEmpty)).Should(BeFalse(), "24h period vs empty period")
	g.Expect(FeedNeedsRestart(periodEmpty, period24h)).Should(BeFalse(), "empty period vs 24h period")
	g.Expect(FeedNeedsRestart(period12h, periodEmpty)).Should(BeTrue(), "12h period vs empty period")
	g.Expect(FeedNeedsRestart(periodEmpty, period12h)).Should(BeTrue(), "empty period vs 12h period")
	g.Expect(FeedNeedsRestart(testGlobalThreatFeed, noHTTP)).Should(BeTrue(), "missing http on right")
	g.Expect(FeedNeedsRestart(noHTTP, testGlobalThreatFeed)).Should(BeTrue(), "missing http on left")
	g.Expect(FeedNeedsRestart(noHTTP, noHTTP)).Should(BeFalse(), "two missing http")
	g.Expect(FeedNeedsRestart(testGlobalThreatFeed, emptyFormat)).Should(BeFalse(), "empty (default) format on right")
	g.Expect(FeedNeedsRestart(emptyFormat, testGlobalThreatFeed)).Should(BeFalse(), "empty (default) format on left")
	g.Expect(FeedNeedsRestart(testGlobalThreatFeed, jsonFormat)).Should(BeTrue(), "json format on right")
	g.Expect(FeedNeedsRestart(jsonFormat, testGlobalThreatFeed)).Should(BeTrue(), "json format on left")
	g.Expect(FeedNeedsRestart(testGlobalThreatFeed, csvFormat)).Should(BeTrue(), "csv format on right")
	g.Expect(FeedNeedsRestart(csvFormat, testGlobalThreatFeed)).Should(BeTrue(), "csv format on left")
	g.Expect(FeedNeedsRestart(jsonFormat, csvFormat)).Should(BeTrue(), "json vs csv format")
	g.Expect(FeedNeedsRestart(csvFormat, jsonFormat)).Should(BeTrue(), "csv vs json format")
	g.Expect(FeedNeedsRestart(jsonFormat, jsonFormat2)).Should(BeTrue(), "two json formats")
	g.Expect(FeedNeedsRestart(csvFormat, csvFormat2)).Should(BeTrue(), "two csv formats")
	g.Expect(FeedNeedsRestart(testGlobalThreatFeed, noGlobalNetworkSet)).Should(BeTrue(), "no gns on right")
	g.Expect(FeedNeedsRestart(noGlobalNetworkSet, testGlobalThreatFeed)).Should(BeTrue(), "no gns on left")
}
