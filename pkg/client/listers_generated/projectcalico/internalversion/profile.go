// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ProfileLister helps list Profiles.
type ProfileLister interface {
	// List lists all Profiles in the indexer.
	List(selector labels.Selector) (ret []*projectcalico.Profile, err error)
	// Get retrieves the Profile from the index for a given name.
	Get(name string) (*projectcalico.Profile, error)
	ProfileListerExpansion
}

// profileLister implements the ProfileLister interface.
type profileLister struct {
	indexer cache.Indexer
}

// NewProfileLister returns a new ProfileLister.
func NewProfileLister(indexer cache.Indexer) ProfileLister {
	return &profileLister{indexer: indexer}
}

// List lists all Profiles in the indexer.
func (s *profileLister) List(selector labels.Selector) (ret []*projectcalico.Profile, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*projectcalico.Profile))
	})
	return ret, err
}

// Get retrieves the Profile from the index for a given name.
func (s *profileLister) Get(name string) (*projectcalico.Profile, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(projectcalico.Resource("profile"), name)
	}
	return obj.(*projectcalico.Profile), nil
}
