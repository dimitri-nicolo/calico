// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ManagedClusterLister helps list ManagedClusters.
// All objects returned here must be treated as read-only.
type ManagedClusterLister interface {
	// List lists all ManagedClusters in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.ManagedCluster, err error)
	// Get retrieves the ManagedCluster from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.ManagedCluster, error)
	ManagedClusterListerExpansion
}

// managedClusterLister implements the ManagedClusterLister interface.
type managedClusterLister struct {
	indexer cache.Indexer
}

// NewManagedClusterLister returns a new ManagedClusterLister.
func NewManagedClusterLister(indexer cache.Indexer) ManagedClusterLister {
	return &managedClusterLister{indexer: indexer}
}

// List lists all ManagedClusters in the indexer.
func (s *managedClusterLister) List(selector labels.Selector) (ret []*v3.ManagedCluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.ManagedCluster))
	})
	return ret, err
}

// Get retrieves the ManagedCluster from the index for a given name.
func (s *managedClusterLister) Get(name string) (*v3.ManagedCluster, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("managedcluster"), name)
	}
	return obj.(*v3.ManagedCluster), nil
}
