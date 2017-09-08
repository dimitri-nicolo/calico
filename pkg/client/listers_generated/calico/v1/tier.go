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
	v1 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TierLister helps list Tiers.
type TierLister interface {
	// List lists all Tiers in the indexer.
	List(selector labels.Selector) (ret []*v1.Tier, err error)
	// Get retrieves the Tier from the index for a given name.
	Get(name string) (*v1.Tier, error)
	TierListerExpansion
}

// tierLister implements the TierLister interface.
type tierLister struct {
	indexer cache.Indexer
}

// NewTierLister returns a new TierLister.
func NewTierLister(indexer cache.Indexer) TierLister {
	return &tierLister{indexer: indexer}
}

// List lists all Tiers in the indexer.
func (s *tierLister) List(selector labels.Selector) (ret []*v1.Tier, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Tier))
	})
	return ret, err
}

// Get retrieves the Tier from the index for a given name.
func (s *tierLister) Get(name string) (*v1.Tier, error) {
	key := &v1.Tier{ObjectMeta: meta_v1.ObjectMeta{Name: name}}
	obj, exists, err := s.indexer.Get(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("tier"), name)
	}
	return obj.(*v1.Tier), nil
}
