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

// BGPConfigurationInformer provides access to a shared informer and lister for
// BGPConfigurations.
type BGPConfigurationInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.BGPConfigurationLister
}

type bGPConfigurationInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewBGPConfigurationInformer constructs a new informer for BGPConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewBGPConfigurationInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredBGPConfigurationInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredBGPConfigurationInformer constructs a new informer for BGPConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredBGPConfigurationInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().BGPConfigurations().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().BGPConfigurations().Watch(options)
			},
		},
		&projectcalico.BGPConfiguration{},
		resyncPeriod,
		indexers,
	)
}

func (f *bGPConfigurationInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredBGPConfigurationInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *bGPConfigurationInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.BGPConfiguration{}, f.defaultInformer)
}

func (f *bGPConfigurationInformer) Lister() internalversion.BGPConfigurationLister {
	return internalversion.NewBGPConfigurationLister(f.Informer().GetIndexer())
}
