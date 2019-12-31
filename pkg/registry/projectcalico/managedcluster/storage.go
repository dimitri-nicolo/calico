// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package managedcluster

import (
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/server"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
}

// EmptyObject returns an empty instance
func EmptyObject() runtime.Object {
	return &calico.ManagedCluster{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &calico.ManagedClusterList{}
}

// StatusREST implements the REST endpoint for changing the status of a deployment
type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &calico.ManagedCluster{}
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *StatusREST) Get(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx genericapirequest.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation)
}


// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, opts server.Options) (*REST, *StatusREST) {
	strategy := NewStrategy(scheme)

	prefix := "/" + opts.ResourcePrefix()
	// We adapt the store's keyFunc so that we can use it with the StorageDecorator
	// without making any assumptions about where objects are stored in etcd
	keyFunc := func(obj runtime.Object) (string, error) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return "", err
		}
		return genericregistry.NoNamespaceKeyFunc(
			genericapirequest.NewContext(),
			prefix,
			accessor.GetName(),
		)
	}
	storageInterface, dFunc := opts.GetStorage(
		&calico.ManagedCluster{},
		prefix,
		keyFunc,
		strategy,
		func() runtime.Object { return &calico.ManagedClusterList{} },
		GetAttrs,
		storage.NoTriggerPublisher,
	)
	store := &genericregistry.Store{
		NewFunc:     func() runtime.Object { return &calico.ManagedCluster{} },
		NewListFunc: func() runtime.Object { return &calico.ManagedClusterList{} },
		KeyRootFunc: opts.KeyRootFunc(false),
		KeyFunc:     opts.KeyFunc(false),
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.ManagedCluster).Name, nil
		},
		PredicateFunc:            MatchManagedCluster,
		DefaultQualifiedResource: calico.Resource("managedclusters"),

		CreateStrategy:          strategy,
		UpdateStrategy:          strategy,
		DeleteStrategy:          strategy,
		EnableGarbageCollection: true,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}

	statusStore := *store
	statusStore.UpdateStrategy = NewStatusStrategy(strategy)

	return &REST{store}, &StatusREST{&statusStore}
}
