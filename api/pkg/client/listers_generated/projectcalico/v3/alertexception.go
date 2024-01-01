// Copyright (c) 2024 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// AlertExceptionLister helps list AlertExceptions.
// All objects returned here must be treated as read-only.
type AlertExceptionLister interface {
	// List lists all AlertExceptions in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.AlertException, err error)
	// Get retrieves the AlertException from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.AlertException, error)
	AlertExceptionListerExpansion
}

// alertExceptionLister implements the AlertExceptionLister interface.
type alertExceptionLister struct {
	indexer cache.Indexer
}

// NewAlertExceptionLister returns a new AlertExceptionLister.
func NewAlertExceptionLister(indexer cache.Indexer) AlertExceptionLister {
	return &alertExceptionLister{indexer: indexer}
}

// List lists all AlertExceptions in the indexer.
func (s *alertExceptionLister) List(selector labels.Selector) (ret []*v3.AlertException, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.AlertException))
	})
	return ret, err
}

// Get retrieves the AlertException from the index for a given name.
func (s *alertExceptionLister) Get(name string) (*v3.AlertException, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("alertexception"), name)
	}
	return obj.(*v3.AlertException), nil
}
