// Copyright (c) 2023 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DeepPacketInspectionLister helps list DeepPacketInspections.
// All objects returned here must be treated as read-only.
type DeepPacketInspectionLister interface {
	// List lists all DeepPacketInspections in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.DeepPacketInspection, err error)
	// DeepPacketInspections returns an object that can list and get DeepPacketInspections.
	DeepPacketInspections(namespace string) DeepPacketInspectionNamespaceLister
	DeepPacketInspectionListerExpansion
}

// deepPacketInspectionLister implements the DeepPacketInspectionLister interface.
type deepPacketInspectionLister struct {
	indexer cache.Indexer
}

// NewDeepPacketInspectionLister returns a new DeepPacketInspectionLister.
func NewDeepPacketInspectionLister(indexer cache.Indexer) DeepPacketInspectionLister {
	return &deepPacketInspectionLister{indexer: indexer}
}

// List lists all DeepPacketInspections in the indexer.
func (s *deepPacketInspectionLister) List(selector labels.Selector) (ret []*v3.DeepPacketInspection, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.DeepPacketInspection))
	})
	return ret, err
}

// DeepPacketInspections returns an object that can list and get DeepPacketInspections.
func (s *deepPacketInspectionLister) DeepPacketInspections(namespace string) DeepPacketInspectionNamespaceLister {
	return deepPacketInspectionNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DeepPacketInspectionNamespaceLister helps list and get DeepPacketInspections.
// All objects returned here must be treated as read-only.
type DeepPacketInspectionNamespaceLister interface {
	// List lists all DeepPacketInspections in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.DeepPacketInspection, err error)
	// Get retrieves the DeepPacketInspection from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.DeepPacketInspection, error)
	DeepPacketInspectionNamespaceListerExpansion
}

// deepPacketInspectionNamespaceLister implements the DeepPacketInspectionNamespaceLister
// interface.
type deepPacketInspectionNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DeepPacketInspections in the indexer for a given namespace.
func (s deepPacketInspectionNamespaceLister) List(selector labels.Selector) (ret []*v3.DeepPacketInspection, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.DeepPacketInspection))
	})
	return ret, err
}

// Get retrieves the DeepPacketInspection from the indexer for a given namespace and name.
func (s deepPacketInspectionNamespaceLister) Get(name string) (*v3.DeepPacketInspection, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("deeppacketinspection"), name)
	}
	return obj.(*v3.DeepPacketInspection), nil
}
