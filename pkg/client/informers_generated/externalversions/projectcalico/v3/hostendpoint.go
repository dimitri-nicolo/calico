// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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

// HostEndpointInformer provides access to a shared informer and lister for
// HostEndpoints.
type HostEndpointInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v3.HostEndpointLister
}

type hostEndpointInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewHostEndpointInformer constructs a new informer for HostEndpoint type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewHostEndpointInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredHostEndpointInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredHostEndpointInformer constructs a new informer for HostEndpoint type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredHostEndpointInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().HostEndpoints().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().HostEndpoints().Watch(options)
			},
		},
		&projectcalicov3.HostEndpoint{},
		resyncPeriod,
		indexers,
	)
}

func (f *hostEndpointInformer) defaultInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredHostEndpointInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *hostEndpointInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalicov3.HostEndpoint{}, f.defaultInformer)
}

func (f *hostEndpointInformer) Lister() v3.HostEndpointLister {
	return v3.NewHostEndpointLister(f.Informer().GetIndexer())
}
