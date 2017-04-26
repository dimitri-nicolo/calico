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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration"
	v1alpha1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1alpha1"
)

// APIServiceLister helps list APIServices.
type APIServiceLister interface {
	// List lists all APIServices in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.APIService, err error)
	// Get retrieves the APIService from the index for a given name.
	Get(name string) (*v1alpha1.APIService, error)
	APIServiceListerExpansion
}

// aPIServiceLister implements the APIServiceLister interface.
type aPIServiceLister struct {
	indexer cache.Indexer
}

// NewAPIServiceLister returns a new APIServiceLister.
func NewAPIServiceLister(indexer cache.Indexer) APIServiceLister {
	return &aPIServiceLister{indexer: indexer}
}

// List lists all APIServices in the indexer.
func (s *aPIServiceLister) List(selector labels.Selector) (ret []*v1alpha1.APIService, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.APIService))
	})
	return ret, err
}

// Get retrieves the APIService from the index for a given name.
func (s *aPIServiceLister) Get(name string) (*v1alpha1.APIService, error) {
	key := &v1alpha1.APIService{ObjectMeta: v1.ObjectMeta{Name: name}}
	obj, exists, err := s.indexer.Get(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(apiregistration.Resource("apiservice"), name)
	}
	return obj.(*v1alpha1.APIService), nil
}
