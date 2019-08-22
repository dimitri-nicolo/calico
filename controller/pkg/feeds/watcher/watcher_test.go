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
	"k8s.io/client-go/tools/cache"

	"github.com/tigera/intrusion-detection/controller/pkg/calico"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/globalnetworksets"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

var testClient = &http.Client{Transport: &puller.MockRoundTripper{Error: errors.New("mock error")}}

func TestWatcher_processQueue(t *testing.T) {
	g := NewGomegaWithT(t)

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	g.Expect(w).ShouldNot(BeNil())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	w.ctx = ctx

	// a non-existing feed is deleted.
	err := w.processQueue(cache.Deltas{
		{
			Type: cache.Deleted,
			Object: &v3.GlobalThreatFeed{
				ObjectMeta: v1.ObjectMeta{
					Name: "nonexisting",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())

	// a feed is added
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Added,
			Object: &v3.GlobalThreatFeed{
				ObjectMeta: v1.ObjectMeta{
					Name: "feed1",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listFeedWatchers()).Should(HaveLen(1))

	// a non-existing feed is updated (should never happen)
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Updated,
			Object: &v3.GlobalThreatFeed{
				ObjectMeta: v1.ObjectMeta{
					Name: "feed2",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listFeedWatchers()).Should(HaveLen(2))

	// an existing feed is added again
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Added,
			Object: &v3.GlobalThreatFeed{
				ObjectMeta: v1.ObjectMeta{
					Name:            "feed1",
					ResourceVersion: "test",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listFeedWatchers()).Should(HaveLen(2))
	fw, ok := w.getFeedWatcher("feed1")
	g.Expect(ok).Should(BeTrue())
	g.Expect(fw.feed.ResourceVersion).Should(Equal("test"))

	// an existing feed is modified
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Added,
			Object: &v3.GlobalThreatFeed{
				ObjectMeta: v1.ObjectMeta{
					Name:            "feed1",
					ResourceVersion: "test2",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listFeedWatchers()).Should(HaveLen(2))
	fw, ok = w.getFeedWatcher("feed1")
	g.Expect(ok).Should(BeTrue())
	g.Expect(fw.feed.ResourceVersion).Should(Equal("test2"))

	// an existing feed is deleted
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Deleted,
			Object: &v3.GlobalThreatFeed{
				ObjectMeta: v1.ObjectMeta{
					Name: "feed1",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listFeedWatchers()).Should(HaveLen(1))
	_, ok = w.getFeedWatcher("feed1")
	g.Expect(ok).Should(BeFalse())
}

func TestWatcher_startFeed_stopFeed(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(gns.NotGCable()).Should(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	w.stopFeedWatcher(ctx, f.Name)
	_, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeFalse(), "FeedWatchers map does not contain feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(0), "No FeedWatchers")
	g.Expect(gns.NotGCable()).ShouldNot(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).ShouldNot(HaveKey(f.Name))
}

func TestWatcher_startFeed_NoPull(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).Should(HaveLen(1), "No FeedWatchers")
	g.Expect(fw.puller).Should(BeNil(), "MockPuller is nil")
	g.Expect(gns.NotGCable()).Should(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))
	g.Expect(fw.statser).ShouldNot(BeNil(), "Statser is not nil")
}

func TestWatcher_startFeed_NoPullHTTP(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).Should(HaveLen(1), "No FeedWatchers")
	g.Expect(fw.puller).Should(BeNil(), "MockPuller is nil")
	g.Expect(gns.NotGCable()).Should(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))
	g.Expect(fw.statser).ShouldNot(BeNil(), "Statser is not nil")
}

func TestWatcher_startFeed_Exists(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	_, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(func() { w.startFeedWatcher(ctx, f) }).Should(Panic())
}

func TestWatcher_stopFeed_notExists(t *testing.T) {
	g := NewGomegaWithT(t)

	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, nil, nil, testClient, nil, nil, nil).(*watcher)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g.Expect(func() { w.stopFeedWatcher(ctx, "mock") }).Should(Panic())
}

func TestWatcher_updateFeed_NotStarted(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	g.Expect(func() { w.updateFeedWatcher(ctx, f, f.DeepCopy()) }).Should(Panic())
}

func TestWatcher_updateFeed_PullToPull(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(gns.NotGCable()).Should(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))

	// hack in some mocks so we can verify that SetFeed was called
	mockPuller := &MockPuller{}
	mockSearcher := &MockSearcher{}
	fw.puller = mockPuller
	fw.searcher = mockSearcher

	w.updateFeedWatcher(ctx, f, f.DeepCopy())

	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(mockPuller.Feed).ShouldNot(BeNil())
	g.Expect(mockPuller.CloseCalled).Should(BeFalse())
	g.Expect(fw.puller).Should(BeIdenticalTo(mockPuller))
	g.Expect(gns.NotGCable()).Should(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))
	g.Expect(mockSearcher.Feed).ShouldNot(BeNil(), "SetFeed was called")
}

func TestWatcher_updateFeed_PullToPush(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(gns.NotGCable()).Should(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))

	// hack in some mocks so we can verify that SetFeed was called
	mockPuller := &MockPuller{}
	mockSearcher := &MockSearcher{}
	fw.puller = mockPuller
	fw.searcher = mockSearcher

	f2 := f.DeepCopy()
	f2.Spec.Pull = nil

	w.updateFeedWatcher(ctx, f, f2)

	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(fw.puller).Should(BeNil())
	g.Expect(gns.NotGCable()).Should(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))
	g.Expect(mockPuller.Feed).Should(BeNil())
	g.Expect(mockPuller.CloseCalled).Should(BeTrue())
	g.Expect(mockSearcher.Feed).ShouldNot(BeNil(), "SetFeed was called")
}
func TestWatcher_updateFeed_PushToPull(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "IPSet",
		},
	}

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(gns.NotGCable()).ShouldNot(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))

	// hack in some mocks so we can verify that SetFeed was called
	searcher := &MockSearcher{}
	fw.searcher = searcher

	f2 := f.DeepCopy()
	f2.Spec.Pull = &v32.Pull{
		Period: "12h",
		HTTP: &v32.HTTPPull{
			Format:  "NewlineDelimited",
			URL:     "http://mock.feed/v1",
			Headers: []v32.HTTPHeader{},
		},
	}
	f2.Spec.GlobalNetworkSet = &v32.GlobalNetworkSetSync{
		Labels: map[string]string{"level": "high"},
	}

	w.updateFeedWatcher(ctx, f, f2)

	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(gns.NotGCable()).Should(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))
	g.Expect(searcher.Feed).ShouldNot(BeNil(), "SetFeed was called")
}

