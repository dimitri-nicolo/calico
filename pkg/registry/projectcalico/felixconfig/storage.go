// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package felixconfig

import (
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/server"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
}

// EmptyObject returns an empty instance
func EmptyObject() runtime.Object {
	return &calico.FelixConfiguration{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &calico.FelixConfigurationList{}
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, opts server.Options) *REST {
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
			genericapirequest.NewContext(),
			prefix,
			accessor.GetName(),
		)
	}
	storageInterface, dFunc := opts.GetStorage(
		&calico.FelixConfiguration{},
		prefix,
		keyFunc,
		strategy,
		func() runtime.Object { return &calico.FelixConfigurationList{} },
		GetAttrs,
		storage.NoTriggerPublisher,
	)
	store := &genericregistry.Store{
		NewFunc:     func() runtime.Object { return &calico.FelixConfiguration{} },
		NewListFunc: func() runtime.Object { return &calico.FelixConfigurationList{} },
		KeyRootFunc: opts.KeyRootFunc(false),
		KeyFunc:     opts.KeyFunc(false),
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.FelixConfiguration).Name, nil
		},
		PredicateFunc:            MatchFelixConfiguration,
		DefaultQualifiedResource: calico.Resource("felixconfigurations"),

		CreateStrategy:          strategy,
		UpdateStrategy:          strategy,
		DeleteStrategy:          strategy,
		EnableGarbageCollection: true,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}

	return &REST{store}
}
