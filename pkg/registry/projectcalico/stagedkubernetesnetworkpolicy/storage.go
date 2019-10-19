/*
Copyright 2019 The Kubernetes Authors.

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

package stagedkubernetesnetworkpolicy

import (
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/authorizer"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/server"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/util"

	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
	authorizer authorizer.TierAuthorizer
}

// EmptyObject returns an empty instance
func EmptyObject() runtime.Object {
	return &calico.StagedKubernetesNetworkPolicy{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &calico.StagedKubernetesNetworkPolicyList{
		//TypeMeta: metav1.TypeMeta{},
		//Items:    []calico.StagedKubernetesNetworkPolicy{},
	}
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
		return registry.NamespaceKeyFunc(genericapirequest.WithNamespace(genericapirequest.NewContext(), accessor.GetNamespace()), prefix, accessor.GetName())
	}
	storageInterface, dFunc := opts.GetStorage(
		&calico.StagedKubernetesNetworkPolicy{},
		prefix,
		keyFunc,
		strategy,
		func() runtime.Object { return &calico.StagedKubernetesNetworkPolicyList{} },
		GetAttrs,
		storage.NoTriggerPublisher,
	)
	store := &genericregistry.Store{
		NewFunc:     func() runtime.Object { return &calico.StagedKubernetesNetworkPolicy{} },
		NewListFunc: func() runtime.Object { return &calico.StagedKubernetesNetworkPolicyList{} },
		KeyRootFunc: opts.KeyRootFunc(true),
		KeyFunc:     opts.KeyFunc(true),
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.StagedKubernetesNetworkPolicy).Name, nil
		},
		PredicateFunc:            MatchPolicy,
		DefaultQualifiedResource: calico.Resource("stagedkubernetesnetworkpolicies"),

		CreateStrategy:          strategy,
		UpdateStrategy:          strategy,
		DeleteStrategy:          strategy,
		EnableGarbageCollection: true,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}

	return &REST{store, authorizer.NewTierAuthorizer(opts.Authorizer)}
}

func (r *REST) List(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	tierName, err := util.GetTierNameFromSelector(options)
	if err != nil {
		return nil, err
	}
	err = r.authorizer.AuthorizeTierOperation(ctx, "", tierName)
	if err != nil {
		return nil, err
	}

	return r.Store.List(ctx, options)
}

func (r *REST) Create(ctx genericapirequest.Context, obj runtime.Object, val rest.ValidateObjectFunc, includeUninitialized bool) (runtime.Object, error) {
	policy := obj.(*calico.StagedKubernetesNetworkPolicy)
	// Is Tier prepended. If not prepend default?
	tierName, _ := util.GetTierPolicy(policy.Name)
	err := r.authorizer.AuthorizeTierOperation(ctx, policy.Name, tierName)
	if err != nil {
		return nil, err
	}

	return r.Store.Create(ctx, obj, val, includeUninitialized)
}

func (r *REST) Update(ctx genericapirequest.Context, name string, objInfo rest.UpdatedObjectInfo, val rest.ValidateObjectFunc, valUp rest.ValidateObjectUpdateFunc) (runtime.Object, bool, error) {
	tierName, _ := util.GetTierPolicy(name)
	err := r.authorizer.AuthorizeTierOperation(ctx, name, tierName)
	if err != nil {
		return nil, false, err
	}

	return r.Store.Update(ctx, name, objInfo, val, valUp)
}

// Get retrieves the item from storage.
func (r *REST) Get(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	tierName, _ := util.GetTierPolicy(name)
	err := r.authorizer.AuthorizeTierOperation(ctx, name, tierName)
	if err != nil {
		return nil, err
	}

	return r.Store.Get(ctx, name, options)
}

func (r *REST) Delete(ctx genericapirequest.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	tierName, _ := util.GetTierPolicy(name)
	err := r.authorizer.AuthorizeTierOperation(ctx, name, tierName)
	if err != nil {
		return nil, false, err
	}

	return r.Store.Delete(ctx, name, options)
}

func (r *REST) Watch(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	tierName, err := util.GetTierNameFromSelector(options)
	if err != nil {
		return nil, err
	}
	err = r.authorizer.AuthorizeTierOperation(ctx, "", tierName)
	if err != nil {
		return nil, err
	}

	return r.Store.Watch(ctx, options)
}
