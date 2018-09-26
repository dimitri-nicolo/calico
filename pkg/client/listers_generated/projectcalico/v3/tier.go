// Copyright (c) 2018 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TierLister helps list Tiers.
type TierLister interface {
	// List lists all Tiers in the indexer.
	List(selector labels.Selector) (ret []*v3.Tier, err error)
	// Get retrieves the Tier from the index for a given name.
	Get(name string) (*v3.Tier, error)
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
func (s *tierLister) List(selector labels.Selector) (ret []*v3.Tier, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.Tier))
	})
	return ret, err
}

// Get retrieves the Tier from the index for a given name.
func (s *tierLister) Get(name string) (*v3.Tier, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("tier"), name)
	}
	return obj.(*v3.Tier), nil
}
