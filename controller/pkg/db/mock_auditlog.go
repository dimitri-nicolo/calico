// Copyright 2019 Tigera Inc. All rights reserved.

package db

import "time"

type MockAuditLog struct {
	Ok  bool
	Err error
}

func (al *MockAuditLog) ObjectCreatedBetween(kind, namespace, name string, before, after time.Time) (bool, error) {
	return al.Ok, al.Err
}

func (al *MockAuditLog) ObjectDeletedBetween(kind, namespace, name string, before, after time.Time) (bool, error) {
	return al.Ok, al.Err
}
