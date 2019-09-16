// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package list

import (
	lmaL "github.com/tigera/lma/pkg/list"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Source is the interface used for listing the current configured resources from source.
type Source interface {
	RetrieveList(kind metav1.TypeMeta) (*lmaL.TimestampedResourceList, error)
}
