// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package authenticationreview

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type REST struct{}

// EmptyObject returns an empty instance
func (r *REST) New() runtime.Object {
	return &v3.AuthenticationReview{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &v3.AuthenticationReviewList{}
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST() *REST {
	return &REST{}
}

// Necessary to satisfy generated informers, but not intended for real use.
func (r *REST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	return NewList(), nil
}

// Necessary to satisfy generated informers, but not intended for real use.
func (r *REST) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	return watch.NewEmptyWatch(), nil
}

// Takes the userinfo that the authn delegate has put into the context and returns it.
func (r *REST) Create(ctx context.Context, obj runtime.Object, val rest.ValidateObjectFunc, createOpt *metav1.CreateOptions) (runtime.Object, error) {
	ar := &v3.AuthenticationReview{
		Status: v3.AuthenticationReviewStatus{},
	}

	user, ok := request.UserFrom(ctx)
	if ok {
		ar.Status.Name = user.GetName()
		ar.Status.UID = user.GetUID()
		ar.Status.Extra = user.GetExtra()
		ar.Status.Groups = user.GetGroups()
	}
	return ar, nil
}

func (r *REST) NamespaceScoped() bool {
	return false
}
