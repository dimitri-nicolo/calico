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

package v3

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
)

// GlobalReportTypeLister helps list GlobalReportTypes.
// All objects returned here must be treated as read-only.
type GlobalReportTypeLister interface {
	// List lists all GlobalReportTypes in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.GlobalReportType, err error)
	// Get retrieves the GlobalReportType from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.GlobalReportType, error)
	GlobalReportTypeListerExpansion
}

// globalReportTypeLister implements the GlobalReportTypeLister interface.
type globalReportTypeLister struct {
	indexer cache.Indexer
}

// NewGlobalReportTypeLister returns a new GlobalReportTypeLister.
func NewGlobalReportTypeLister(indexer cache.Indexer) GlobalReportTypeLister {
	return &globalReportTypeLister{indexer: indexer}
}

// List lists all GlobalReportTypes in the indexer.
func (s *globalReportTypeLister) List(selector labels.Selector) (ret []*v3.GlobalReportType, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.GlobalReportType))
	})
	return ret, err
}

// Get retrieves the GlobalReportType from the index for a given name.
func (s *globalReportTypeLister) Get(name string) (*v3.GlobalReportType, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("globalreporttype"), name)
	}
	return obj.(*v3.GlobalReportType), nil
}
