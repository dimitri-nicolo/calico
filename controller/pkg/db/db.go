package db

import (
	"context"

	"github.com/tigera/intrusion-detection/controller/pkg/events"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
)

type IPSet interface {
	// Put a set of IPs in the database. IPs are sent as strings to avoid
	// overhead of decoding and encoding net.IP, since they are strings on the
	// wire to elastic.
	PutIPSet(ctx context.Context, name string, set feed.IPSet) error
	GetIPSet(name string) ([]string, error)
}

type SuspiciousIP interface {
	QueryIPSet(ctx context.Context, name string) (SecurityEventIterator, error)
}

type SecurityEventIterator interface {
	Next() bool
	Value() events.SecurityEvent
	Err() error
}

type Events interface {
	PutSecurityEvent(context.Context, events.SecurityEvent) error
}
