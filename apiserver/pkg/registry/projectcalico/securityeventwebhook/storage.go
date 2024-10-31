// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package securityeventwebhook

import (
	calico "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic/registry"

	"github.com/projectcalico/calico/apiserver/pkg/registry/projectcalico/server"
)

type REST struct {
	*registry.Store
	shortNames []string
}

func (r *REST) ShortNames() []string {
	return r.shortNames
}

func (r *REST) Categories() []string {
	return []string{""}
}

// EmptyObject returns an empty instance
func EmptyObject() runtime.Object {
	return &calico.SecurityEventWebhook{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &calico.SecurityEventWebhookList{}
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, opts server.Options) (*REST, error) {
	strategy := NewStrategy(scheme)

	prefix := "/" + opts.ResourcePrefix()
	// We adapt the store's keyFunc so that we can use it with the StorageDecorator
	// without making any assumptions about where objects are stored in etcd
	keyFunc := func(obj runtime.Object) (string, error) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return "", err
		}
		return registry.NoNamespaceKeyFunc(
			request.NewContext(),
			prefix,
			accessor.GetName(),
		)
	}
	storageInterface, dFunc, err := opts.GetStorage(
		prefix,
		keyFunc,
		strategy,
		func() runtime.Object { return &calico.SecurityEventWebhook{} },
		func() runtime.Object { return &calico.SecurityEventWebhookList{} },
		GetAttrs,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}
	store := &registry.Store{
		NewFunc:     func() runtime.Object { return &calico.SecurityEventWebhook{} },
		NewListFunc: func() runtime.Object { return &calico.SecurityEventWebhookList{} },
		KeyRootFunc: opts.KeyRootFunc(false),
		KeyFunc:     opts.KeyFunc(false),
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.SecurityEventWebhook).Name, nil
		},
		PredicateFunc:            MatchSecurityEventWebhookConfiguration,
		DefaultQualifiedResource: calico.Resource("securityeventwebhooks"),

		CreateStrategy:          strategy,
		UpdateStrategy:          strategy,
		DeleteStrategy:          strategy,
		EnableGarbageCollection: true,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}

	return &REST{store, opts.ShortNames}, nil
}
