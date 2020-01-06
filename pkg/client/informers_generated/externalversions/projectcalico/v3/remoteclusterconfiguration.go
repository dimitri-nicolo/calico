// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package v3

import (
	time "time"

	projectcalicov3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	clientset "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"
	internalinterfaces "github.com/tigera/calico-k8sapiserver/pkg/client/informers_generated/externalversions/internalinterfaces"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/client/listers_generated/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// RemoteClusterConfigurationInformer provides access to a shared informer and lister for
// RemoteClusterConfigurations.
type RemoteClusterConfigurationInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v3.RemoteClusterConfigurationLister
}

type remoteClusterConfigurationInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewRemoteClusterConfigurationInformer constructs a new informer for RemoteClusterConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRemoteClusterConfigurationInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredRemoteClusterConfigurationInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredRemoteClusterConfigurationInformer constructs a new informer for RemoteClusterConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRemoteClusterConfigurationInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().RemoteClusterConfigurations().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().RemoteClusterConfigurations().Watch(options)
			},
		},
		&projectcalicov3.RemoteClusterConfiguration{},
		resyncPeriod,
		indexers,
	)
}

func (f *remoteClusterConfigurationInformer) defaultInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredRemoteClusterConfigurationInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *remoteClusterConfigurationInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalicov3.RemoteClusterConfiguration{}, f.defaultInformer)
}

func (f *remoteClusterConfigurationInformer) Lister() v3.RemoteClusterConfigurationLister {
	return v3.NewRemoteClusterConfigurationLister(f.Informer().GetIndexer())
}
