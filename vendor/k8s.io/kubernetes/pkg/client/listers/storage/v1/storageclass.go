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
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	storage "k8s.io/kubernetes/pkg/apis/storage"
	v1 "k8s.io/kubernetes/pkg/apis/storage/v1"
)

// StorageClassLister helps list StorageClasses.
type StorageClassLister interface {
	// List lists all StorageClasses in the indexer.
	List(selector labels.Selector) (ret []*v1.StorageClass, err error)
	// Get retrieves the StorageClass from the index for a given name.
	Get(name string) (*v1.StorageClass, error)
	StorageClassListerExpansion
}

// storageClassLister implements the StorageClassLister interface.
type storageClassLister struct {
	indexer cache.Indexer
}

// NewStorageClassLister returns a new StorageClassLister.
func NewStorageClassLister(indexer cache.Indexer) StorageClassLister {
	return &storageClassLister{indexer: indexer}
}

// List lists all StorageClasses in the indexer.
func (s *storageClassLister) List(selector labels.Selector) (ret []*v1.StorageClass, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.StorageClass))
	})
	return ret, err
}

// Get retrieves the StorageClass from the index for a given name.
func (s *storageClassLister) Get(name string) (*v1.StorageClass, error) {
	key := &v1.StorageClass{ObjectMeta: meta_v1.ObjectMeta{Name: name}}
	obj, exists, err := s.indexer.Get(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(storage.Resource("storageclass"), name)
	}
	return obj.(*v1.StorageClass), nil
}
