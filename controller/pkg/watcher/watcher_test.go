// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v32 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/mock"
	"github.com/tigera/intrusion-detection/controller/pkg/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

var testClient = &http.Client{Transport: &mock.RoundTripper{Error: errors.New("mock error")}}

func TestWatcher_HandleEvent(t *testing.T) {
	g := NewGomegaWithT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ipSet := &mock.IPSet{}
	gns := &mock.GlobalNetworkSetInterface{}

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	g.Expect(w).ShouldNot(BeNil())

	// a k8s status is reported
	w.handleEvent(ctx, watch.Event{
		Type:   watch.Error,
		Object: &v1.Status{Reason: v1.StatusReasonInternalError},
	})

	// an error is reported with something unexpected
	w.handleEvent(ctx, watch.Event{
		Type:   watch.Error,
		Object: nil,
	})

	// a non-existing feed is deleted.
	w.handleEvent(ctx, watch.Event{
		Type: watch.Deleted,
		Object: &v3.GlobalThreatFeed{
			ObjectMeta: v1.ObjectMeta{
				Name: "nonexisting",
			},
		},
	})

	// a feed is added
	w.handleEvent(ctx, watch.Event{
		Type: watch.Added,
		Object: &v3.GlobalThreatFeed{
			ObjectMeta: v1.ObjectMeta{
				Name: "feed1",
			},
		},
	})
	g.Expect(w.listFeedWatchers()).Should(HaveLen(1))

	// a non-existing feed is with Modified (should never happen)
	w.handleEvent(ctx, watch.Event{
		Type: watch.Modified,
		Object: &v3.GlobalThreatFeed{
			ObjectMeta: v1.ObjectMeta{
				Name: "feed2",
			},
		},
	})
	g.Expect(w.listFeedWatchers()).Should(HaveLen(2))

	// an existing feed is added again
	w.handleEvent(ctx, watch.Event{
		Type: watch.Added,
		Object: &v3.GlobalThreatFeed{
			ObjectMeta: v1.ObjectMeta{
				Name:            "feed1",
				ResourceVersion: "test",
			},
		},
	})
	g.Expect(w.listFeedWatchers()).Should(HaveLen(2))
	fw, ok := w.getFeedWatcher("feed1")
	g.Expect(ok).Should(BeTrue())
	g.Expect(fw.feed.ResourceVersion).Should(Equal("test"))

	// an existing feed is modified
	w.handleEvent(ctx, watch.Event{
		Type: watch.Added,
		Object: &v3.GlobalThreatFeed{
			ObjectMeta: v1.ObjectMeta{
				Name:            "feed1",
				ResourceVersion: "test2",
			},
		},
	})
	g.Expect(w.listFeedWatchers()).Should(HaveLen(2))
	fw, ok = w.getFeedWatcher("feed1")
	g.Expect(ok).Should(BeTrue())
	g.Expect(fw.feed.ResourceVersion).Should(Equal("test2"))

	// an existing feed is deleted
	w.handleEvent(ctx, watch.Event{
		Type: watch.Deleted,
		Object: &v3.GlobalThreatFeed{
			ObjectMeta: v1.ObjectMeta{
				Name: "feed1",
			},
		},
	})
	g.Expect(w.listFeedWatchers()).Should(HaveLen(1))
	_, ok = w.getFeedWatcher("feed1")
	g.Expect(ok).Should(BeFalse())

	// a nil feed is added
	w.handleEvent(ctx, watch.Event{
		Type:   watch.Added,
		Object: nil,
	})

	// a nil feed is modified
	w.handleEvent(ctx, watch.Event{
		Type:   watch.Modified,
		Object: nil,
	})

	// a nil feed is deleted
	w.handleEvent(ctx, watch.Event{
		Type:   watch.Deleted,
		Object: nil,
	})

	// an unexpected event type is received
	w.handleEvent(ctx, watch.Event{
		Type:   "something",
		Object: nil,
	})
}

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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(*fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.syncer).ShouldNot(BeNil())
	g.Expect(fw.garbageCollector).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	w.stopFeed(f.Name)
	_, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeFalse(), "FeedWatchers map does not contain feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(0), "No FeedWatchers")
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).Should(HaveLen(1), "No FeedWatchers")
	g.Expect(fw.puller).Should(BeNil(), "MockPuller is nil")
	g.Expect(fw.syncer).Should(BeNil(), "MockSyncer is nil")
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).Should(HaveLen(1), "No FeedWatchers")
	g.Expect(fw.puller).Should(BeNil(), "MockPuller is nil")
	g.Expect(fw.syncer).Should(BeNil(), "MockSyncer is nil")
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	_, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(func() { w.startFeed(ctx, f) }).Should(Panic())
}

func TestWatcher_stopFeed_notExists(t *testing.T) {
	g := NewGomegaWithT(t)

	w := NewWatcher(nil, nil, nil, nil, testClient, nil, nil, nil).(*watcher)

	g.Expect(func() { w.stopFeed("mock") }).Should(Panic())
}

