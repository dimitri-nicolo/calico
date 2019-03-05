package puller

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
)

type Puller interface {
	// Run activates the feed and returns a channel that sends snapshots of the
	// IPs that are considered suspicious.
	Run(ctx context.Context) <-chan feed.IPSet
	Close()
}
