package puller

import (
	"context"
)

type FeedPuller interface {
	Name() string
	Namespace() string
	// Run activates the feed and returns a channel that sends snapshots of the
	// IPs that are considered suspicious.
	Run(ctx context.Context) <-chan []string
}

