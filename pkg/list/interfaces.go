// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package list

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Source is the interface used for listing the current configured resources from source.
type Source interface {
	RetrieveList(kind metav1.TypeMeta) (*TimestampedResourceList, error)
}

// Destination is the interface used for managing the archived time-dependent resource lists.
type Destination interface {
	RetrieveList(kind metav1.TypeMeta, from, to *time.Time, sortAscendingTime bool) (*TimestampedResourceList, error)
	StoreList(metav1.TypeMeta, *TimestampedResourceList) error
}
