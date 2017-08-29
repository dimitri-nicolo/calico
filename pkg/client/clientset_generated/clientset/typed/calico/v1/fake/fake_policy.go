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
	v1 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePolicies implements PolicyInterface
type FakePolicies struct {
	Fake *FakeCalicoV1
	ns   string
}

var policiesResource = schema.GroupVersionResource{Group: "calico.tigera.io", Version: "v1", Resource: "policies"}

func (c *FakePolicies) Create(policy *v1.Policy) (result *v1.Policy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(policiesResource, c.ns, policy), &v1.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Policy), err
}

func (c *FakePolicies) Update(policy *v1.Policy) (result *v1.Policy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(policiesResource, c.ns, policy), &v1.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Policy), err
}

func (c *FakePolicies) UpdateStatus(policy *v1.Policy) (*v1.Policy, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(policiesResource, "status", c.ns, policy), &v1.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Policy), err
}

func (c *FakePolicies) Delete(name string, options *meta_v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(policiesResource, c.ns, name), &v1.Policy{})

	return err
}

func (c *FakePolicies) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(policiesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1.PolicyList{})
	return err
}

func (c *FakePolicies) Get(name string, options meta_v1.GetOptions) (result *v1.Policy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(policiesResource, c.ns, name), &v1.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Policy), err
}

func (c *FakePolicies) List(opts meta_v1.ListOptions) (result *v1.PolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(policiesResource, c.ns, opts), &v1.PolicyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.PolicyList{}
	for _, item := range obj.(*v1.PolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested policies.
func (c *FakePolicies) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(policiesResource, c.ns, opts))

}

// Patch applies the patch and returns the patched policy.
func (c *FakePolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Policy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(policiesResource, c.ns, name, data, subresources...), &v1.Policy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Policy), err
}
