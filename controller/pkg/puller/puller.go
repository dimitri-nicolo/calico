package puller

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

const statserType = "PullFailed"

type Puller interface {
	// Run activates the feed and returns a channel that sends snapshots of the
	// IPs that are considered suspicious.
	Run(context.Context, statser.Statser) <-chan feed.IPSet
	Close()
}
