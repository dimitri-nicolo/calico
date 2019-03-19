// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"errors"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	v32 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/mock"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

/*
func TestWatcher_sync(t *testing.T) {
	g := NewGomegaWithT(t)

	gti := &GlobalThreatFeedInterface{}
	db := &mockDB{}
	gns := &GlobalNetworkSetInterface{}
	w := NewWatcher(nil, nil, gti, gns, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, db, db, db).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	err := w.sync(ctx)
	g.Expect(err).Should(HaveOccurred(), "Fails with nil GlobalThreatFeeds pointer")
	g.Expect(w.feeds).Should(HaveLen(0), "No feeds running")

	gti.list = &v3.GlobalThreatFeedList{}
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred(), "Succeeds with empty GlobalThreatFeeds")
	g.Expect(w.feeds).Should(HaveLen(0), "No feeds running")

	gti.err = errors.New("mock error")
	err = w.sync(ctx)
	g.Expect(err).Should(HaveOccurred(), "Fails with GlobalThreatFeeds.List error")
	g.Expect(w.feeds).Should(HaveLen(0), "No feeds running")
	gti.err = nil

	// Add a new feed
	gti.list = &v3.GlobalThreatFeedList{
		Items: []v3.GlobalThreatFeed{
			{
				ObjectMeta: v1.ObjectMeta{
					Name:      "mock",
					Namespace: feed.FeedsNamespace,
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
							Format:  "NewlineDelimited",
							URL:     "http://test.feed/v1",
							Headers: []v32.HTTPHeader{},
						},
					},
				},
			},
		},
	}
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred(), "Succeeds with empty GlobalThreatFeeds")
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[gti.list.Items[0].Name].feed.GlobalThreatFeed).Should(Equal(gti.list.Items[0]), "Feed is as expected")

	// Change the URL
	gti.list.Items[0].Spec.Pull.HTTP.URL = "http://test.feed/v2"
	puller := w.feeds[gti.list.Items[0].Name].puller
	g.Expect(w.feeds[gti.list.Items[0].Name].feed.GlobalThreatFeed.Spec.Pull.HTTP.URL).ShouldNot(Equal(gti.list.Items[0].Spec.Pull.HTTP.URL), "Feed url is about to change")
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred(), "Succeeds with empty GlobalThreatFeeds")
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[gti.list.Items[0].Name].feed.GlobalThreatFeed).Should(Equal(gti.list.Items[0]), "Feed updated")
	g.Expect(w.feeds[gti.list.Items[0].Name].puller).Should(Equal(puller), "Puller was not restarted")

	// Change the period
	gti.list.Items[0].Spec.Pull.Period = "3h"
	puller = w.feeds[gti.list.Items[0].Name].puller
	syncer := w.feeds[gti.list.Items[0].Name].syncer
	g.Expect(w.feeds[gti.list.Items[0].Name].feed.GlobalThreatFeed.Spec.Pull.Period).ShouldNot(Equal(gti.list.Items[0].Spec.Pull.Period), "Period is about to change")
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred(), "Succeeds with empty GlobalThreatFeeds")
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[gti.list.Items[0].Name].feed.GlobalThreatFeed).Should(Equal(gti.list.Items[0]), "Feed updated")
	g.Expect(w.feeds[gti.list.Items[0].Name].puller).ShouldNot(Equal(puller), "Puller was restarted")
	g.Expect(w.feeds[gti.list.Items[0].Name].syncer).ShouldNot(Equal(syncer), "Syncer was restarted")

	// Add another feed
	gti.list.Items = append(gti.list.Items, v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: feed.FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "IPSet",
			GlobalNetworkSet: &v32.GlobalNetworkSetSync{
				Labels: map[string]string{
					"level": "high",
				},
			},
			Pull: &v32.Pull{
				Period: "24h",
				HTTP: &v32.HTTPPull{
					Format:  "NewlineDelimited",
					URL:     "http://test2.feed/v1",
					Headers: []v32.HTTPHeader{},
				},
			},
		},
	})
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(w.feeds).Should(HaveLen(2), "2 feeds running")

	// Remove a feed
	canceledName := gti.list.Items[0].Name
	gti.list.Items = gti.list.Items[1:]
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[canceledName]).Should(BeNil(), "Feed 1 is not running")
	g.Expect(w.feeds[gti.list.Items[0].Name]).ShouldNot(BeNil(), "Feed 2 is still running")

	// Add a feed with missing Pull section
	gti.list.Items = append(gti.list.Items, v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test3",
			Namespace: feed.FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "IPSet",
			GlobalNetworkSet: &v32.GlobalNetworkSetSync{
				Labels: map[string]string{
					"level": "high",
				},
			},
			Pull: nil,
		},
	})
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[gti.list.Items[0].Name]).ShouldNot(BeNil(), "Feed 2 is still running")

	// Add a feed with a broken pull section
	gti.list.Items = append(gti.list.Items, v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test4",
			Namespace: feed.FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "IPSet",
			GlobalNetworkSet: &v32.GlobalNetworkSetSync{
				Labels: map[string]string{
					"level": "high",
				},
			},
			Pull: &v32.Pull{
				Period: "abc",
				HTTP: &v32.HTTPPull{
					Format:  "NewlineDelmited",
					URL:     "http://test3.feed/v1",
					Headers: nil,
				},
			},
		},
	})
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[gti.list.Items[0].Name]).ShouldNot(BeNil(), "Feed 2 is still running")

	// Add a feed with an unsupported Content
	gti.list.Items = append(gti.list.Items, v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test4",
			Namespace: feed.FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "DomainSet",
			GlobalNetworkSet: &v32.GlobalNetworkSetSync{
				Labels: map[string]string{
					"level": "high",
				},
			},
			Pull: &v32.Pull{
				Period: "12h",
				HTTP: &v32.HTTPPull{
					Format:  "NewlineDelmited",
					URL:     "http://test3.feed/v1",
					Headers: nil,
				},
			},
		},
	})
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[gti.list.Items[0].Name]).ShouldNot(BeNil(), "Feed 2 is still running")

	// A valid feed is refreshed with an invalid one
	gti.list.Items[0].Spec.Pull.Period = "abc"
	pullBefore := *w.feeds[gti.list.Items[0].Name].feed.Spec.Pull
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[gti.list.Items[0].Name]).ShouldNot(BeNil(), "Feed 2 is still running")
	g.Expect(*w.feeds[gti.list.Items[0].Name].feed.Spec.Pull).Should(Equal(pullBefore), "period did not change")

	// A valid feed is refreshed with one with a missing pull section
	gti.list.Items[0].Spec.Pull = nil
	pullBefore = *w.feeds[gti.list.Items[0].Name].feed.Spec.Pull
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(w.feeds).Should(HaveLen(1), "1 feed running")
	g.Expect(w.feeds[gti.list.Items[0].Name]).ShouldNot(BeNil(), "Feed 2 is still running")
	g.Expect(*w.feeds[gti.list.Items[0].Name].feed.Spec.Pull).Should(Equal(pullBefore), "period did not change")

	// Remove all feeds
	gti.list = &v3.GlobalThreatFeedList{}
	err = w.sync(ctx)
	g.Expect(err).ShouldNot(HaveOccurred(), "Succeeds with empty GlobalThreatFeeds")
	g.Expect(w.feeds).Should(HaveLen(0), "All feeds stopped")
}
*/

