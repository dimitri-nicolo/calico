package watcher

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/puller"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

// FeedWatcher accepts updates from threat feeds and synchronizes them to the
// database
type FeedWatcher interface {
	// Run starts the feed synchronization.
	Run(ctx context.Context)
}

type feedWatcher struct {
	db    db.IPSet
	feeds map[string]puller.FeedPuller
}

func NewFeedWatcher(db db.IPSet) FeedWatcher {
	// hardcode feeds for now
	abuseipdbAPIKey := "5427761c0994c123fef64b5e9c973d1d24eb2035b59ac5f00cfdf0c41f88b4139ed331d7ee1576f7"
	abuseipdbUrl, _ := url.Parse("https://api.abuseipdb.com/api/v2/blacklist")
	headers := http.Header{}
	headers.Add("Key", abuseipdbAPIKey)
	headers.Add("Accept", "text/plain")
	feeds := map[string]puller.FeedPuller{
		"abuseipdb": puller.NewHTTPPuller("abuseipdb", "calico-monitoring", abuseipdbUrl, 24*time.Hour, headers)}
	return &feedWatcher{db: db, feeds: feeds}
}

func (s *feedWatcher) Run(ctx context.Context) {
	for name, feed := range s.feeds {
		c := feed.Run(ctx)
		go s.syncIPFeed(ctx, name, c)
	}
}

func (s *feedWatcher) syncIPFeed(ctx context.Context, name string, c <-chan []string) {
	for {
		select {
		case <-ctx.Done():
			return
		case set, ok := <-c:
			if ok {
				err := s.db.PutIPSet(ctx, name, set)
				if err != nil {
					log.WithError(err).Error("could not put FeedPuller set from feed")
				}
			} else {
				// channel closed
				return
			}
		}
	}
}
