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

package fake

import (
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGlobalNetworkPolicies implements GlobalNetworkPolicyInterface
type FakeGlobalNetworkPolicies struct {
	Fake *FakeProjectcalico
}

var globalnetworkpoliciesResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "globalnetworkpolicies"}

var globalnetworkpoliciesKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "GlobalNetworkPolicy"}

func (c *FakeGlobalNetworkPolicies) Create(globalNetworkPolicy *calico.GlobalNetworkPolicy) (result *calico.GlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalnetworkpoliciesResource, globalNetworkPolicy), &calico.GlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.GlobalNetworkPolicy), err
}

func (c *FakeGlobalNetworkPolicies) Update(globalNetworkPolicy *calico.GlobalNetworkPolicy) (result *calico.GlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalnetworkpoliciesResource, globalNetworkPolicy), &calico.GlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.GlobalNetworkPolicy), err
}

func (c *FakeGlobalNetworkPolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(globalnetworkpoliciesResource, name), &calico.GlobalNetworkPolicy{})
	return err
}

func (c *FakeGlobalNetworkPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalnetworkpoliciesResource, listOptions)

	_, err := c.Fake.Invokes(action, &calico.GlobalNetworkPolicyList{})
	return err
}

func (c *FakeGlobalNetworkPolicies) Get(name string, options v1.GetOptions) (result *calico.GlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalnetworkpoliciesResource, name), &calico.GlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.GlobalNetworkPolicy), err
}

func (c *FakeGlobalNetworkPolicies) List(opts v1.ListOptions) (result *calico.GlobalNetworkPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalnetworkpoliciesResource, globalnetworkpoliciesKind, opts), &calico.GlobalNetworkPolicyList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &calico.GlobalNetworkPolicyList{}
	for _, item := range obj.(*calico.GlobalNetworkPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globalNetworkPolicies.
func (c *FakeGlobalNetworkPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(globalnetworkpoliciesResource, opts))
}

// Patch applies the patch and returns the patched globalNetworkPolicy.
func (c *FakeGlobalNetworkPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.GlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalnetworkpoliciesResource, name, data, subresources...), &calico.GlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.GlobalNetworkPolicy), err
}
