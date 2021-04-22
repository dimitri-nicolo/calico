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

// StagedKubernetesNetworkPolicyLister helps list StagedKubernetesNetworkPolicies.
// All objects returned here must be treated as read-only.
type StagedKubernetesNetworkPolicyLister interface {
	// List lists all StagedKubernetesNetworkPolicies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*projectcalico.StagedKubernetesNetworkPolicy, err error)
	// StagedKubernetesNetworkPolicies returns an object that can list and get StagedKubernetesNetworkPolicies.
	StagedKubernetesNetworkPolicies(namespace string) StagedKubernetesNetworkPolicyNamespaceLister
	StagedKubernetesNetworkPolicyListerExpansion
}

// stagedKubernetesNetworkPolicyLister implements the StagedKubernetesNetworkPolicyLister interface.
type stagedKubernetesNetworkPolicyLister struct {
	indexer cache.Indexer
}

// NewStagedKubernetesNetworkPolicyLister returns a new StagedKubernetesNetworkPolicyLister.
func NewStagedKubernetesNetworkPolicyLister(indexer cache.Indexer) StagedKubernetesNetworkPolicyLister {
	return &stagedKubernetesNetworkPolicyLister{indexer: indexer}
}

// List lists all StagedKubernetesNetworkPolicies in the indexer.
func (s *stagedKubernetesNetworkPolicyLister) List(selector labels.Selector) (ret []*projectcalico.StagedKubernetesNetworkPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.StagedKubernetesNetworkPolicy))
	})
	return ret, err
}

// StagedKubernetesNetworkPolicies returns an object that can list and get StagedKubernetesNetworkPolicies.
func (s *stagedKubernetesNetworkPolicyLister) StagedKubernetesNetworkPolicies(namespace string) StagedKubernetesNetworkPolicyNamespaceLister {
	return stagedKubernetesNetworkPolicyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// StagedKubernetesNetworkPolicyNamespaceLister helps list and get StagedKubernetesNetworkPolicies.
// All objects returned here must be treated as read-only.
type StagedKubernetesNetworkPolicyNamespaceLister interface {
	// List lists all StagedKubernetesNetworkPolicies in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*projectcalico.StagedKubernetesNetworkPolicy, err error)
	// Get retrieves the StagedKubernetesNetworkPolicy from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*projectcalico.StagedKubernetesNetworkPolicy, error)
	StagedKubernetesNetworkPolicyNamespaceListerExpansion
}

// stagedKubernetesNetworkPolicyNamespaceLister implements the StagedKubernetesNetworkPolicyNamespaceLister
// interface.
type stagedKubernetesNetworkPolicyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all StagedKubernetesNetworkPolicies in the indexer for a given namespace.
func (s stagedKubernetesNetworkPolicyNamespaceLister) List(selector labels.Selector) (ret []*projectcalico.StagedKubernetesNetworkPolicy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.StagedKubernetesNetworkPolicy))
	})
	return ret, err
}

// Get retrieves the StagedKubernetesNetworkPolicy from the indexer for a given namespace and name.
func (s stagedKubernetesNetworkPolicyNamespaceLister) Get(name string) (*projectcalico.StagedKubernetesNetworkPolicy, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("stagedkubernetesnetworkpolicy"), name)
	}
	return obj.(*projectcalico.StagedKubernetesNetworkPolicy), nil
}
