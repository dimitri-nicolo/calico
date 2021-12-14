// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.

package etcd

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
)

// Options is the set of options necessary for creating etcd-backed storage
type Options struct {
	RESTOptions   generic.RESTOptions
	Capacity      int
	ObjectType    runtime.Object
	ScopeStrategy rest.NamespaceScopedStrategy
	NewListFunc   func() runtime.Object
	GetAttrsFunc  func(runtime.Object) (labels.Set, fields.Set, error)
	Trigger       storage.IndexerFunc
}
