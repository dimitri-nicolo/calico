package feed

import (
	"context"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/phulax/pkg/db"
)

type Feed interface {
	Name() string
	Namespace() string
}

type IPSetPuller interface {
	Feed
	// Run activates the feed and returns a channel that sends snapshots of the
	// IPs that are considered suspicious.
	Run(ctx context.Context) <-chan []string
}

// Syncher accepts updates from threat feeds and synchronizes them to the
// database
type Syncher interface {
	// Sync starts the feed synchronization.
	Sync(ctx context.Context)
}

type syncher struct {
	db    db.IPSet
	feeds map[string]IPSetPuller
}

func NewSyncher(db db.IPSet) Syncher {
	// hardcode feeds for now
	abuseipdbAPIKey := "5427761c0994c123fef64b5e9c973d1d24eb2035b59ac5f00cfdf0c41f88b4139ed331d7ee1576f7"
	abuseipdbUrl, _ := url.Parse("https://api.abuseipdb.com/api/v2/blacklist")
	headers := http.Header{}
	headers.Add("Key", abuseipdbAPIKey)
	headers.Add("Accept", "text/plain")
	feeds := map[string]IPSetPuller{
		"abuseipdb": NewHTTPPuller("abuseipdb", "calico-monitoring", abuseipdbUrl, 24*time.Hour, headers)}
	return &syncher{db: db, feeds: feeds}
}

func (s *syncher) Sync(ctx context.Context) {
	for name, feed := range s.feeds {
		c := feed.Run(ctx)
		go s.syncIPFeed(ctx, name, c)
	}
}

func (s *syncher) syncIPFeed(ctx context.Context, name string, c <-chan []string) {
	for {
		select {
		case <-ctx.Done():
			return
		case set, ok := <-c:
			if ok {
				err := s.db.PutIPSet(name, set)
				if err != nil {
					log.WithError(err).Error("could not put IPSetPuller set from feed")
				}
			} else {
				// channel closed
				return
			}
		}
	}
}
