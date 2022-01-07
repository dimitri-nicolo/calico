// Copyright (c) 2022 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// GlobalReportLister helps list GlobalReports.
// All objects returned here must be treated as read-only.
type GlobalReportLister interface {
	// List lists all GlobalReports in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.GlobalReport, err error)
	// Get retrieves the GlobalReport from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.GlobalReport, error)
	GlobalReportListerExpansion
}

// globalReportLister implements the GlobalReportLister interface.
type globalReportLister struct {
	indexer cache.Indexer
}

// NewGlobalReportLister returns a new GlobalReportLister.
func NewGlobalReportLister(indexer cache.Indexer) GlobalReportLister {
	return &globalReportLister{indexer: indexer}
}

// List lists all GlobalReports in the indexer.
func (s *globalReportLister) List(selector labels.Selector) (ret []*v3.GlobalReport, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.GlobalReport))
	})
	return ret, err
}

// Get retrieves the GlobalReport from the index for a given name.
func (s *globalReportLister) Get(name string) (*v3.GlobalReport, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("globalreport"), name)
	}
	return obj.(*v3.GlobalReport), nil
}
