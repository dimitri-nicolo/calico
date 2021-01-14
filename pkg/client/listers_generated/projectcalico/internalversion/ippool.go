// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// IPPoolLister helps list IPPools.
// All objects returned here must be treated as read-only.
type IPPoolLister interface {
	// List lists all IPPools in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*projectcalico.IPPool, err error)
	// Get retrieves the IPPool from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*projectcalico.IPPool, error)
	IPPoolListerExpansion
}

// iPPoolLister implements the IPPoolLister interface.
type iPPoolLister struct {
	indexer cache.Indexer
}

// NewIPPoolLister returns a new IPPoolLister.
func NewIPPoolLister(indexer cache.Indexer) IPPoolLister {
	return &iPPoolLister{indexer: indexer}
}

// List lists all IPPools in the indexer.
func (s *iPPoolLister) List(selector labels.Selector) (ret []*projectcalico.IPPool, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.IPPool))
	})
	return ret, err
}

// Get retrieves the IPPool from the index for a given name.
func (s *iPPoolLister) Get(name string) (*projectcalico.IPPool, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("ippool"), name)
	}
	return obj.(*projectcalico.IPPool), nil
}
