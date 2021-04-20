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

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
)

// NetworkSetLister helps list NetworkSets.
// All objects returned here must be treated as read-only.
type NetworkSetLister interface {
	// List lists all NetworkSets in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.NetworkSet, err error)
	// NetworkSets returns an object that can list and get NetworkSets.
	NetworkSets(namespace string) NetworkSetNamespaceLister
	NetworkSetListerExpansion
}

// networkSetLister implements the NetworkSetLister interface.
type networkSetLister struct {
	indexer cache.Indexer
}

// NewNetworkSetLister returns a new NetworkSetLister.
func NewNetworkSetLister(indexer cache.Indexer) NetworkSetLister {
	return &networkSetLister{indexer: indexer}
}

// List lists all NetworkSets in the indexer.
func (s *networkSetLister) List(selector labels.Selector) (ret []*v3.NetworkSet, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.NetworkSet))
	})
	return ret, err
}

// NetworkSets returns an object that can list and get NetworkSets.
func (s *networkSetLister) NetworkSets(namespace string) NetworkSetNamespaceLister {
	return networkSetNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// NetworkSetNamespaceLister helps list and get NetworkSets.
// All objects returned here must be treated as read-only.
type NetworkSetNamespaceLister interface {
	// List lists all NetworkSets in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.NetworkSet, err error)
	// Get retrieves the NetworkSet from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.NetworkSet, error)
	NetworkSetNamespaceListerExpansion
}

// networkSetNamespaceLister implements the NetworkSetNamespaceLister
// interface.
type networkSetNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all NetworkSets in the indexer for a given namespace.
func (s networkSetNamespaceLister) List(selector labels.Selector) (ret []*v3.NetworkSet, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.NetworkSet))
	})
	return ret, err
}

// Get retrieves the NetworkSet from the indexer for a given namespace and name.
func (s networkSetNamespaceLister) Get(name string) (*v3.NetworkSet, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("networkset"), name)
	}
	return obj.(*v3.NetworkSet), nil
}
