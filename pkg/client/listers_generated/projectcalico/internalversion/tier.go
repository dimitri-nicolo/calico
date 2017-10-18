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

package internalversion

import (
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TierLister helps list Tiers.
type TierLister interface {
	// List lists all Tiers in the indexer.
	List(selector labels.Selector) (ret []*calico.Tier, err error)
	// Tiers returns an object that can list and get Tiers.
	Tiers(namespace string) TierNamespaceLister
	TierListerExpansion
}

// tierLister implements the TierLister interface.
type tierLister struct {
	indexer cache.Indexer
}

// NewTierLister returns a new TierLister.
func NewTierLister(indexer cache.Indexer) TierLister {
	return &tierLister{indexer: indexer}
}

// List lists all Tiers in the indexer.
func (s *tierLister) List(selector labels.Selector) (ret []*calico.Tier, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*calico.Tier))
	})
	return ret, err
}

// Tiers returns an object that can list and get Tiers.
func (s *tierLister) Tiers(namespace string) TierNamespaceLister {
	return tierNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TierNamespaceLister helps list and get Tiers.
type TierNamespaceLister interface {
	// List lists all Tiers in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*calico.Tier, err error)
	// Get retrieves the Tier from the indexer for a given namespace and name.
	Get(name string) (*calico.Tier, error)
	TierNamespaceListerExpansion
}

// tierNamespaceLister implements the TierNamespaceLister
// interface.
type tierNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Tiers in the indexer for a given namespace.
func (s tierNamespaceLister) List(selector labels.Selector) (ret []*calico.Tier, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*calico.Tier))
	})
	return ret, err
}

// Get retrieves the Tier from the indexer for a given namespace and name.
func (s tierNamespaceLister) Get(name string) (*calico.Tier, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(calico.Resource("tier"), name)
	}
	return obj.(*calico.Tier), nil
}