func TestWatcher_updateFeed_PushToPush(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
		ObjectMeta: v1.ObjectMeta{
			Name:      "mock",
			Namespace: util.FeedsNamespace,
		},
		Spec: v32.GlobalThreatFeedSpec{
			Content: "IPSet",
		},
	}

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(gns.NotGCable()).ShouldNot(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))

	searcher := &MockSearcher{}
	fw.searcher = searcher

	w.updateFeedWatcher(ctx, f, f.DeepCopy())

	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")
	g.Expect(fw.puller).Should(BeNil())
	g.Expect(gns.NotGCable()).ShouldNot(HaveKey(util.GlobalNetworkSetNameFromThreatFeed(f.Name)))
	g.Expect(eip.NotGCable()).Should(HaveKey(f.Name))
	g.Expect(searcher.Feed).ShouldNot(BeNil(), "SetFeed was called")
}

func TestWatcher_restartPuller(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	oldPuller := fw.puller

	w.restartPuller(ctx, f)
	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue())
	g.Expect(fw.puller).ShouldNot(Equal(oldPuller))
}

func TestWatcher_restartPuller_NoPull(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	f.Spec.Pull = nil

	w.restartPuller(ctx, f)
	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue())
	g.Expect(fw.puller).Should(BeNil())
}

func TestWatcher_restartPuller_NoPullHTTP(t *testing.T) {
	g := NewGomegaWithT(t)

	f := &v3.GlobalThreatFeed{
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

	ipSet := &db.MockSets{}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	w := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, &db.MockSuspiciousIP{}, &db.MockEvents{}).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startFeedWatcher(ctx, f)

	fw, ok := w.getFeedWatcher(f.Name)
	g.Expect(ok).Should(BeTrue(), "FeedWatchers map contains feed")
	g.Expect(w.listFeedWatchers()).To(HaveLen(1), "Only one FeedWatcher")

	g.Expect(fw.feed).Should(Equal(f))
	g.Expect(fw.puller).ShouldNot(BeNil())
	g.Expect(fw.statser).ShouldNot(BeNil())
	g.Expect(fw.searcher).ShouldNot(BeNil())

	f.Spec.Pull.HTTP = nil

	w.restartPuller(ctx, f)
	fw, ok = w.getFeedWatcher(f.Name)
	g.Expect(fw.puller).Should(BeNil())
}

