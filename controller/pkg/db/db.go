// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"time"
)

type Meta struct {
	Name    string
	Version *int64
	Kind    Kind
}

type Kind string

const (
	KindIPSet         Kind = "IPSet"
	KindDomainNameSet Kind = "DomainNameSet"
)

type Sets interface {
	PutSet(ctx context.Context, meta Meta, value interface{}) error
	GetIPSet(ctx context.Context, name string) (IPSetSpec, error)
	GetIPSetModified(ctx context.Context, name string) (time.Time, error)
	ListSets(ctx context.Context, kind Kind) ([]Meta, error)
	DeleteSet(ctx context.Context, m Meta) error
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

// IPs are sent as strings to avoid overhead of decoding and encoding net.IP, since they are strings on the
// wire to elastic.
type IPSetSpec []string

type DomainNameSetSpec []string
