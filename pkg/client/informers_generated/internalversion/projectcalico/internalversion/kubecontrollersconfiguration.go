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

// KubeControllersConfigurationInformer provides access to a shared informer and lister for
// KubeControllersConfigurations.
type KubeControllersConfigurationInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.KubeControllersConfigurationLister
}

type kubeControllersConfigurationInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewKubeControllersConfigurationInformer constructs a new informer for KubeControllersConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewKubeControllersConfigurationInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredKubeControllersConfigurationInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredKubeControllersConfigurationInformer constructs a new informer for KubeControllersConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredKubeControllersConfigurationInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().KubeControllersConfigurations().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().KubeControllersConfigurations().Watch(context.TODO(), options)
			},
		},
		&projectcalico.KubeControllersConfiguration{},
		resyncPeriod,
		indexers,
	)
}

func (f *kubeControllersConfigurationInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredKubeControllersConfigurationInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *kubeControllersConfigurationInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.KubeControllersConfiguration{}, f.defaultInformer)
}

func (f *kubeControllersConfigurationInformer) Lister() internalversion.KubeControllersConfigurationLister {
	return internalversion.NewKubeControllersConfigurationLister(f.Informer().GetIndexer())
}
