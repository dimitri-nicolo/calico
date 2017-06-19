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

package policy

import (
	"fmt"
	"strings"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation/path"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/client-go/pkg/api"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
	legacyStore *legacyREST
}

// KeyRootFunc returns the root etcd key for policy.
// This is used for operations that work on the entire collection.
func KeyRootFunc(ctx genericapirequest.Context, prefix string) string {
	key := prefix
	ns, ok := genericapirequest.NamespaceFrom(ctx)
	if ok && len(ns) > 0 {
		key = key + "/" + ns
	}
	return key
}

// KeyFunc is the default function for constructing storage paths for Policy resource.
func KeyFunc(ctx genericapirequest.Context, prefix string, name string) (string, error) {
	key := KeyRootFunc(ctx, prefix)
	if len(name) == 0 {
		return "", kubeerr.NewBadRequest("Name parameter required.")
	}
	if msgs := path.IsValidPathSegmentName(name); len(msgs) != 0 {
		return "", kubeerr.NewBadRequest(fmt.Sprintf("Name parameter invalid: %q: %s", name, strings.Join(msgs, ";")))
	}
	key = key + "/" + name
	return key, nil
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(optsGetter generic.RESTOptionsGetter) *REST {
	store := &genericregistry.Store{
		Copier:      api.Scheme,
		NewFunc:     func() runtime.Object { return &calico.Policy{} },
		NewListFunc: func() runtime.Object { return &calico.PolicyList{} },
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.Policy).Name, nil
		},
		PredicateFunc:     MatchPolicy,
		QualifiedResource: api.Resource("policies"),

		KeyFunc: func(ctx genericapirequest.Context, name string) (string, error) {
			return KeyFunc(ctx, "/policy", name)
		},
		KeyRootFunc: func(ctx genericapirequest.Context) string {
			return KeyRootFunc(ctx, "/policy")
		},
		CreateStrategy: Strategy,
		UpdateStrategy: Strategy,
		DeleteStrategy: Strategy,
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	legacyStore := NewLegacyREST(store)
	return &REST{store, legacyStore}
}

func (r *REST) Create(ctx genericapirequest.Context, obj runtime.Object) (runtime.Object, error) {
	err := r.legacyStore.create(obj)
	if err != nil {
		return nil, err
	}
	obj, err = r.Store.Create(ctx, obj)
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
