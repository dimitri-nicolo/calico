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

package internalversion

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	extensions "k8s.io/kubernetes/pkg/apis/extensions"
)

// ThirdPartyResourceLister helps list ThirdPartyResources.
type ThirdPartyResourceLister interface {
	// List lists all ThirdPartyResources in the indexer.
	List(selector labels.Selector) (ret []*extensions.ThirdPartyResource, err error)
	// Get retrieves the ThirdPartyResource from the index for a given name.
	Get(name string) (*extensions.ThirdPartyResource, error)
	ThirdPartyResourceListerExpansion
}

// thirdPartyResourceLister implements the ThirdPartyResourceLister interface.
type thirdPartyResourceLister struct {
	indexer cache.Indexer
}

// NewThirdPartyResourceLister returns a new ThirdPartyResourceLister.
func NewThirdPartyResourceLister(indexer cache.Indexer) ThirdPartyResourceLister {
	return &thirdPartyResourceLister{indexer: indexer}
}

// List lists all ThirdPartyResources in the indexer.
func (s *thirdPartyResourceLister) List(selector labels.Selector) (ret []*extensions.ThirdPartyResource, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*extensions.ThirdPartyResource))
	})
	return ret, err
}

// Get retrieves the ThirdPartyResource from the index for a given name.
func (s *thirdPartyResourceLister) Get(name string) (*extensions.ThirdPartyResource, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(extensions.Resource("thirdpartyresource"), name)
	}
	return obj.(*extensions.ThirdPartyResource), nil
}
