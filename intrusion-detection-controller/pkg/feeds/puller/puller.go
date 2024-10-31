// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/cacher"
)

type SyncFailFunction func(error)

type Puller interface {
	// Run activates the puller to start pulling from the feed.
	Run(context.Context, cacher.GlobalThreatFeedCacher)

	// SetFeed updates the feed the puller should use.
	SetFeed(*v3.GlobalThreatFeed)

	// Close stops the puller and ends its goroutines
	Close()
}
