// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/events"
)

type IPSet interface {
	// Put a set of IPs in the database. IPs are sent as strings to avoid
	// overhead of decoding and encoding net.IP, since they are strings on the
	// wire to elastic.
	PutIPSet(ctx context.Context, name string, set IPSetSpec) error
	GetIPSet(ctx context.Context, name string) (IPSetSpec, error)
	GetIPSetModified(ctx context.Context, name string) (time.Time, error)
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

type IPSetSpec []string
