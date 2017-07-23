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

package node

import (
	"fmt"

	libcalico "github.com/projectcalico/libcalico-go/lib/api"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/pkg/api"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
	legacyStore *legacyREST
	authorizer  authorizer.Authorizer
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(optsGetter generic.RESTOptionsGetter, authorizer authorizer.Authorizer) *REST {
	store := &genericregistry.Store{
		Copier:      api.Scheme,
		NewFunc:     func() runtime.Object { return &calico.Node{} },
		NewListFunc: func() runtime.Object { return &calico.NodeList{} },
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.Node).Name, nil
		},
		PredicateFunc:     MatchNode,
		QualifiedResource: calico.Resource("nodes"),

		CreateStrategy: Strategy,
		UpdateStrategy: Strategy,
		DeleteStrategy: Strategy,
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	legacyStore := NewLegacyREST(store)
	return &REST{store, legacyStore, authorizer}
}

func (r *REST) List(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	nMD := libcalico.NodeMetadata{}
	if options != nil && options.LabelSelector != nil {
		reqs, _ := options.LabelSelector.Requirements()
		for _, req := range reqs {
			if req.Key() == "name" {
				nMD.Name, _ = req.Values().PopAny()
			}
		}
	}
	nodes, err := r.legacyStore.list(nMD)
	if err != nil {
		return nil, err
	}

	apiNodes := &calico.NodeList{}
	apiNodes.APIVersion = "calico.tigera.io/v1"
	apiNodes.Kind = "List"
	for _, node := range nodes.Items {
		ae := calico.Node{}
		ae.APIVersion = "calico.tigera.io/v1"
		ae.Kind = "Node"
		nMD := node.Metadata
		ae.Name = nMD.Name
		ae.Spec = node.Spec
		apiNodes.Items = append(apiNodes.Items, ae)
	}
	return apiNodes, nil
}

func (r *REST) Create(ctx genericapirequest.Context, obj runtime.Object) (runtime.Object, error) {
	return nil, fmt.Errorf("Create not supported.")
}

func (r *REST) Update(ctx genericapirequest.Context, name string, objInfo rest.UpdatedObjectInfo) (runtime.Object, bool, error) {
	return nil, false, fmt.Errorf("Update not supported.")
}

// Get retrieves the item from storage.
func (r *REST) Get(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	nMD := libcalico.NodeMetadata{}
	nMD.Name = name

	nodes, err := r.legacyStore.list(nMD)
	if err != nil {
		return nil, err
	}
	node := nodes.Items[0]

	apiNode := &calico.Node{}
	apiNode.APIVersion = "calico.tigera.io/v1"
	apiNode.Kind = "Node"
	apiNode.Kind = "Node"
	apiNode.Name = name
	apiNode.Spec = node.Spec

	return apiNode, nil
}

func (r *REST) Delete(ctx genericapirequest.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return nil, false, fmt.Errorf("Delete not supported.")
}

func (r *REST) Watch(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("Watch not supported.")
}
