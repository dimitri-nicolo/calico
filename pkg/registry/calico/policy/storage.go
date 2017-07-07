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

	"github.com/golang/glog"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/filters"
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
		NewFunc:     func() runtime.Object { return &calico.Policy{} },
		NewListFunc: func() runtime.Object { return &calico.PolicyList{} },
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.Policy).Name, nil
		},
		PredicateFunc:     MatchPolicy,
		QualifiedResource: calico.Resource("policies"),

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

// TODO: Remove this. Its purely for debugging purposes.
func logAuthorizerAttributes(requestAttributes authorizer.Attributes) {
	glog.Infof("Authorizer APIGroup: %s", requestAttributes.GetAPIGroup())
	glog.Infof("Authorizer APIVersion: %s", requestAttributes.GetAPIVersion())
	glog.Infof("Authorizer Name: %s", requestAttributes.GetName())
	glog.Infof("Authorizer Namespace: %s", requestAttributes.GetNamespace())
	glog.Infof("Authorizer Resource: %s", requestAttributes.GetResource())
	glog.Infof("Authorizer Subresource: %s", requestAttributes.GetSubresource())
	glog.Infof("Authorizer User: %s", requestAttributes.GetUser())
	glog.Infof("Authorizer Verb: %s", requestAttributes.GetVerb())
}

func getTierNamesFromSelector(options *metainternalversion.ListOptions) []string {
	tierNames := []string{}
	if options.LabelSelector == nil {
		options.LabelSelector = labels.NewSelector()
		requirement, _ := labels.NewRequirement("tier", selection.DoubleEquals, []string{"default"})
		options.LabelSelector = options.LabelSelector.Add(*requirement)
	}
	requirements, _ := options.LabelSelector.Requirements()
	for _, requirement := range requirements {
		if requirement.Key() == "tier" {
			tierNames = append(tierNames, requirement.Values().UnsortedList()...)
		}
	}

	return tierNames
}

func (r *REST) authorizeTierOperation(ctx genericapirequest.Context, tierName string) error {
	attributes, err := filters.GetAuthorizerAttributes(ctx)
	if err != nil {
		return err
	}
	attrs := authorizer.AttributesRecord{}
	attrs.APIGroup = attributes.GetAPIGroup()
	attrs.APIVersion = attributes.GetAPIVersion()
	attrs.Name = tierName
	attrs.Resource = "tiers"
	attrs.User = attributes.GetUser()
	attrs.Verb = attributes.GetVerb()
	glog.Infof("Tier Auth Attributes for the given Policy")
	logAuthorizerAttributes(attrs)
	authorized, reason, err := r.authorizer.Authorize(attrs)
	if err != nil {
		return err
	}
	if !authorized {
		return fmt.Errorf(reason)
	}
	return nil
}

func (r *REST) List(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	tierNames := getTierNamesFromSelector(options)
	for _, tierName := range tierNames {
		err := r.authorizeTierOperation(ctx, tierName)
		if err != nil {
			return nil, err
		}
	}

	return r.Store.List(ctx, options)
}

func (r *REST) Create(ctx genericapirequest.Context, obj runtime.Object) (runtime.Object, error) {
	policy := obj.(*calico.Policy)
	// Is Tier prepended. If not prepend default?
	tierName, _ := getTierPolicy(policy.Name)
	err := r.authorizeTierOperation(ctx, tierName)
	if err != nil {
		return nil, err
	}

	err = r.legacyStore.create(obj)
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

func (r *REST) Update(ctx genericapirequest.Context, name string, objInfo rest.UpdatedObjectInfo) (runtime.Object, bool, error) {
	tierName, _ := getTierPolicy(name)
	err := r.authorizeTierOperation(ctx, tierName)
	if err != nil {
		return nil, false, err
	}

	return r.Store.Update(ctx, name, objInfo)
}

// Get retrieves the item from storage.
func (r *REST) Get(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	tierName, _ := getTierPolicy(name)
	err := r.authorizeTierOperation(ctx, tierName)
	if err != nil {
		return nil, err
	}

	return r.Store.Get(ctx, name, options)
}

func (r *REST) Delete(ctx genericapirequest.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	tierName, _ := getTierPolicy(name)
	err := r.authorizeTierOperation(ctx, tierName)
	if err != nil {
		return nil, false, err
	}

	_, err = r.legacyStore.delete(name)
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

func (r *REST) Watch(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	tierNames := getTierNamesFromSelector(options)
	for _, tierName := range tierNames {
		err := r.authorizeTierOperation(ctx, tierName)
		if err != nil {
			return nil, err
		}
	}

	return r.Store.Watch(ctx, options)
}
