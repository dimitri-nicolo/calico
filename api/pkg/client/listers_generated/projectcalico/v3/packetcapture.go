// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PacketCaptureLister helps list PacketCaptures.
// All objects returned here must be treated as read-only.
type PacketCaptureLister interface {
	// List lists all PacketCaptures in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.PacketCapture, err error)
	// PacketCaptures returns an object that can list and get PacketCaptures.
	PacketCaptures(namespace string) PacketCaptureNamespaceLister
	PacketCaptureListerExpansion
}

// packetCaptureLister implements the PacketCaptureLister interface.
type packetCaptureLister struct {
	indexer cache.Indexer
}

// NewPacketCaptureLister returns a new PacketCaptureLister.
func NewPacketCaptureLister(indexer cache.Indexer) PacketCaptureLister {
	return &packetCaptureLister{indexer: indexer}
}

// List lists all PacketCaptures in the indexer.
func (s *packetCaptureLister) List(selector labels.Selector) (ret []*v3.PacketCapture, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.PacketCapture))
	})
	return ret, err
}

// PacketCaptures returns an object that can list and get PacketCaptures.
func (s *packetCaptureLister) PacketCaptures(namespace string) PacketCaptureNamespaceLister {
	return packetCaptureNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PacketCaptureNamespaceLister helps list and get PacketCaptures.
// All objects returned here must be treated as read-only.
type PacketCaptureNamespaceLister interface {
	// List lists all PacketCaptures in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.PacketCapture, err error)
	// Get retrieves the PacketCapture from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.PacketCapture, error)
	PacketCaptureNamespaceListerExpansion
}

// packetCaptureNamespaceLister implements the PacketCaptureNamespaceLister
// interface.
type packetCaptureNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all PacketCaptures in the indexer for a given namespace.
func (s packetCaptureNamespaceLister) List(selector labels.Selector) (ret []*v3.PacketCapture, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.PacketCapture))
	})
	return ret, err
}

// Get retrieves the PacketCapture from the indexer for a given namespace and name.
func (s packetCaptureNamespaceLister) Get(name string) (*v3.PacketCapture, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("packetcapture"), name)
	}
	return obj.(*v3.PacketCapture), nil
}
