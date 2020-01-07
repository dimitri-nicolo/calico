// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package projectcalico

import (
	time "time"

	apisprojectcalico "github.com/tigera/api/pkg/apis/projectcalico"
	clientset "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	internalinterfaces "github.com/tigera/api/pkg/client/informers_generated/externalversions/internalinterfaces"
	projectcalico "github.com/tigera/api/pkg/client/listers_generated/apis/projectcalico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// GlobalReportTypeInformer provides access to a shared informer and lister for
// GlobalReportTypes.
type GlobalReportTypeInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() projectcalico.GlobalReportTypeLister
}

type globalReportTypeInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewGlobalReportTypeInformer constructs a new informer for GlobalReportType type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewGlobalReportTypeInformer(client clientset.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredGlobalReportTypeInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredGlobalReportTypeInformer constructs a new informer for GlobalReportType type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredGlobalReportTypeInformer(client clientset.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoProjectcalico().GlobalReportTypes(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoProjectcalico().GlobalReportTypes(namespace).Watch(options)
			},
		},
		&apisprojectcalico.GlobalReportType{},
		resyncPeriod,
		indexers,
	)
}

func (f *globalReportTypeInformer) defaultInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredGlobalReportTypeInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *globalReportTypeInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisprojectcalico.GlobalReportType{}, f.defaultInformer)
}

func (f *globalReportTypeInformer) Lister() projectcalico.GlobalReportTypeLister {
	return projectcalico.NewGlobalReportTypeLister(f.Informer().GetIndexer())
}
