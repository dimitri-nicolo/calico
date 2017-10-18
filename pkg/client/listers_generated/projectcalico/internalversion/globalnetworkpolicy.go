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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// GlobalNetworkPolicyLister helps list GlobalNetworkPolicies.
type GlobalNetworkPolicyLister interface {
	// List lists all GlobalNetworkPolicies in the indexer.
	List(selector labels.Selector) (ret []*calico.GlobalNetworkPolicy, err error)
	// Get retrieves the GlobalNetworkPolicy from the index for a given name.
	Get(name string) (*calico.GlobalNetworkPolicy, error)
	GlobalNetworkPolicyListerExpansion
}

// globalNetworkPolicyLister implements the GlobalNetworkPolicyLister interface.
type globalNetworkPolicyLister struct {
	indexer cache.Indexer
}

// NewGlobalNetworkPolicyLister returns a new GlobalNetworkPolicyLister.
func NewGlobalNetworkPolicyLister(indexer cache.Indexer) GlobalNetworkPolicyLister {
	return &globalNetworkPolicyLister{indexer: indexer}
}

// List lists all GlobalNetworkPolicies in the indexer.
func (s *globalNetworkPolicyLister) List(selector labels.Selector) (ret []*calico.GlobalNetworkPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*calico.GlobalNetworkPolicy))
	})
	return ret, err
}

// Get retrieves the GlobalNetworkPolicy from the index for a given name.
func (s *globalNetworkPolicyLister) Get(name string) (*calico.GlobalNetworkPolicy, error) {
	key := &calico.GlobalNetworkPolicy{ObjectMeta: v1.ObjectMeta{Name: name}}
	obj, exists, err := s.indexer.Get(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(calico.Resource("globalnetworkpolicy"), name)
	}
	return obj.(*calico.GlobalNetworkPolicy), nil
}
