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

package internalversion

import (
	"context"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"

	projectcalico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
	internalclientset "github.com/projectcalico/apiserver/pkg/client/clientset_generated/internalclientset"
	internalinterfaces "github.com/projectcalico/apiserver/pkg/client/informers_generated/internalversion/internalinterfaces"
	internalversion "github.com/projectcalico/apiserver/pkg/client/listers_generated/projectcalico/internalversion"
)

// ProfileInformer provides access to a shared informer and lister for
// Profiles.
type ProfileInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.ProfileLister
}

type profileInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewProfileInformer constructs a new informer for Profile type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewProfileInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredProfileInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredProfileInformer constructs a new informer for Profile type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredProfileInformer(client internalclientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().Profiles().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Projectcalico().Profiles().Watch(context.TODO(), options)
			},
		},
		&projectcalico.Profile{},
		resyncPeriod,
		indexers,
	)
}

func (f *profileInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredProfileInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *profileInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&projectcalico.Profile{}, f.defaultInformer)
}

func (f *profileInformer) Lister() internalversion.ProfileLister {
	return internalversion.NewProfileLister(f.Informer().GetIndexer())
}
