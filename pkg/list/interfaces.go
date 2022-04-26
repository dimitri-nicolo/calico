// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package list

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lmaL "github.com/tigera/lma/pkg/list"
)

// Source is the interface used for listing the current configured resources from source.
type Source interface {
	RetrieveList(kind metav1.TypeMeta) (*lmaL.TimestampedResourceList, error)
}
