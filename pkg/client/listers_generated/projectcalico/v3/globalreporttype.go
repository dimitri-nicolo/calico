// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// GlobalReportTypeLister helps list GlobalReportTypes.
type GlobalReportTypeLister interface {
	// List lists all GlobalReportTypes in the indexer.
	List(selector labels.Selector) (ret []*v3.GlobalReportType, err error)
	// Get retrieves the GlobalReportType from the index for a given name.
	Get(name string) (*v3.GlobalReportType, error)
	GlobalReportTypeListerExpansion
}

// globalReportTypeLister implements the GlobalReportTypeLister interface.
type globalReportTypeLister struct {
	indexer cache.Indexer
}

// NewGlobalReportTypeLister returns a new GlobalReportTypeLister.
func NewGlobalReportTypeLister(indexer cache.Indexer) GlobalReportTypeLister {
	return &globalReportTypeLister{indexer: indexer}
}

// List lists all GlobalReportTypes in the indexer.
func (s *globalReportTypeLister) List(selector labels.Selector) (ret []*v3.GlobalReportType, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.GlobalReportType))
	})
	return ret, err
}

// Get retrieves the GlobalReportType from the index for a given name.
func (s *globalReportTypeLister) Get(name string) (*v3.GlobalReportType, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("globalreporttype"), name)
	}
	return obj.(*v3.GlobalReportType), nil
}
