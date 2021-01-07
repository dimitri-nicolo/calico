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

// StagedGlobalNetworkPolicyInformer provides access to a shared informer and lister for
// StagedGlobalNetworkPolicies.
type StagedGlobalNetworkPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.StagedGlobalNetworkPolicyLister
}

type stagedGlobalNetworkPolicyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewStagedGlobalNetworkPolicyInformer constructs a new informer for StagedGlobalNetworkPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewStagedGlobalNetworkPolicyInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredStagedGlobalNetworkPolicyInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredStagedGlobalNetworkPolicyInformer constructs a new informer for StagedGlobalNetworkPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredStagedGlobalNetworkPolicyInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().StagedGlobalNetworkPolicies().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().StagedGlobalNetworkPolicies().Watch(context.TODO(), options)
			},
		},
		&projectcalico.StagedGlobalNetworkPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *stagedGlobalNetworkPolicyInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredStagedGlobalNetworkPolicyInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *stagedGlobalNetworkPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.StagedGlobalNetworkPolicy{}, f.defaultInformer)
}

func (f *stagedGlobalNetworkPolicyInformer) Lister() internalversion.StagedGlobalNetworkPolicyLister {
	return internalversion.NewStagedGlobalNetworkPolicyLister(f.Informer().GetIndexer())
}
