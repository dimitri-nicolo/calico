// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// GlobalAlertTemplateLister helps list GlobalAlertTemplates.
type GlobalAlertTemplateLister interface {
	// List lists all GlobalAlertTemplates in the indexer.
	List(selector labels.Selector) (ret []*v3.GlobalAlertTemplate, err error)
	// Get retrieves the GlobalAlertTemplate from the index for a given name.
	Get(name string) (*v3.GlobalAlertTemplate, error)
	GlobalAlertTemplateListerExpansion
}

// globalAlertTemplateLister implements the GlobalAlertTemplateLister interface.
type globalAlertTemplateLister struct {
	indexer cache.Indexer
}

// NewGlobalAlertTemplateLister returns a new GlobalAlertTemplateLister.
func NewGlobalAlertTemplateLister(indexer cache.Indexer) GlobalAlertTemplateLister {
	return &globalAlertTemplateLister{indexer: indexer}
}

// List lists all GlobalAlertTemplates in the indexer.
func (s *globalAlertTemplateLister) List(selector labels.Selector) (ret []*v3.GlobalAlertTemplate, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.GlobalAlertTemplate))
	})
	return ret, err
}

// Get retrieves the GlobalAlertTemplate from the index for a given name.
func (s *globalAlertTemplateLister) Get(name string) (*v3.GlobalAlertTemplate, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("globalalerttemplate"), name)
	}
	return obj.(*v3.GlobalAlertTemplate), nil
}
