// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package syncer

import (
	"fmt"

	"github.com/tigera/compliance/pkg/resources"
)

type SyncerClient interface {
	NewSyncer(cb SyncerCallbacks) Syncer
}

type Syncer interface {
	Start()
}

type SyncerCallbacks interface {
	OnStatusUpdate(status StatusUpdate)
	OnUpdate(update Update)
}

type Update struct {
	Type       UpdateType
	ResourceID resources.ResourceID
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

type StatusUpdate struct {
	Type  StatusType
	Error error
}

func (s StatusUpdate) String() string {
	switch s.Type {
	case StatusTypeInSync:
		return "in-sync"
	case StatusTypeComplete:
		return "complete"
	case StatusUpdateFailed:
		return fmt.Sprintf("failed: %v", s.Error)
	default:
		return "unknown"
	}
}

type StatusType int8

const (
	StatusTypeInSync StatusType = iota
	StatusTypeComplete
	StatusUpdateFailed
)
