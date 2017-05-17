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
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/client-go/pkg/api"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
}

// KeyRootFunc returns the root etcd key for policy.
// This is used for operations that work on the entire collection.
func KeyRootFunc(ctx genericapirequest.Context, prefix string) string {
	key := prefix
	ns, ok := genericapirequest.NamespaceFrom(ctx)
	if ok && len(ns) > 0 {
		key = key + "/tier/" + ns + "/policy"
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

		CreateStrategy: Strategy,
		UpdateStrategy: Strategy,
		DeleteStrategy: Strategy,
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}

	// Setup the key funcs
	opts, err := options.RESTOptions.GetRESTOptions(store.QualifiedResource)
	if err != nil {
		panic(err)
	}

	opts.ResourcePrefix = "/calico/v1"
	prefix := opts.ResourcePrefix + "/policy"

	store.KeyFunc = func(ctx genericapirequest.Context, name string) (string, error) {
		return KeyFunc(ctx, prefix, name)
	}
	store.KeyRootFunc = func(ctx genericapirequest.Context) string {
		return KeyRootFunc(ctx, prefix)
	}

	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}
	return &REST{store}
}
