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

// FakePolicies implements PolicyInterface
type FakePolicies struct {
	Fake *FakeCalico
	ns   string
}

var policiesResource = schema.GroupVersionResource{Group: "calico.tigera.io", Version: "", Resource: "policies"}

var policiesKind = schema.GroupVersionKind{Group: "calico.tigera.io", Version: "", Kind: "Policy"}

func (c *FakePolicies) Create(policy *calico.Policy) (result *calico.Policy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(policiesResource, c.ns, policy), &calico.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Policy), err
}

func (c *FakePolicies) Update(policy *calico.Policy) (result *calico.Policy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(policiesResource, c.ns, policy), &calico.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Policy), err
}

func (c *FakePolicies) UpdateStatus(policy *calico.Policy) (*calico.Policy, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(policiesResource, "status", c.ns, policy), &calico.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Policy), err
}

func (c *FakePolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(policiesResource, c.ns, name), &calico.Policy{})

	return err
}

func (c *FakePolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(policiesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &calico.PolicyList{})
	return err
}

func (c *FakePolicies) Get(name string, options v1.GetOptions) (result *calico.Policy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(policiesResource, c.ns, name), &calico.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Policy), err
}

func (c *FakePolicies) List(opts v1.ListOptions) (result *calico.PolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(policiesResource, policiesKind, c.ns, opts), &calico.PolicyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &calico.PolicyList{}
	for _, item := range obj.(*calico.PolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested policies.
func (c *FakePolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(policiesResource, c.ns, opts))

}

// Patch applies the patch and returns the patched policy.
func (c *FakePolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Policy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(policiesResource, c.ns, name, data, subresources...), &calico.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Policy), err
}
