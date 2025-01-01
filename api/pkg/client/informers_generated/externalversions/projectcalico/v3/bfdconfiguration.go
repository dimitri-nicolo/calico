// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package v3

import (
	"context"
	time "time"

	projectcalicov3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	clientset "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	internalinterfaces "github.com/tigera/api/pkg/client/informers_generated/externalversions/internalinterfaces"
	v3 "github.com/tigera/api/pkg/client/listers_generated/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// BFDConfigurationInformer provides access to a shared informer and lister for
// BFDConfigurations.
type BFDConfigurationInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v3.BFDConfigurationLister
}

type bFDConfigurationInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewBFDConfigurationInformer constructs a new informer for BFDConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewBFDConfigurationInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredBFDConfigurationInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredBFDConfigurationInformer constructs a new informer for BFDConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredBFDConfigurationInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().BFDConfigurations().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().BFDConfigurations().Watch(context.TODO(), options)
			},
		},
		&projectcalicov3.BFDConfiguration{},
		resyncPeriod,
		indexers,
	)
}

func (f *bFDConfigurationInformer) defaultInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredBFDConfigurationInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *bFDConfigurationInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalicov3.BFDConfiguration{}, f.defaultInformer)
}

func (f *bFDConfigurationInformer) Lister() v3.BFDConfigurationLister {
	return v3.NewBFDConfigurationLister(f.Informer().GetIndexer())
}
