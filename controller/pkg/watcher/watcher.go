package watcher

import (
	"context"
	"net/http"
	"net/url"
	goSync "sync"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/gc"
	"github.com/tigera/intrusion-detection/controller/pkg/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/searcher"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/sync"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

// Watcher accepts updates from threat pullers and synchronizes them to the
// database
type Watcher interface {
	// Run starts the feed synchronization.
	Run(ctx context.Context)
	Close()
}

type watcher struct {
	ipSet        db.IPSet
	suspiciousIP db.SuspiciousIP
	events       db.Events
	feeds        map[string]feedWatcher
	ctx          context.Context
	cancel       context.CancelFunc
	once         goSync.Once
}

type feedWatcher struct {
	feed             feed.Feed
	puller           puller.Puller
	syncer           sync.Syncer
	garbageCollector gc.GarbageCollector
	searcher         searcher.FlowSearcher
	statser          statser.Statser
}

func NewWatcher(ipSet db.IPSet, suspiciousIP db.SuspiciousIP, events db.Events) Watcher {
	feeds := map[string]feedWatcher{}
	return &watcher{
		ipSet:        ipSet,
		suspiciousIP: suspiciousIP,
		events:       events,
		feeds:        feeds,
	}
}

func (s *watcher) Run(ctx context.Context) {
	s.once.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		s.ctx, s.cancel = context.WithCancel(ctx)

		// hardcode pullers for now
		testUrl, _ := url.Parse("https://test.test/blacklist")
		headers := http.Header{}
		headers.Add("Accept", "text/plain")

		feed := feed.NewFeed("test", "calico-monitoring")
		puller := puller.NewHTTPPuller(feed, testUrl, 24*time.Hour, headers)

		s.startFeed(feed, puller)
	})
}

func (s *watcher) startFeed(feed feed.Feed, puller puller.Puller) {
	fw := feedWatcher{
		feed:             feed,
		puller:           puller,
		syncer:           sync.NewSyncer(feed, s.ipSet),
		garbageCollector: gc.NewGarbageCollector(feed, time.Hour),
		searcher:         searcher.NewFlowSearcher(feed, time.Minute, s.suspiciousIP, s.events),
		statser:          statser.NewStatser(),
	}

	s.feeds[feed.Name()] = fw
	c := puller.Run(s.ctx, fw.statser)
	fw.syncer.Run(s.ctx, c, fw.statser)
	fw.garbageCollector.Run(s.ctx, fw.statser)
	fw.searcher.Run(s.ctx, fw.statser)
}

func (s *watcher) Close() {
	s.cancel()
}
