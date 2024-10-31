// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package syncer

import (
	"context"
	"fmt"
	"strconv"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/resources"
)

type Starter interface {
	Start(context.Context)
}

type SyncerCallbacks interface {
	OnStatusUpdate(status StatusUpdate)
	OnUpdates(updates []Update)
}

type Update struct {
	Type       UpdateType
	ResourceID apiv3.ResourceID
	Resource   resources.Resource
}

type UpdateType int64

// The default set of update types. Note that the xref cache uses it's own set of update type values, and specifies
// the updates as a set of bit values (i.e. a single update may contain multiple updates in one).
const (
	UpdateTypeUnknown UpdateType = 1 << iota
	UpdateTypeSet
	UpdateTypeDeleted
)

func (t UpdateType) String() string {
	return strconv.FormatInt(int64(t), 2)
}

type StatusUpdate struct {
	Type  StatusType
	Error error
}

func NewStatusUpdateComplete() StatusUpdate {
	return StatusUpdate{StatusTypeComplete, nil}
}

func NewStatusUpdateInSync() StatusUpdate {
	return StatusUpdate{StatusTypeInSync, nil}
}

func NewStatusUpdateFailed(err error) StatusUpdate {
	return StatusUpdate{StatusTypeFailed, err}
}

func (s StatusUpdate) String() string {
	switch s.Type {
	case StatusTypeInSync:
		return "in-sync"
	case StatusTypeComplete:
		return "complete"
	case StatusTypeFailed:
		return fmt.Sprintf("failed: %v", s.Error)
	default:
		return "unknown"
	}
}

type StatusType int8

const (
	StatusTypeInSync StatusType = iota
	StatusTypeComplete
	StatusTypeFailed
)
