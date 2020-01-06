// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package internalversion

import (
	time "time"

	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	internalclientset "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/internalclientset"
	internalinterfaces "github.com/tigera/calico-k8sapiserver/pkg/client/informers_generated/internalversion/internalinterfaces"
	internalversion "github.com/tigera/calico-k8sapiserver/pkg/client/listers_generated/projectcalico/internalversion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// GlobalAlertTemplateInformer provides access to a shared informer and lister for
// GlobalAlertTemplates.
type GlobalAlertTemplateInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.GlobalAlertTemplateLister
}

type globalAlertTemplateInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewGlobalAlertTemplateInformer constructs a new informer for GlobalAlertTemplate type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewGlobalAlertTemplateInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredGlobalAlertTemplateInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredGlobalAlertTemplateInformer constructs a new informer for GlobalAlertTemplate type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredGlobalAlertTemplateInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().GlobalAlertTemplates().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().GlobalAlertTemplates().Watch(options)
			},
		},
		&projectcalico.GlobalAlertTemplate{},
		resyncPeriod,
		indexers,
	)
}

func (f *globalAlertTemplateInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredGlobalAlertTemplateInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *globalAlertTemplateInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.GlobalAlertTemplate{}, f.defaultInformer)
}

func (f *globalAlertTemplateInformer) Lister() internalversion.GlobalAlertTemplateLister {
	return internalversion.NewGlobalAlertTemplateLister(f.Informer().GetIndexer())
}
