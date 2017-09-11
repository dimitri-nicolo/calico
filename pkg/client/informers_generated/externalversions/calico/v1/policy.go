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

package v1

import (
	calico_v1 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1"
	clientset "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"
	internalinterfaces "github.com/tigera/calico-k8sapiserver/pkg/client/informers_generated/externalversions/internalinterfaces"
	v1 "github.com/tigera/calico-k8sapiserver/pkg/client/listers_generated/calico/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	time "time"
)

// PolicyInformer provides access to a shared informer and lister for
// Policies.
type PolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.PolicyLister
}

type policyInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

func newPolicyInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	sharedIndexInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.CalicoV1().Policies(meta_v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.CalicoV1().Policies(meta_v1.NamespaceAll).Watch(options)
			},
		},
		&calico_v1.Policy{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	return sharedIndexInformer
}

func (f *policyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&calico_v1.Policy{}, newPolicyInformer)
}

func (f *policyInformer) Lister() v1.PolicyLister {
	return v1.NewPolicyLister(f.Informer().GetIndexer())
}
