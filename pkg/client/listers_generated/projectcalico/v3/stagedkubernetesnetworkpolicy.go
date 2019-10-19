// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// StagedKubernetesNetworkPolicyLister helps list StagedKubernetesNetworkPolicies.
type StagedKubernetesNetworkPolicyLister interface {
	// List lists all StagedKubernetesNetworkPolicies in the indexer.
	List(selector labels.Selector) (ret []*v3.StagedKubernetesNetworkPolicy, err error)
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
func (s *stagedKubernetesNetworkPolicyLister) List(selector labels.Selector) (ret []*v3.StagedKubernetesNetworkPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.StagedKubernetesNetworkPolicy))
	})
	return ret, err
}

// StagedKubernetesNetworkPolicies returns an object that can list and get StagedKubernetesNetworkPolicies.
func (s *stagedKubernetesNetworkPolicyLister) StagedKubernetesNetworkPolicies(namespace string) StagedKubernetesNetworkPolicyNamespaceLister {
	return stagedKubernetesNetworkPolicyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// StagedKubernetesNetworkPolicyNamespaceLister helps list and get StagedKubernetesNetworkPolicies.
type StagedKubernetesNetworkPolicyNamespaceLister interface {
	// List lists all StagedKubernetesNetworkPolicies in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v3.StagedKubernetesNetworkPolicy, err error)
	// Get retrieves the StagedKubernetesNetworkPolicy from the indexer for a given namespace and name.
	Get(name string) (*v3.StagedKubernetesNetworkPolicy, error)
	StagedKubernetesNetworkPolicyNamespaceListerExpansion
}

// stagedKubernetesNetworkPolicyNamespaceLister implements the StagedKubernetesNetworkPolicyNamespaceLister
// interface.
type stagedKubernetesNetworkPolicyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all StagedKubernetesNetworkPolicies in the indexer for a given namespace.
func (s stagedKubernetesNetworkPolicyNamespaceLister) List(selector labels.Selector) (ret []*v3.StagedKubernetesNetworkPolicy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.StagedKubernetesNetworkPolicy))
	})
	return ret, err
}

// Get retrieves the StagedKubernetesNetworkPolicy from the indexer for a given namespace and name.
func (s stagedKubernetesNetworkPolicyNamespaceLister) Get(name string) (*v3.StagedKubernetesNetworkPolicy, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("stagedkubernetesnetworkpolicy"), name)
	}
	return obj.(*v3.StagedKubernetesNetworkPolicy), nil
}
