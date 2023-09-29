// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package managedcluster

import (
	"context"

	"github.com/projectcalico/calico/apiserver/pkg/registry/projectcalico/server"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/apiserver/pkg/storage/calico"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
}

// EmptyObject returns an empty instance
func EmptyObject() runtime.Object {
	return &v3.ManagedCluster{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &v3.ManagedClusterList{}
}

// StatusREST implements the REST endpoint for changing the status of a deployment
type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &v3.ManagedCluster{}
}

func (r *StatusREST) Destroy() {
	r.store.Destroy()
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, opts server.Options) (*REST, *StatusREST, error) {
	strategy := NewStrategy(scheme)

	prefix := "/" + opts.ResourcePrefix()
	// We adapt the store's keyFunc so that we can use it with the StorageDecorator
	// without making any assumptions about where objects are stored in etcd
	keyFunc := func(obj runtime.Object) (string, error) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return "", err
		}
		if calico.MultiTenantEnabled {
			return genericregistry.NamespaceKeyFunc(
				genericapirequest.WithNamespace(
					genericapirequest.NewContext(),
					accessor.GetNamespace()),
				prefix, accessor.GetName())
		}
		return genericregistry.NoNamespaceKeyFunc(
			genericapirequest.NewContext(),
			prefix,
			accessor.GetName(),
		)
	}
	storageInterface, dFunc, err := opts.GetStorage(
		prefix,
		keyFunc,
		strategy,
		func() runtime.Object { return &v3.ManagedCluster{} },
		func() runtime.Object { return &v3.ManagedClusterList{} },
		GetAttrs,
		nil,
		nil,
	)
	if err != nil {
		return nil, nil, err
	}
	store := &genericregistry.Store{
		NewFunc:     func() runtime.Object { return &v3.ManagedCluster{} },
		NewListFunc: func() runtime.Object { return &v3.ManagedClusterList{} },
		KeyRootFunc: opts.KeyRootFunc(calico.MultiTenantEnabled),
		KeyFunc:     opts.KeyFunc(calico.MultiTenantEnabled),
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*v3.ManagedCluster).Name, nil
		},
		PredicateFunc:            MatchManagedCluster,
		DefaultQualifiedResource: v3.Resource("managedclusters"),

		CreateStrategy:          strategy,
		UpdateStrategy:          strategy,
		DeleteStrategy:          strategy,
		EnableGarbageCollection: true,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}

	statusStore := *store
	statusStore.UpdateStrategy = NewStatusStrategy(strategy)

	return &REST{store}, &StatusREST{&statusStore}, nil
}
