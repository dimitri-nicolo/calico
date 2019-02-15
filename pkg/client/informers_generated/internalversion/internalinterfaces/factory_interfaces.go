// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package internalinterfaces

import (
	time "time"

	internalclientset "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/internalclientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	cache "k8s.io/client-go/tools/cache"
)

type NewInformerFunc func(internalclientset.Interface, time.Duration) cache.SharedIndexInformer

// SharedInformerFactory a small interface to allow for adding an informer without an import cycle
type SharedInformerFactory interface {
	Start(stopCh <-chan struct{})
	InformerFor(obj runtime.Object, newFunc NewInformerFunc) cache.SharedIndexInformer
}

type TweakListOptionsFunc func(*v1.ListOptions)
