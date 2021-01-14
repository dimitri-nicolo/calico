// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// GlobalNetworkSetLister helps list GlobalNetworkSets.
// All objects returned here must be treated as read-only.
type GlobalNetworkSetLister interface {
	// List lists all GlobalNetworkSets in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.GlobalNetworkSet, err error)
	// Get retrieves the GlobalNetworkSet from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.GlobalNetworkSet, error)
	GlobalNetworkSetListerExpansion
}

// globalNetworkSetLister implements the GlobalNetworkSetLister interface.
type globalNetworkSetLister struct {
	indexer cache.Indexer
}

// NewGlobalNetworkSetLister returns a new GlobalNetworkSetLister.
func NewGlobalNetworkSetLister(indexer cache.Indexer) GlobalNetworkSetLister {
	return &globalNetworkSetLister{indexer: indexer}
}

// List lists all GlobalNetworkSets in the indexer.
func (s *globalNetworkSetLister) List(selector labels.Selector) (ret []*v3.GlobalNetworkSet, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.GlobalNetworkSet))
	})
	return ret, err
}

// Get retrieves the GlobalNetworkSet from the index for a given name.
func (s *globalNetworkSetLister) Get(name string) (*v3.GlobalNetworkSet, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("globalnetworkset"), name)
	}
	return obj.(*v3.GlobalNetworkSet), nil
}
