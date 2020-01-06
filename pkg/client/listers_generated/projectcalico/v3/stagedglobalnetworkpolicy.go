// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// StagedGlobalNetworkPolicyLister helps list StagedGlobalNetworkPolicies.
type StagedGlobalNetworkPolicyLister interface {
	// List lists all StagedGlobalNetworkPolicies in the indexer.
	List(selector labels.Selector) (ret []*v3.StagedGlobalNetworkPolicy, err error)
	// Get retrieves the StagedGlobalNetworkPolicy from the index for a given name.
	Get(name string) (*v3.StagedGlobalNetworkPolicy, error)
	StagedGlobalNetworkPolicyListerExpansion
}

// stagedGlobalNetworkPolicyLister implements the StagedGlobalNetworkPolicyLister interface.
type stagedGlobalNetworkPolicyLister struct {
	indexer cache.Indexer
}

// NewStagedGlobalNetworkPolicyLister returns a new StagedGlobalNetworkPolicyLister.
func NewStagedGlobalNetworkPolicyLister(indexer cache.Indexer) StagedGlobalNetworkPolicyLister {
	return &stagedGlobalNetworkPolicyLister{indexer: indexer}
}

// List lists all StagedGlobalNetworkPolicies in the indexer.
func (s *stagedGlobalNetworkPolicyLister) List(selector labels.Selector) (ret []*v3.StagedGlobalNetworkPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.StagedGlobalNetworkPolicy))
	})
	return ret, err
}

// Get retrieves the StagedGlobalNetworkPolicy from the index for a given name.
func (s *stagedGlobalNetworkPolicyLister) Get(name string) (*v3.StagedGlobalNetworkPolicy, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("stagedglobalnetworkpolicy"), name)
	}
	return obj.(*v3.StagedGlobalNetworkPolicy), nil
}
