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
	v1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// SelfSubjectAccessReviewLister helps list SelfSubjectAccessReviews.
type SelfSubjectAccessReviewLister interface {
	// List lists all SelfSubjectAccessReviews in the indexer.
	List(selector labels.Selector) (ret []*v1.SelfSubjectAccessReview, err error)
	// Get retrieves the SelfSubjectAccessReview from the index for a given name.
	Get(name string) (*v1.SelfSubjectAccessReview, error)
	SelfSubjectAccessReviewListerExpansion
}

// selfSubjectAccessReviewLister implements the SelfSubjectAccessReviewLister interface.
type selfSubjectAccessReviewLister struct {
	indexer cache.Indexer
}

// NewSelfSubjectAccessReviewLister returns a new SelfSubjectAccessReviewLister.
func NewSelfSubjectAccessReviewLister(indexer cache.Indexer) SelfSubjectAccessReviewLister {
	return &selfSubjectAccessReviewLister{indexer: indexer}
}

// List lists all SelfSubjectAccessReviews in the indexer.
func (s *selfSubjectAccessReviewLister) List(selector labels.Selector) (ret []*v1.SelfSubjectAccessReview, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.SelfSubjectAccessReview))
	})
	return ret, err
}

// Get retrieves the SelfSubjectAccessReview from the index for a given name.
func (s *selfSubjectAccessReviewLister) Get(name string) (*v1.SelfSubjectAccessReview, error) {
	key := &v1.SelfSubjectAccessReview{ObjectMeta: meta_v1.ObjectMeta{Name: name}}
	obj, exists, err := s.indexer.Get(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("selfsubjectaccessreview"), name)
	}
	return obj.(*v1.SelfSubjectAccessReview), nil
}
