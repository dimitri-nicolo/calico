// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package internalversion

import (
	"context"
	time "time"

	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	internalclientset "github.com/tigera/apiserver/pkg/client/clientset_generated/internalclientset"
	internalinterfaces "github.com/tigera/apiserver/pkg/client/informers_generated/internalversion/internalinterfaces"
	internalversion "github.com/tigera/apiserver/pkg/client/listers_generated/projectcalico/internalversion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// GlobalThreatFeedInformer provides access to a shared informer and lister for
// GlobalThreatFeeds.
type GlobalThreatFeedInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.GlobalThreatFeedLister
}

type globalThreatFeedInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewGlobalThreatFeedInformer constructs a new informer for GlobalThreatFeed type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewGlobalThreatFeedInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredGlobalThreatFeedInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredGlobalThreatFeedInformer constructs a new informer for GlobalThreatFeed type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredGlobalThreatFeedInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().GlobalThreatFeeds().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().GlobalThreatFeeds().Watch(context.TODO(), options)
			},
		},
		&projectcalico.GlobalThreatFeed{},
		resyncPeriod,
		indexers,
	)
}

func (f *globalThreatFeedInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredGlobalThreatFeedInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *globalThreatFeedInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.GlobalThreatFeed{}, f.defaultInformer)
}

func (f *globalThreatFeedInformer) Lister() internalversion.GlobalThreatFeedLister {
	return internalversion.NewGlobalThreatFeedLister(f.Informer().GetIndexer())
}
