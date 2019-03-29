// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package syncer

import (
	"github.com/tigera/compliance/pkg/resources"
)

type SyncerClient interface {
	NewSyncer(cb SyncerCallbacks) Syncer
}

type Syncer interface {
	Start()
}

type SyncerCallbacks interface {
	OnStatusUpdate(status StatusType)
	OnUpdate(update Update)
}

type Update struct {
	Type       UpdateType
	ResourceID resources.ResourceID
	Resource   resources.Resource
}

type UpdateType int64

const (
	UpdateTypeUnknown UpdateType = 1 << iota
	UpdateTypeNew
	UpdateTypeUpdated
	UpdateTypeDeleted
)

type StatusType int8

const (
	StatusTypeInSync StatusType = iota
	StatusTypeComplete
)

func (s StatusType) String() string {
	switch s {
	case StatusTypeInSync:
		return "in-sync"
	case StatusTypeComplete:
		return "complete"
	default:
		return "unknown"
	}
}
