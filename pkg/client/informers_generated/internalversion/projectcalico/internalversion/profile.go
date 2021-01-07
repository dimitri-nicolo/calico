// Copyright (c) 2021 Tigera, Inc. All rights reserved.

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

// ProfileInformer provides access to a shared informer and lister for
// Profiles.
type ProfileInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.ProfileLister
}

type profileInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewProfileInformer constructs a new informer for Profile type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewProfileInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredProfileInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredProfileInformer constructs a new informer for Profile type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredProfileInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().Profiles().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().Profiles().Watch(context.TODO(), options)
			},
		},
		&projectcalico.Profile{},
		resyncPeriod,
		indexers,
	)
}

func (f *profileInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredProfileInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *profileInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.Profile{}, f.defaultInformer)
}

func (f *profileInformer) Lister() internalversion.ProfileLister {
	return internalversion.NewProfileLister(f.Informer().GetIndexer())
}
