package watcher

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/gc"
	"github.com/tigera/intrusion-detection/controller/pkg/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/searcher"
	"github.com/tigera/intrusion-detection/controller/pkg/sync"
	"net/http"
	"net/url"
	goSync "sync"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

// FeedWatcher accepts updates from threat pullers and synchronizes them to the
// database
type FeedWatcher interface {
	// Run starts the feed synchronization.
	Run(ctx context.Context)
	Close()
}

type feedWatcher struct {
	ipSet             db.IPSet
	suspiciousIP      db.SuspiciousIP
	events            db.Events
	feeds             map[string]feed.Feed
	pullers           map[string]puller.Puller
	syncers           map[string]sync.Syncer
	garbageCollectors map[string]gc.GarbageCollector
	searchers         map[string]searcher.FlowSearcher
	ctx               context.Context
	cancel            context.CancelFunc
	once              goSync.Once
}

func NewFeedWatcher(ipSet db.IPSet, suspiciousIP db.SuspiciousIP, events db.Events) FeedWatcher {
	// hardcode pullers for now
	abuseipdbAPIKey := "5427761c0994c123fef64b5e9c973d1d24eb2035b59ac5f00cfdf0c41f88b4139ed331d7ee1576f7"
	abuseipdbUrl, _ := url.Parse("https://api.abuseipdb.com/api/v2/blacklist")
	headers := http.Header{}
	headers.Add("Key", abuseipdbAPIKey)
	headers.Add("Accept", "text/plain")
	feeds := map[string]feed.Feed{
		"abuseipdb": feed.NewFeed("abuseipdb", "calico-monitoring"),
	}
	pullers := map[string]puller.Puller{
		"abuseipdb": puller.NewHTTPPuller(feeds["abuseipdb"], abuseipdbUrl, 24*time.Hour, headers)}
	syncers := map[string]sync.Syncer{}
	garbageCollectors := map[string]gc.GarbageCollector{}
	searchers := map[string]searcher.FlowSearcher{}
	return &feedWatcher{
		ipSet:             ipSet,
		suspiciousIP:      suspiciousIP,
		events:            events,
		feeds:             feeds,
		pullers:           pullers,
		syncers:           syncers,
		garbageCollectors: garbageCollectors,
		searchers:         searchers,
	}
}

func (s *feedWatcher) Run(ctx context.Context) {
	s.once.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		s.ctx, s.cancel = context.WithCancel(ctx)

		for name, feed := range s.feeds {
			c := s.pullers[name].Run(s.ctx)
			s.syncers[name] = sync.NewSyncer(feed, c, s.ipSet)
			s.syncers[name].Run(ctx)
			s.garbageCollectors[name] = gc.NewGarbageCollector(feed, time.Hour)
			s.garbageCollectors[name].Run(ctx)
			s.searchers[name] = searcher.NewFlowSearcher(s.suspiciousIP, s.events)
			s.searchers[name].Run(s.ctx, name, 1*time.Minute)
		}
	})
}

func (s *feedWatcher) Close() {
	s.cancel()
}
