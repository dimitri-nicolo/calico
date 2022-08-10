// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.
package api

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Resource interface {
	runtime.Object
	v1.ObjectMetaAccessor
}
