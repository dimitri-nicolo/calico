// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package v3

import (
	"context"
	time "time"

	projectcalicov3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	clientset "github.com/tigera/apiserver/pkg/client/clientset_generated/clientset"
	internalinterfaces "github.com/tigera/apiserver/pkg/client/informers_generated/externalversions/internalinterfaces"
	v3 "github.com/tigera/apiserver/pkg/client/listers_generated/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// StagedKubernetesNetworkPolicyInformer provides access to a shared informer and lister for
// StagedKubernetesNetworkPolicies.
type StagedKubernetesNetworkPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v3.StagedKubernetesNetworkPolicyLister
}

type stagedKubernetesNetworkPolicyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewStagedKubernetesNetworkPolicyInformer constructs a new informer for StagedKubernetesNetworkPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewStagedKubernetesNetworkPolicyInformer(client clientset.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredStagedKubernetesNetworkPolicyInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredStagedKubernetesNetworkPolicyInformer constructs a new informer for StagedKubernetesNetworkPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredStagedKubernetesNetworkPolicyInformer(client clientset.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().StagedKubernetesNetworkPolicies(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().StagedKubernetesNetworkPolicies(namespace).Watch(context.TODO(), options)
			},
		},
		&projectcalicov3.StagedKubernetesNetworkPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *stagedKubernetesNetworkPolicyInformer) defaultInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredStagedKubernetesNetworkPolicyInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *stagedKubernetesNetworkPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalicov3.StagedKubernetesNetworkPolicy{}, f.defaultInformer)
}

func (f *stagedKubernetesNetworkPolicyInformer) Lister() v3.StagedKubernetesNetworkPolicyLister {
	return v3.NewStagedKubernetesNetworkPolicyLister(f.Informer().GetIndexer())
}