func TestWatcher_startFeed_stopFeed(t *testing.T) {
	g := NewGomegaWithT(t)

	f := v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
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
					Format:  "NewlineDelimited",
					URL:     "http://mock.feed/v1",
					Headers: []v32.HTTPHeader{},
				},
			},
		},
	}

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}

	w := NewWatcher(nil, nil, nil, gns, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.feeds[f.Name]
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.feeds).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(*fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.syncer).ShouldNot(BeNil())
	g.Expect(fw.garbageCollector).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	w.stopFeed(f.Name)
	_, ok = w.feeds[f.Name]
	g.Expect(ok).Should(BeFalse(), "FeedWatchers map does not contain feed")
	g.Expect(w.feeds).To(HaveLen(0), "No FeedWatchers")
}

func TestWatcher_startFeed_NoPull(t *testing.T) {
	g := NewGomegaWithT(t)

	f := v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "IPSet",
			GlobalNetworkSet: &v32.GlobalNetworkSetSync{
				Labels: map[string]string{
					"level": "high",
				},
			},
		},
	}

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}

	w := NewWatcher(nil, nil, nil, gns, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.feeds[f.Name]
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.feeds).Should(HaveLen(1), "No FeedWatchers")
	g.Expect(fw.puller).Should(BeNil(), "Puller is nil")
	g.Expect(fw.syncer).Should(BeNil(), "Syncer is nil")
	g.Expect(fw.statser).ShouldNot(BeNil(), "Statser is not nil")
	g.Expect(fw.garbageCollector).ShouldNot(BeNil(), "GC is not nil")
}

