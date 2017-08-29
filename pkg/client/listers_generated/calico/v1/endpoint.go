/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was automatically generated by lister-gen

package v1

import (
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	v1 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// EndpointLister helps list Endpoints.
type EndpointLister interface {
	// List lists all Endpoints in the indexer.
	List(selector labels.Selector) (ret []*v1.Endpoint, err error)
	// Get retrieves the Endpoint from the index for a given name.
	Get(name string) (*v1.Endpoint, error)
	EndpointListerExpansion
}

// endpointLister implements the EndpointLister interface.
type endpointLister struct {
	indexer cache.Indexer
}

// NewEndpointLister returns a new EndpointLister.
func NewEndpointLister(indexer cache.Indexer) EndpointLister {
	return &endpointLister{indexer: indexer}
}

// List lists all Endpoints in the indexer.
func (s *endpointLister) List(selector labels.Selector) (ret []*v1.Endpoint, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Endpoint))
	})
	return ret, err
}

// Get retrieves the Endpoint from the index for a given name.
func (s *endpointLister) Get(name string) (*v1.Endpoint, error) {
	key := &v1.Endpoint{ObjectMeta: meta_v1.ObjectMeta{Name: name}}
	obj, exists, err := s.indexer.Get(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(calico.Resource("endpoint"), name)
	}
	return obj.(*v1.Endpoint), nil
}
