// Copyright (c) 2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by informer-gen. DO NOT EDIT.

package v3

import (
	"context"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"

	projectcalicov3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	clientset "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset"
	internalinterfaces "github.com/projectcalico/apiserver/pkg/client/informers_generated/externalversions/internalinterfaces"
	v3 "github.com/projectcalico/apiserver/pkg/client/listers_generated/projectcalico/v3"
)

// GlobalAlertTemplateInformer provides access to a shared informer and lister for
// GlobalAlertTemplates.
type GlobalAlertTemplateInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v3.GlobalAlertTemplateLister
}

type globalAlertTemplateInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewGlobalAlertTemplateInformer constructs a new informer for GlobalAlertTemplate type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewGlobalAlertTemplateInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredGlobalAlertTemplateInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredGlobalAlertTemplateInformer constructs a new informer for GlobalAlertTemplate type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredGlobalAlertTemplateInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().GlobalAlertTemplates().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ProjectcalicoV3().GlobalAlertTemplates().Watch(context.TODO(), options)
			},
		},
		&projectcalicov3.GlobalAlertTemplate{},
		resyncPeriod,
		indexers,
	)
}

func (f *globalAlertTemplateInformer) defaultInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredGlobalAlertTemplateInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *globalAlertTemplateInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalicov3.GlobalAlertTemplate{}, f.defaultInformer)
}

func (f *globalAlertTemplateInformer) Lister() v3.GlobalAlertTemplateLister {
	return v3.NewGlobalAlertTemplateLister(f.Informer().GetIndexer())
}
