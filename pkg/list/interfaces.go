// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package list

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Source is the interface used for listing the current configured resources from source.
type Source interface {
	RetrieveList(kind schema.GroupVersionKind) (*TimestampedResourceList, error)
}

// Destination is the interface used for managing the archived time-dependent resource lists.
type Destination interface {
	RetrieveList(kind schema.GroupVersionKind, from, to *time.Time, sortAscendingTime bool) (*TimestampedResourceList, error)
	StoreList(schema.GroupVersionKind, *TimestampedResourceList) error
}
