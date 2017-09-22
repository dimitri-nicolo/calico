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

	"github.com/golang/glog"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/calico/server"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/filters"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/pkg/api"
)

const (
	policyDelim = "."
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
	authorizer authorizer.Authorizer
}

// EmptyObject returns an empty instance
func EmptyObject() runtime.Object {
	return &calico.NetworkPolicy{}
}

// NewList returns a new shell of a binding list
func NewList() runtime.Object {
	return &calico.NetworkPolicyList{
	//TypeMeta: metav1.TypeMeta{},
	//Items:    []calico.NetworkPolicy{},
	}
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(opts server.Options) *REST {
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
		1000,
		&calico.NetworkPolicy{},
		prefix,
		keyFunc,
		Strategy,
		func() runtime.Object { return &calico.NetworkPolicyList{} },
		GetAttrs,
		storage.NoTriggerPublisher,
	)
	store := &genericregistry.Store{
		Copier:      api.Scheme,
		NewFunc:     func() runtime.Object { return &calico.NetworkPolicy{} },
		NewListFunc: func() runtime.Object { return &calico.NetworkPolicyList{} },
		KeyRootFunc: opts.KeyRootFunc(true),
		KeyFunc:     opts.KeyFunc(true),
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*calico.NetworkPolicy).Name, nil
		},
		PredicateFunc:     MatchPolicy,
		QualifiedResource: calico.Resource("networkpolicies"),

		CreateStrategy:          Strategy,
		UpdateStrategy:          Strategy,
		DeleteStrategy:          Strategy,
		EnableGarbageCollection: true,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}

	return &REST{store, opts.Authorizer}
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
	if r.authorizer == nil {
		glog.Infof("Authorization disabled for testing purposes")
		return nil
	}
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
	attrs.ResourceRequest = attributes.IsResourceRequest()
	attrs.Path = "/apis/calico.tigera.io/v1/tiers/" + tierName
	glog.Infof("Tier Auth Attributes for the given Policy")
	logAuthorizerAttributes(attrs)
	authorized, reason, err := r.authorizer.Authorize(attrs)
	if err != nil {
		return err
	}
	if !authorized {
		if reason == "" {
			reason = fmt.Sprintf("(Forbidden) Policy operation is assocaited with tier %s. User \"%s\" cannot %s tiers.calico.tigera.io at the cluster scope. (get tiers.calico.tigera.io)", tierName, attrs.User.GetName(), attrs.Verb)
		}
		return fmt.Errorf(reason)
	}
	return nil
}

func getTierPolicy(policyName string) (string, string) {
	policySlice := strings.Split(policyName, policyDelim)
	if len(policySlice) < 2 {
		return "default", policySlice[0]
	}
	return policySlice[0], policySlice[1]
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

func (r *REST) Create(ctx genericapirequest.Context, obj runtime.Object, includeUninitialized bool) (runtime.Object, error) {
	policy := obj.(*calico.NetworkPolicy)
	// Is Tier prepended. If not prepend default?
	tierName, _ := getTierPolicy(policy.Name)
	err := r.authorizeTierOperation(ctx, tierName)
	if err != nil {
		return nil, err
	}

	return r.Store.Create(ctx, obj, false)
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

	return r.Store.Delete(ctx, name, options)
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
