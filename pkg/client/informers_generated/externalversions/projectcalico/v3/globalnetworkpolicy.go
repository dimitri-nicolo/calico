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

// GlobalNetworkPolicyInformer provides access to a shared informer and lister for
// GlobalNetworkPolicies.
type GlobalNetworkPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v3.GlobalNetworkPolicyLister
}

type globalNetworkPolicyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewGlobalNetworkPolicyInformer constructs a new informer for GlobalNetworkPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewGlobalNetworkPolicyInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredGlobalNetworkPolicyInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredGlobalNetworkPolicyInformer constructs a new informer for GlobalNetworkPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredGlobalNetworkPolicyInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().GlobalNetworkPolicies().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().GlobalNetworkPolicies().Watch(options)
			},
		},
		&projectcalicov3.GlobalNetworkPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *globalNetworkPolicyInformer) defaultInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredGlobalNetworkPolicyInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *globalNetworkPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalicov3.GlobalNetworkPolicy{}, f.defaultInformer)
}

func (f *globalNetworkPolicyInformer) Lister() v3.GlobalNetworkPolicyLister {
	return v3.NewGlobalNetworkPolicyLister(f.Informer().GetIndexer())
}
