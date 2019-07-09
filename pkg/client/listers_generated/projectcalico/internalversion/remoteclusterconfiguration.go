// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// RemoteClusterConfigurationLister helps list RemoteClusterConfigurations.
type RemoteClusterConfigurationLister interface {
	// List lists all RemoteClusterConfigurations in the indexer.
	List(selector labels.Selector) (ret []*projectcalico.RemoteClusterConfiguration, err error)
	// Get retrieves the RemoteClusterConfiguration from the index for a given name.
	Get(name string) (*projectcalico.RemoteClusterConfiguration, error)
	RemoteClusterConfigurationListerExpansion
}

// remoteClusterConfigurationLister implements the RemoteClusterConfigurationLister interface.
type remoteClusterConfigurationLister struct {
	indexer cache.Indexer
}

// NewRemoteClusterConfigurationLister returns a new RemoteClusterConfigurationLister.
func NewRemoteClusterConfigurationLister(indexer cache.Indexer) RemoteClusterConfigurationLister {
	return &remoteClusterConfigurationLister{indexer: indexer}
}

// List lists all RemoteClusterConfigurations in the indexer.
func (s *remoteClusterConfigurationLister) List(selector labels.Selector) (ret []*projectcalico.RemoteClusterConfiguration, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.RemoteClusterConfiguration))
	})
	return ret, err
}

// Get retrieves the RemoteClusterConfiguration from the index for a given name.
func (s *remoteClusterConfigurationLister) Get(name string) (*projectcalico.RemoteClusterConfiguration, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("remoteclusterconfiguration"), name)
	}
	return obj.(*projectcalico.RemoteClusterConfiguration), nil
}
