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

// ClusterInformationInformer provides access to a shared informer and lister for
// ClusterInformations.
type ClusterInformationInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.ClusterInformationLister
}

type clusterInformationInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterInformationInformer constructs a new informer for ClusterInformation type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterInformationInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterInformationInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterInformationInformer constructs a new informer for ClusterInformation type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterInformationInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().ClusterInformations().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().ClusterInformations().Watch(options)
			},
		},
		&projectcalico.ClusterInformation{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterInformationInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterInformationInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterInformationInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.ClusterInformation{}, f.defaultInformer)
}

func (f *clusterInformationInformer) Lister() internalversion.ClusterInformationLister {
	return internalversion.NewClusterInformationLister(f.Informer().GetIndexer())
}
