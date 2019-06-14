// Copyright 2019 Tigera Inc. All rights reserved.

package db

import (
	"context"
	"time"
)

type MockAuditLogArgs struct {
	Kind      string
	Namespace string
	Name      string
	Before    time.Time
	After     time.Time
}

type MockAuditLog struct {
	CreatedOk   bool
	CreatedErr  error
	CreatedArgs *MockAuditLogArgs
	DeletedOk   bool
	DeletedErr  error
	DeletedArgs *MockAuditLogArgs
}

func (al *MockAuditLog) ObjectCreatedBetween(ctx context.Context, kind, namespace, name string, before, after time.Time) (bool, error) {
	al.CreatedArgs = &MockAuditLogArgs{kind, namespace, name, before, after}
	return al.CreatedOk, al.CreatedErr
}

func (al *MockAuditLog) ObjectDeletedBetween(ctx context.Context, kind, namespace, name string, before, after time.Time) (bool, error) {
	al.DeletedArgs = &MockAuditLogArgs{kind, namespace, name, before, after}
	return al.DeletedOk, al.DeletedErr
}