func TestWatcher_restartPuller_notExists(t *testing.T) {
	g := NewGomegaWithT(t)

	globalThreatFeed := &v3.GlobalThreatFeed{
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
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: globalThreatFeed,
	}

	w := NewWatcher(nil, nil, gtf, nil, nil, testClient, nil, nil, nil).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	g.Expect(func() { w.restartPuller(ctx, globalThreatFeed) }).Should(Panic())
}

func TestWatcher_Ping(t *testing.T) {
	g := NewWithT(t)

	// Include an empty list so that the controller doesn't complain
	gtf := &calico.MockGlobalThreatFeedInterface{GlobalThreatFeedList: &v3.GlobalThreatFeedList{}}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	uut := NewWatcher(nil, nil, gtf, gns, eip, testClient, nil, nil, nil)

	ch := make(chan struct{})
	defer func() {
		g.Eventually(ch).Should(BeClosed(), "Test cleans up correctly")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	go func() {
		defer close(ch)
		err := uut.Ping(ctx)
		g.Expect(err).ToNot(HaveOccurred())
	}()
	g.Consistently(ch).ShouldNot(BeClosed(), "Ping does not complete before Run is called")

	uut.Run(ctx)

	g.Eventually(ch).Should(BeClosed(), "Ping completes after Run is called")
}

func TestWatcher_PingFail(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond)
	defer cancel()

	// Include an empty list so that the controller doesn't complain
	gtf := &calico.MockGlobalThreatFeedInterface{GlobalThreatFeedList: &v3.GlobalThreatFeedList{}}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	uut := NewWatcher(nil, nil, gtf, gns, eip, testClient, nil, nil, nil)

	err := uut.Ping(ctx)
	g.Expect(err).Should(MatchError(context.DeadlineExceeded), "Ping times out")
}

func TestWatcher_Ready(t *testing.T) {
	g := NewWithT(t)

	gtf := &calico.MockGlobalThreatFeedInterface{W: &calico.MockWatch{make(chan watch.Event)}, GlobalThreatFeedList: &v3.GlobalThreatFeedList{}, GlobalThreatFeed: &v3.GlobalThreatFeed{}}
	ipSet := &db.MockSets{}
	sIP := &db.MockSuspiciousIP{ErrorIndex: -1}
	gns := globalnetworksets.NewMockGlobalNetworkSetController()
	eip := elastic.NewMockElasticIPSetController()
	uut := NewWatcher(nil, nil, gtf, gns, eip, testClient, ipSet, sIP, &db.MockEvents{})

	g.Expect(uut.Ready()).To(BeFalse())

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	uut.Run(ctx)
	g.Eventually(uut.Ready).Should(BeTrue())

	// Send in gtf with no error
	g.Eventually(gtf.W.C).Should(BeSent(watch.Event{Type: watch.Added, Object: util.NewGlobalThreatFeedFromName("mock0")}))
	g.Consistently(uut.Ready).Should(BeTrue())

	// New gtf has error
	sIP.Error = errors.New("test")
	g.Eventually(gtf.W.C).Should(BeSent(watch.Event{Type: watch.Added, Object: util.NewGlobalThreatFeedFromName("mock1")}))
	g.Eventually(uut.Ready).Should(BeFalse())

	// Remove both GTFs
	g.Eventually(gtf.W.C).Should(BeSent(watch.Event{Type: watch.Deleted, Object: util.NewGlobalThreatFeedFromName("mock0")}))
	g.Eventually(gtf.W.C).Should(BeSent(watch.Event{Type: watch.Deleted, Object: util.NewGlobalThreatFeedFromName("mock1")}))
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

func (*MockPuller) Run(context.Context, statser.Statser) {
	panic("implement me")
}

func (p *MockPuller) SetFeed(f *v3.GlobalThreatFeed) {
	p.Feed = f
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
