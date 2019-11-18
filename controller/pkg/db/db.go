// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"time"
)

type Meta struct {
	Name        string
	SeqNo       *int64
	PrimaryTerm *int64
}

type IPSet interface {
	PutIPSet(ctx context.Context, name string, set IPSetSpec) error
	GetIPSet(ctx context.Context, name string) (IPSetSpec, error)
	GetIPSetModified(ctx context.Context, name string) (time.Time, error)
	ListIPSets(ctx context.Context) ([]Meta, error)
	DeleteIPSet(ctx context.Context, m Meta) error
}

type DomainNameSet interface {
	PutDomainNameSet(ctx context.Context, name string, set DomainNameSetSpec) error
	GetDomainNameSetModified(ctx context.Context, name string) (time.Time, error)
	ListDomainNameSets(ctx context.Context) ([]Meta, error)
	DeleteDomainNameSet(ctx context.Context, m Meta) error
}

type SuspiciousSet interface {
	QuerySet(ctx context.Context, name string) ([]SecurityEventInterface, error)
}

type SecurityEventInterface interface {
	ID() string
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

type QueryKey int

const (
	QueryKeyUnknown QueryKey = iota
	QueryKeyFlowLogSourceIP
	QueryKeyFlowLogDestIP
	QueryKeyDNSLogQName
	QueryKeyDNSLogRRSetsName
	QueryKeyDNSLogRRSetsRData
)
