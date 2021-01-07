// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ManagedClusterLister helps list ManagedClusters.
type ManagedClusterLister interface {
	// List lists all ManagedClusters in the indexer.
	List(selector labels.Selector) (ret []*projectcalico.ManagedCluster, err error)
	// Get retrieves the ManagedCluster from the index for a given name.
	Get(name string) (*projectcalico.ManagedCluster, error)
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
func (s *managedClusterLister) List(selector labels.Selector) (ret []*projectcalico.ManagedCluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.ManagedCluster))
	})
	return ret, err
}

// Get retrieves the ManagedCluster from the index for a given name.
func (s *managedClusterLister) Get(name string) (*projectcalico.ManagedCluster, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("managedcluster"), name)
	}
	return obj.(*projectcalico.ManagedCluster), nil
}
