// Copyright (c) 2024 Tigera, Inc. All rights reserved.

// Code generated by lister-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// AuthenticationReviewLister helps list AuthenticationReviews.
// All objects returned here must be treated as read-only.
type AuthenticationReviewLister interface {
	// List lists all AuthenticationReviews in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v3.AuthenticationReview, err error)
	// Get retrieves the AuthenticationReview from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v3.AuthenticationReview, error)
	AuthenticationReviewListerExpansion
}

// authenticationReviewLister implements the AuthenticationReviewLister interface.
type authenticationReviewLister struct {
	indexer cache.Indexer
}

// NewAuthenticationReviewLister returns a new AuthenticationReviewLister.
func NewAuthenticationReviewLister(indexer cache.Indexer) AuthenticationReviewLister {
	return &authenticationReviewLister{indexer: indexer}
}

// List lists all AuthenticationReviews in the indexer.
func (s *authenticationReviewLister) List(selector labels.Selector) (ret []*v3.AuthenticationReview, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.AuthenticationReview))
	})
	return ret, err
}

// Get retrieves the AuthenticationReview from the index for a given name.
func (s *authenticationReviewLister) Get(name string) (*v3.AuthenticationReview, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v3.Resource("authenticationreview"), name)
	}
	return obj.(*v3.AuthenticationReview), nil
}
