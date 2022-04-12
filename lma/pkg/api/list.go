// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package api

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/lma/pkg/list"
)

// Destination is the interface used for managing the archived time-dependent resource lists.
type ListDestination interface {
	RetrieveList(kind metav1.TypeMeta, from, to *time.Time, sortAscendingTime bool) (*list.TimestampedResourceList, error)
	StoreList(metav1.TypeMeta, *list.TimestampedResourceList) error
}
