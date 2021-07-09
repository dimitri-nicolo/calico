// Copyright (c) 2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	projectcalico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// DeepPacketInspectionLister helps list DeepPacketInspections.
// All objects returned here must be treated as read-only.
type DeepPacketInspectionLister interface {
	// List lists all DeepPacketInspections in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*projectcalico.DeepPacketInspection, err error)
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
func (s *deepPacketInspectionLister) List(selector labels.Selector) (ret []*projectcalico.DeepPacketInspection, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.DeepPacketInspection))
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
	List(selector labels.Selector) (ret []*projectcalico.DeepPacketInspection, err error)
	// Get retrieves the DeepPacketInspection from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*projectcalico.DeepPacketInspection, error)
	DeepPacketInspectionNamespaceListerExpansion
}

// deepPacketInspectionNamespaceLister implements the DeepPacketInspectionNamespaceLister
// interface.
type deepPacketInspectionNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DeepPacketInspections in the indexer for a given namespace.
func (s deepPacketInspectionNamespaceLister) List(selector labels.Selector) (ret []*projectcalico.DeepPacketInspection, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.DeepPacketInspection))
	})
	return ret, err
}

// Get retrieves the DeepPacketInspection from the indexer for a given namespace and name.
func (s deepPacketInspectionNamespaceLister) Get(name string) (*projectcalico.DeepPacketInspection, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("deeppacketinspection"), name)
	}
	return obj.(*projectcalico.DeepPacketInspection), nil
}