func TestWatcher_startFeed_NoPullHTTP(t *testing.T) {
	g := NewGomegaWithT(t)

	f := v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "IPSet",
			GlobalNetworkSet: &v32.GlobalNetworkSetSync{
				Labels: map[string]string{
					"level": "high",
				},
			},
			Pull: &v32.Pull{},
		},
	}

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}

	w := NewWatcher(nil, nil, nil, gns, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.feeds[f.Name]
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.feeds).Should(HaveLen(1), "No FeedWatchers")
	g.Expect(fw.puller).Should(BeNil(), "Puller is nil")
	g.Expect(fw.syncer).Should(BeNil(), "Syncer is nil")
	g.Expect(fw.statser).ShouldNot(BeNil(), "Statser is not nil")
	g.Expect(fw.garbageCollector).ShouldNot(BeNil(), "GC is not nil")
}

func TestWatcher_startFeed_Exists(t *testing.T) {
	g := NewGomegaWithT(t)

	f := v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
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
					Format:  "NewlineDelimited",
					URL:     "http://mock.feed/v1",
					Headers: []v32.HTTPHeader{},
				},
			},
		},
	}

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}

	w := NewWatcher(nil, nil, nil, gns, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	_, ok := w.feeds[f.Name]
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.feeds).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(func() { w.startFeed(ctx, f) }).Should(Panic())
}

func TestWatcher_stopFeed_notExists(t *testing.T) {
	g := NewGomegaWithT(t)

	w := NewWatcher(nil, nil, nil, nil, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, nil, nil, nil).(*watcher)

	g.Expect(func() { w.stopFeed("mock") }).Should(Panic())
}

func TestWatcher_restartPuller(t *testing.T) {
	g := NewGomegaWithT(t)

	f := v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
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
					Format:  "NewlineDelimited",
					URL:     "http://mock.feed/v1",
					Headers: []v32.HTTPHeader{},
				},
			},
		},
	}

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}

	w := NewWatcher(nil, nil, nil, gns, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.feeds[f.Name]
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.feeds).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(*fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.syncer).ShouldNot(BeNil())
	g.Expect(fw.garbageCollector).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	oldPuller := fw.puller
	oldSyncer := fw.syncer

	w.restartPuller(ctx, f)
	g.Expect(w.feeds[f.Name].puller).ShouldNot(Equal(oldPuller))
	g.Expect(w.feeds[f.Name].syncer).ShouldNot(Equal(oldSyncer))
}

func TestWatcher_restartPuller_Fail(t *testing.T) {
	g := NewGomegaWithT(t)

	f := v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
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
					Format:  "NewlineDelimited",
					URL:     "http://mock.feed/v1",
					Headers: []v32.HTTPHeader{},
				},
			},
		},
	}

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}

	w := NewWatcher(nil, nil, nil, gns, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	g.Expect(func() { w.restartPuller(ctx, f) }).Should(Panic())

	/*
		TODO move to another test
		err := w.startFeed(ctx, f)
		g.Expect(err).ShouldNot(HaveOccurred())

		fw, ok := w.feeds[f.Name]
		g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
		g.Expect(w.feeds).To(HaveLen(1), "Only one FeedWatcher")

		oldPuller := fw.puller
		oldSyncer := fw.syncer

		f.Spec.Pull.Period = "abc"
		err = w.restartPuller(ctx, f)
		g.Expect(err).ShouldNot(HaveOccurred(), "restartPuller was not successful")
		g.Expect(w.feeds[f.Name].puller).Should(Equal(oldPuller), "puller was not restarted")
		g.Expect(w.feeds[f.Name].syncer).Should(Equal(oldSyncer), "syncer was not restarted")
	*/
}

func TestWatcher_restartPuller_notExists(t *testing.T) {
	g := NewGomegaWithT(t)

	globalThreatFeed := v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
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
					Format:  "NewlineDelimited",
					URL:     "http://mock.feed/v1",
					Headers: []v32.HTTPHeader{},
				},
			},
		},
	}

	w := NewWatcher(nil, nil, nil, nil, &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}, nil, nil, nil).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	g.Expect(func() { w.restartPuller(ctx, globalThreatFeed) }).Should(Panic())
}