func TestWatcher_updateFeed_NotStarted(t *testing.T) {
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	g.Expect(func() { w.updateFeed(ctx, f) }).Should(Panic())
}

func TestWatcher_updateFeed_PullToPull(t *testing.T) {
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	// hack in some mocks so we can verify that SetFeed was called
	mockPuller := &MockPuller{}
	mockSyncer := &MockSyncer{}
	mockSearcher := &MockSearcher{}
	mockGC := &MockGC{}
	fw.puller = mockPuller
	fw.syncer = mockSyncer
	fw.searcher = mockSearcher
	fw.garbageCollector = mockGC

	w.updateFeed(ctx, f)

	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(mockPuller.Feed).ShouldNot(BeNil())
	g.Expect(mockPuller.CloseCalled).Should(BeFalse())
	g.Expect(fw.puller).Should(BeIdenticalTo(mockPuller))
	g.Expect(mockSyncer.Feed).ShouldNot(BeNil())
	g.Expect(mockSyncer.CloseCalled).Should(BeFalse())
	g.Expect(fw.syncer).Should(BeIdenticalTo(mockSyncer))
	g.Expect(mockSearcher.Feed).ShouldNot(BeNil(), "SetFeed was called")
	g.Expect(mockGC.Feed).ShouldNot(BeNil(), "SetFeed was called")
}

func TestWatcher_updateFeed_PullToPush(t *testing.T) {
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	// hack in some mocks so we can verify that SetFeed was called
	mockPuller := &MockPuller{}
	mockSyncer := &MockSyncer{}
	mockSearcher := &MockSearcher{}
	mockGC := &MockGC{}
	fw.puller = mockPuller
	fw.syncer = mockSyncer
	fw.searcher = mockSearcher
	fw.garbageCollector = mockGC

	f.Spec.Pull = nil

	w.updateFeed(ctx, f)

	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(fw.puller).Should(BeNil())
	g.Expect(fw.syncer).Should(BeNil())
	g.Expect(mockPuller.Feed).Should(BeNil())
	g.Expect(mockPuller.CloseCalled).Should(BeTrue())
	g.Expect(mockSyncer.Feed).Should(BeNil())
	g.Expect(mockSyncer.CloseCalled).Should(BeTrue())
	g.Expect(mockSearcher.Feed).ShouldNot(BeNil(), "SetFeed was called")
	g.Expect(mockGC.Feed).ShouldNot(BeNil(), "SetFeed was called")
}
func TestWatcher_updateFeed_PushToPull(t *testing.T) {
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	// hack in some mocks so we can verify that SetFeed was called
	searcher := &MockSearcher{}
	garbageCollector := &MockGC{}
	fw.searcher = searcher
	fw.garbageCollector = garbageCollector

	f.Spec.Pull = &v32.Pull{
		Period: "12h",
		HTTP: &v32.HTTPPull{
			Format:  "NewlineDelimited",
			URL:     "http://mock.feed/v1",
			Headers: []v32.HTTPHeader{},
		},
	}

	w.updateFeed(ctx, f)

	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.syncer).ShouldNot(BeNil())
	g.Expect(searcher.Feed).ShouldNot(BeNil(), "SetFeed was called")
	g.Expect(garbageCollector.Feed).ShouldNot(BeNil(), "SetFeed was called")
}

func TestWatcher_updateFeed_PushToPush(t *testing.T) {
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	searcher := &MockSearcher{}
	garbageCollector := &MockGC{}
	fw.searcher = searcher
	fw.garbageCollector = garbageCollector

	w.updateFeed(ctx, f)

	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(fw.puller).Should(BeNil())
	g.Expect(fw.syncer).Should(BeNil())
	g.Expect(searcher.Feed).ShouldNot(BeNil(), "SetFeed was called")
	g.Expect(garbageCollector.Feed).ShouldNot(BeNil(), "SetFeed was called")
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(*fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.syncer).ShouldNot(BeNil())
	g.Expect(fw.garbageCollector).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	oldPuller := fw.puller
	oldSyncer := fw.syncer

	w.restartPuller(ctx, f)
	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue())
	g.Expect(fw.puller).ShouldNot(Equal(oldPuller))
	g.Expect(fw.syncer).ShouldNot(Equal(oldSyncer))
}

func TestWatcher_restartPuller_NoPull(t *testing.T) {
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(*fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.syncer).ShouldNot(BeNil())
	g.Expect(fw.garbageCollector).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	f.Spec.Pull = nil

	w.restartPuller(ctx, f)
	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue())
	g.Expect(fw.puller).Should(BeNil())
	g.Expect(fw.syncer).Should(BeNil())
}

