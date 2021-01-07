// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ClusterInformationLister helps list ClusterInformations.
type ClusterInformationLister interface {
	// List lists all ClusterInformations in the indexer.
	List(selector labels.Selector) (ret []*projectcalico.ClusterInformation, err error)
	// Get retrieves the ClusterInformation from the index for a given name.
	Get(name string) (*projectcalico.ClusterInformation, error)
	ClusterInformationListerExpansion
}

// clusterInformationLister implements the ClusterInformationLister interface.
type clusterInformationLister struct {
	indexer cache.Indexer
}

// NewClusterInformationLister returns a new ClusterInformationLister.
func NewClusterInformationLister(indexer cache.Indexer) ClusterInformationLister {
	return &clusterInformationLister{indexer: indexer}
}

// List lists all ClusterInformations in the indexer.
func (s *clusterInformationLister) List(selector labels.Selector) (ret []*projectcalico.ClusterInformation, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.ClusterInformation))
	})
	return ret, err
}

// Get retrieves the ClusterInformation from the index for a given name.
func (s *clusterInformationLister) Get(name string) (*projectcalico.ClusterInformation, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("clusterinformation"), name)
	}
	return obj.(*projectcalico.ClusterInformation), nil
}
