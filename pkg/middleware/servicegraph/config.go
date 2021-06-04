// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"time"
)

type Config struct {
	// The maximum number of entries that we keep in the warm cache.
	ServiceGraphCacheMaxEntries int

	// The time after which cached entries that would be polled in the background are removed after the last time they
	// were accessed. This ensures cached entries are not polled forever if they are not being accessed.
	ServiceGraphCachePolledEntryAgeOut time.Duration

	// The poll loop interval. The time between background polling of all cache entries that require periodic updates.
	ServiceGraphCachePollLoopInterval time.Duration

	// The min time between starting successive background data queries. This is used to ensure we are sending too many
	// requests in quick succession and overwhelming elastic or the kubernetes API. This does not gate user driven
	// queries.
	ServiceGraphCachePollQueryInterval time.Duration

	// The max time we expect it to take for data to be collected and stored in elastic. This is used to deterine
	// whether a cache entry should be background polled for updates.
	ServiceGraphCacheDataSettleTime time.Duration
}
