// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"time"
)

type IPSetMeta struct {
	Name    string
	Version *int64
}

type IPSet interface {
	// Put a set of IPs in the database. IPs are sent as strings to avoid
	// overhead of decoding and encoding net.IP, since they are strings on the
	// wire to elastic.
	PutIPSet(ctx context.Context, name string, set IPSetSpec) error
	GetIPSet(ctx context.Context, name string) (IPSetSpec, error)
	GetIPSetModified(ctx context.Context, name string) (time.Time, error)
	ListIPSets(ctx context.Context) ([]IPSetMeta, error)
	DeleteIPSet(ctx context.Context, m IPSetMeta) error
}

type SuspiciousIP interface {
	QueryIPSet(ctx context.Context, name string) (SecurityEventIterator, error)
}

type SecurityEventInterface interface {
	ID() string
}

type SecurityEventIterator interface {
	Next() bool
	Value() SecurityEventInterface
	Err() error
}

type Events interface {
	PutSecurityEvent(context.Context, SecurityEventInterface) error
}

type AuditLog interface {
	ObjectCreatedBetween(ctx context.Context, kind, namespace, name string, before, after time.Time) (bool, error)
	ObjectDeletedBetween(ctx context.Context, kind, namespace, name string, before, after time.Time) (bool, error)
}

type IPSetSpec []string
