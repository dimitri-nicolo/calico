// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package api

import (
	"context"
	"time"

	auditv1 "k8s.io/apiserver/pkg/apis/audit"
)

const (
	EventVerbCreate = "create"
	EventVerbUpdate = "update"
	EventVerbPatch  = "patch"
	EventVerbDelete = "delete"
)

var (
	EventConfigurationVerbs = []string{
		EventVerbCreate, EventVerbUpdate, EventVerbPatch, EventVerbDelete,
	}
)

type AuditEventResult struct {
	*auditv1.Event
	Err error
}

type ReportEventFetcher interface {
	GetAuditEvents(context.Context, *time.Time, *time.Time) <-chan *AuditEventResult
}
