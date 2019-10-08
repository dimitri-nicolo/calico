// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// GlobalAlertLister helps list GlobalAlerts.
type GlobalAlertLister interface {
	// List lists all GlobalAlerts in the indexer.
	List(selector labels.Selector) (ret []*projectcalico.GlobalAlert, err error)
	// Get retrieves the GlobalAlert from the index for a given name.
	Get(name string) (*projectcalico.GlobalAlert, error)
	GlobalAlertListerExpansion
}

// globalAlertLister implements the GlobalAlertLister interface.
type globalAlertLister struct {
	indexer cache.Indexer
}

// NewGlobalAlertLister returns a new GlobalAlertLister.
func NewGlobalAlertLister(indexer cache.Indexer) GlobalAlertLister {
	return &globalAlertLister{indexer: indexer}
}

// List lists all GlobalAlerts in the indexer.
func (s *globalAlertLister) List(selector labels.Selector) (ret []*projectcalico.GlobalAlert, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.GlobalAlert))
	})
	return ret, err
}

// Get retrieves the GlobalAlert from the index for a given name.
func (s *globalAlertLister) Get(name string) (*projectcalico.GlobalAlert, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("globalalert"), name)
	}
	return obj.(*projectcalico.GlobalAlert), nil
}
