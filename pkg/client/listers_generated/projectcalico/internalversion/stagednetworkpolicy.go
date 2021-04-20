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

package internalversion

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	projectcalico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// StagedNetworkPolicyLister helps list StagedNetworkPolicies.
// All objects returned here must be treated as read-only.
type StagedNetworkPolicyLister interface {
	// List lists all StagedNetworkPolicies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*projectcalico.StagedNetworkPolicy, err error)
	// StagedNetworkPolicies returns an object that can list and get StagedNetworkPolicies.
	StagedNetworkPolicies(namespace string) StagedNetworkPolicyNamespaceLister
	StagedNetworkPolicyListerExpansion
}

// stagedNetworkPolicyLister implements the StagedNetworkPolicyLister interface.
type stagedNetworkPolicyLister struct {
	indexer cache.Indexer
}

// NewStagedNetworkPolicyLister returns a new StagedNetworkPolicyLister.
func NewStagedNetworkPolicyLister(indexer cache.Indexer) StagedNetworkPolicyLister {
	return &stagedNetworkPolicyLister{indexer: indexer}
}

// List lists all StagedNetworkPolicies in the indexer.
func (s *stagedNetworkPolicyLister) List(selector labels.Selector) (ret []*projectcalico.StagedNetworkPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.StagedNetworkPolicy))
	})
	return ret, err
}

// StagedNetworkPolicies returns an object that can list and get StagedNetworkPolicies.
func (s *stagedNetworkPolicyLister) StagedNetworkPolicies(namespace string) StagedNetworkPolicyNamespaceLister {
	return stagedNetworkPolicyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// StagedNetworkPolicyNamespaceLister helps list and get StagedNetworkPolicies.
// All objects returned here must be treated as read-only.
type StagedNetworkPolicyNamespaceLister interface {
	// List lists all StagedNetworkPolicies in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*projectcalico.StagedNetworkPolicy, err error)
	// Get retrieves the StagedNetworkPolicy from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*projectcalico.StagedNetworkPolicy, error)
	StagedNetworkPolicyNamespaceListerExpansion
}

// stagedNetworkPolicyNamespaceLister implements the StagedNetworkPolicyNamespaceLister
// interface.
type stagedNetworkPolicyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all StagedNetworkPolicies in the indexer for a given namespace.
func (s stagedNetworkPolicyNamespaceLister) List(selector labels.Selector) (ret []*projectcalico.StagedNetworkPolicy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.StagedNetworkPolicy))
	})
	return ret, err
}

// Get retrieves the StagedNetworkPolicy from the indexer for a given namespace and name.
func (s stagedNetworkPolicyNamespaceLister) Get(name string) (*projectcalico.StagedNetworkPolicy, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("stagednetworkpolicy"), name)
	}
	return obj.(*projectcalico.StagedNetworkPolicy), nil
}
