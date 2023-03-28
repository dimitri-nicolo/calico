// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package api

import (
	"context"
	"time"

	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

var EventConfigurationVerbs = []v1.Verb{v1.Create, v1.Update, v1.Patch, v1.Delete}

type AuditEventResult struct {
	*auditv1.Event
	Err error
}

type ReportEventFetcher interface {
	GetAuditEvents(context.Context, *time.Time, *time.Time) <-chan *AuditEventResult
}
