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

package tier

import (
	"fmt"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/pkg/api"
)

// REST implements a RESTStorage for tiers
type REST struct {
	*genericregistry.Store
	legacyStore *legacyREST
	policyStore rest.Storage
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(optsGetter generic.RESTOptionsGetter, policyStore rest.Storage) *REST {
	store := &genericregistry.Store{
		Copier:      api.Scheme,
		NewFunc:     func() runtime.Object { return &calico.Tier{} },
		NewListFunc: func() runtime.Object { return &calico.TierList{} },
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.Tier).Name, nil
		},
		PredicateFunc:     MatchTier,
		QualifiedResource: calico.Resource("tiers"),

		CreateStrategy: Strategy,
		UpdateStrategy: Strategy,
		DeleteStrategy: Strategy,
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	legacyStore := NewLegacyREST(store)
	return &REST{store, legacyStore, policyStore}
}

func (r *REST) Create(ctx genericapirequest.Context, obj runtime.Object) (runtime.Object, error) {
	err := r.legacyStore.create(obj)
	if err != nil {
		return nil, err
	}
	obj, err = r.Store.Create(ctx, obj, false)
	if err != nil {
		objectName, err := r.ObjectNameFunc(obj)
		if err != nil {
			panic("failed parsing object name of an already stored object")
		}
		r.legacyStore.delete(objectName)
		return nil, err
	}
	return obj, nil
}

func (r *REST) Delete(ctx genericapirequest.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	count := r.legacyStore.getPolicyCount(name)
	if count != 0 {
		return nil, false, fmt.Errorf("delete on a tier consisting of policies not implemented currently.")
	}
	_, err := r.legacyStore.delete(name)
	if err != nil {
		return nil, false, err
	}
	obj, ok, err := r.Store.Delete(ctx, name, options)
	if err != nil || !ok {
		// TODO
		//r.legacyStore.create(policy)
		return nil, false, err
	}
	return obj, ok, err
}
