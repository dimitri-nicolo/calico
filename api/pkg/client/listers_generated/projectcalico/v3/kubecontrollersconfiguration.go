// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// KubeControllersConfigurationLister helps list KubeControllersConfigurations.
// All objects returned here must be treated as read-only.
type KubeControllersConfigurationLister interface {
	// List lists all KubeControllersConfigurations in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.KubeControllersConfiguration, err error)
	// Get retrieves the KubeControllersConfiguration from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.KubeControllersConfiguration, error)
	KubeControllersConfigurationListerExpansion
}

// kubeControllersConfigurationLister implements the KubeControllersConfigurationLister interface.
type kubeControllersConfigurationLister struct {
	indexer cache.Indexer
}

// NewKubeControllersConfigurationLister returns a new KubeControllersConfigurationLister.
func NewKubeControllersConfigurationLister(indexer cache.Indexer) KubeControllersConfigurationLister {
	return &kubeControllersConfigurationLister{indexer: indexer}
}

// List lists all KubeControllersConfigurations in the indexer.
func (s *kubeControllersConfigurationLister) List(selector labels.Selector) (ret []*v3.KubeControllersConfiguration, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.KubeControllersConfiguration))
	})
	return ret, err
}

// Get retrieves the KubeControllersConfiguration from the index for a given name.
func (s *kubeControllersConfigurationLister) Get(name string) (*v3.KubeControllersConfiguration, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("kubecontrollersconfiguration"), name)
	}
	return obj.(*v3.KubeControllersConfiguration), nil
}