func TestWatcher_restartPuller_NoPullHTTP(t *testing.T) {
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

	w := NewWatcher(nil, nil, nil, gns, testClient, ipSet, &mock.SuspiciousIP{}, &mock.Events{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeed(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(*fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.syncer).ShouldNot(BeNil())
	g.Expect(fw.garbageCollector).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	f.Spec.Pull.HTTP = nil

	w.restartPuller(ctx, f)
	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(fw.puller).Should(BeNil())
	g.Expect(fw.syncer).Should(BeNil())
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

	w := NewWatcher(nil, nil, nil, nil, testClient, nil, nil, nil).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	g.Expect(func() { w.restartPuller(ctx, globalThreatFeed) }).Should(Panic())
}

func TestWatcher_Ping(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	gtf := &mock.GlobalThreatFeedInterface{}
	uut := NewWatcher(nil, nil, gtf, nil, testClient, nil, nil, nil)

	ch := make(chan struct{})
	defer func() {
		select {
		case <-ctx.Done():
			t.Error("Ping did not terminate before context cancellation")
		case <-ch:
			// ok
		}
	}()

	var done bool
	go func() {
		defer close(ch)
		err := uut.Ping(ctx)
		done = true
		g.Expect(err).ToNot(HaveOccurred())
	}()
	g.Consistently(func() bool { return done }).Should(BeFalse())

	uut.Run(ctx)

	g.Eventually(func() bool { return done }).Should(BeTrue())
}

func TestWatcher_PingFail(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond)
	defer cancel()

	uut := NewWatcher(nil, nil, nil, nil, testClient, nil, nil, nil)

	err := uut.Ping(ctx)
	g.Expect(err).Should(MatchError(context.DeadlineExceeded), "Ping times out")
}

func TestWatcher_Ready(t *testing.T) {
	g := NewWithT(t)

	gtf := &mock.GlobalThreatFeedInterface{W: &mock.Watch{make(chan watch.Event)}}
	ipSet := &mock.IPSet{}
	sIP := &mock.SuspiciousIP{ErrorIndex: -1}
	uut := NewWatcher(nil, nil, gtf, nil, testClient, ipSet, sIP, &mock.Events{})

	g.Expect(uut.Ready()).To(BeFalse())

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	uut.Run(ctx)
	g.Eventually(uut.Ready).Should(BeTrue())

	// Send in gtf with no error
	select {
	case <-ctx.Done():
		t.Fatal("Timed out")
	case gtf.W.C <- watch.Event{Type: watch.Added, Object: util.NewGlobalThreatFeedFromName("mock0")}:
		// ok
	}
	g.Consistently(uut.Ready).Should(BeTrue())

	// New gtf has error
	sIP.Error = errors.New("test")
	select {
	case <-ctx.Done():
		t.Fatal("Timed out")
	case gtf.W.C <- watch.Event{Type: watch.Added, Object: util.NewGlobalThreatFeedFromName("mock1")}:
		// ok
	}
	g.Eventually(uut.Ready).Should(BeFalse())

	// Remove both GTFs
	select {
	case <-ctx.Done():
		t.Fatal("Timed out")
	case gtf.W.C <- watch.Event{Type: watch.Deleted, Object: util.NewGlobalThreatFeedFromName("mock0")}:
		// ok
	}
	select {
	case <-ctx.Done():
		t.Fatal("Timed out")
	case gtf.W.C <- watch.Event{Type: watch.Deleted, Object: util.NewGlobalThreatFeedFromName("mock1")}:
		// ok
	}
	g.Eventually(uut.Ready).Should(BeTrue())

	// Stop the watch loop
	cancel()
	g.Eventually(uut.Ready).Should(BeFalse())
}

type MockPuller struct {
	Feed        *v3.GlobalThreatFeed
	CloseCalled bool
}

func (p *MockPuller) Close() {
	p.CloseCalled = true
}

func (*MockPuller) Run(context.Context, statser.Statser) (<-chan db.IPSetSpec, puller.SyncFailFunction) {
	panic("implement me")
}

func (p *MockPuller) SetFeed(f *v3.GlobalThreatFeed) {
	p.Feed = f
}

type MockSyncer struct {
	Feed        *v3.GlobalThreatFeed
	CloseCalled bool
}

func (s *MockSyncer) Close() {
	s.CloseCalled = true
}

func (*MockSyncer) Run(context.Context, <-chan db.IPSetSpec, puller.SyncFailFunction, statser.Statser) {
	panic("implement me")
}

func (s *MockSyncer) SetFeed(f *v3.GlobalThreatFeed) {
	s.Feed = f
}

type MockSearcher struct {
	Feed *v3.GlobalThreatFeed
}

func (*MockSearcher) Close() {
	panic("implement me")
}

func (*MockSearcher) Run(context.Context, statser.Statser) {
	panic("implement me")
}

func (m *MockSearcher) SetFeed(f *v3.GlobalThreatFeed) {
	m.Feed = f
}

type MockGC struct {
	Feed *v3.GlobalThreatFeed
}

func (*MockGC) Run(context.Context, statser.Statser) {
	panic("implement me")
}

func (m *MockGC) SetFeed(f *v3.GlobalThreatFeed) {
	m.Feed = f
}

func (*MockGC) Close() {
	panic("implement me")
}
