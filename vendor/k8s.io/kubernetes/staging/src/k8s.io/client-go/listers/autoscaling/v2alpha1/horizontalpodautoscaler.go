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

// This file was automatically generated by lister-gen

package v2alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	v2alpha1 "k8s.io/client-go/pkg/apis/autoscaling/v2alpha1"
	"k8s.io/client-go/tools/cache"
)

// HorizontalPodAutoscalerLister helps list HorizontalPodAutoscalers.
type HorizontalPodAutoscalerLister interface {
	// List lists all HorizontalPodAutoscalers in the indexer.
	List(selector labels.Selector) (ret []*v2alpha1.HorizontalPodAutoscaler, err error)
	// HorizontalPodAutoscalers returns an object that can list and get HorizontalPodAutoscalers.
	HorizontalPodAutoscalers(namespace string) HorizontalPodAutoscalerNamespaceLister
	HorizontalPodAutoscalerListerExpansion
}

// horizontalPodAutoscalerLister implements the HorizontalPodAutoscalerLister interface.
type horizontalPodAutoscalerLister struct {
	indexer cache.Indexer
}

// NewHorizontalPodAutoscalerLister returns a new HorizontalPodAutoscalerLister.
func NewHorizontalPodAutoscalerLister(indexer cache.Indexer) HorizontalPodAutoscalerLister {
	return &horizontalPodAutoscalerLister{indexer: indexer}
}

// List lists all HorizontalPodAutoscalers in the indexer.
func (s *horizontalPodAutoscalerLister) List(selector labels.Selector) (ret []*v2alpha1.HorizontalPodAutoscaler, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v2alpha1.HorizontalPodAutoscaler))
	})
	return ret, err
}

// HorizontalPodAutoscalers returns an object that can list and get HorizontalPodAutoscalers.
func (s *horizontalPodAutoscalerLister) HorizontalPodAutoscalers(namespace string) HorizontalPodAutoscalerNamespaceLister {
	return horizontalPodAutoscalerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// HorizontalPodAutoscalerNamespaceLister helps list and get HorizontalPodAutoscalers.
type HorizontalPodAutoscalerNamespaceLister interface {
	// List lists all HorizontalPodAutoscalers in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v2alpha1.HorizontalPodAutoscaler, err error)
	// Get retrieves the HorizontalPodAutoscaler from the indexer for a given namespace and name.
	Get(name string) (*v2alpha1.HorizontalPodAutoscaler, error)
	HorizontalPodAutoscalerNamespaceListerExpansion
}

// horizontalPodAutoscalerNamespaceLister implements the HorizontalPodAutoscalerNamespaceLister
// interface.
type horizontalPodAutoscalerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all HorizontalPodAutoscalers in the indexer for a given namespace.
func (s horizontalPodAutoscalerNamespaceLister) List(selector labels.Selector) (ret []*v2alpha1.HorizontalPodAutoscaler, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v2alpha1.HorizontalPodAutoscaler))
	})
	return ret, err
}

// Get retrieves the HorizontalPodAutoscaler from the indexer for a given namespace and name.
func (s horizontalPodAutoscalerNamespaceLister) Get(name string) (*v2alpha1.HorizontalPodAutoscaler, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v2alpha1.Resource("horizontalpodautoscaler"), name)
	}
	return obj.(*v2alpha1.HorizontalPodAutoscaler), nil
}
