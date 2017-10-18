/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was automatically generated by informer-gen

package v2

import (
	calico_v2 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v2"
	clientset "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"
	internalinterfaces "github.com/tigera/calico-k8sapiserver/pkg/client/informers_generated/externalversions/internalinterfaces"
	v2 "github.com/tigera/calico-k8sapiserver/pkg/client/listers_generated/projectcalico/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	time "time"
)

// GlobalNetworkPolicyInformer provides access to a shared informer and lister for
// GlobalNetworkPolicies.
type GlobalNetworkPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v2.GlobalNetworkPolicyLister
}

type globalNetworkPolicyInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

func newGlobalNetworkPolicyInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	sharedIndexInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.ProjectcalicoV2().GlobalNetworkPolicies().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.ProjectcalicoV2().GlobalNetworkPolicies().Watch(options)
			},
		},
		&calico_v2.GlobalNetworkPolicy{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	return sharedIndexInformer
}

func (f *globalNetworkPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&calico_v2.GlobalNetworkPolicy{}, newGlobalNetworkPolicyInformer)
}

func (f *globalNetworkPolicyInformer) Lister() v2.GlobalNetworkPolicyLister {
	return v2.NewGlobalNetworkPolicyLister(f.Informer().GetIndexer())
}
