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

package v1beta1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	apiregistration_v1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	clientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	internalinterfaces "k8s.io/kube-aggregator/pkg/client/informers/externalversions/internalinterfaces"
	v1beta1 "k8s.io/kube-aggregator/pkg/client/listers/apiregistration/v1beta1"
	time "time"
)

// APIServiceInformer provides access to a shared informer and lister for
// APIServices.
type APIServiceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta1.APIServiceLister
}

type aPIServiceInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

func newAPIServiceInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	sharedIndexInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.ApiregistrationV1beta1().APIServices().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.ApiregistrationV1beta1().APIServices().Watch(options)
			},
		},
		&apiregistration_v1beta1.APIService{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	return sharedIndexInformer
}

func (f *aPIServiceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apiregistration_v1beta1.APIService{}, newAPIServiceInformer)
}

func (f *aPIServiceInformer) Lister() v1beta1.APIServiceLister {
	return v1beta1.NewAPIServiceLister(f.Informer().GetIndexer())
}
