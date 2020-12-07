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

// NetworkPolicyInformer provides access to a shared informer and lister for
// NetworkPolicies.
type NetworkPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.NetworkPolicyLister
}

type networkPolicyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewNetworkPolicyInformer constructs a new informer for NetworkPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewNetworkPolicyInformer(client internalclientset.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredNetworkPolicyInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredNetworkPolicyInformer constructs a new informer for NetworkPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredNetworkPolicyInformer(client internalclientset.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().NetworkPolicies(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().NetworkPolicies(namespace).Watch(context.TODO(), options)
			},
		},
		&projectcalico.NetworkPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *networkPolicyInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNetworkPolicyInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *networkPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.NetworkPolicy{}, f.defaultInformer)
}

func (f *networkPolicyInformer) Lister() internalversion.NetworkPolicyLister {
	return internalversion.NewNetworkPolicyLister(f.Informer().GetIndexer())
}
