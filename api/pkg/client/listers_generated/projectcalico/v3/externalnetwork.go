// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ExternalNetworkLister helps list ExternalNetworks.
// All objects returned here must be treated as read-only.
type ExternalNetworkLister interface {
	// List lists all ExternalNetworks in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.ExternalNetwork, err error)
	// Get retrieves the ExternalNetwork from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.ExternalNetwork, error)
	ExternalNetworkListerExpansion
}

// externalNetworkLister implements the ExternalNetworkLister interface.
type externalNetworkLister struct {
	indexer cache.Indexer
}

// NewExternalNetworkLister returns a new ExternalNetworkLister.
func NewExternalNetworkLister(indexer cache.Indexer) ExternalNetworkLister {
	return &externalNetworkLister{indexer: indexer}
}

// List lists all ExternalNetworks in the indexer.
func (s *externalNetworkLister) List(selector labels.Selector) (ret []*v3.ExternalNetwork, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.ExternalNetwork))
	})
	return ret, err
}

// Get retrieves the ExternalNetwork from the index for a given name.
func (s *externalNetworkLister) Get(name string) (*v3.ExternalNetwork, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("externalnetwork"), name)
	}
	return obj.(*v3.ExternalNetwork), nil
}
