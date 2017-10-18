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

// FakeTiers implements TierInterface
type FakeTiers struct {
	Fake *FakeProjectcalico
	ns   string
}

var tiersResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "tiers"}

var tiersKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "Tier"}

func (c *FakeTiers) Create(tier *calico.Tier) (result *calico.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(tiersResource, c.ns, tier), &calico.Tier{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Tier), err
}

func (c *FakeTiers) Update(tier *calico.Tier) (result *calico.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(tiersResource, c.ns, tier), &calico.Tier{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Tier), err
}

func (c *FakeTiers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(tiersResource, c.ns, name), &calico.Tier{})

	return err
}

func (c *FakeTiers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(tiersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &calico.TierList{})
	return err
}

func (c *FakeTiers) Get(name string, options v1.GetOptions) (result *calico.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(tiersResource, c.ns, name), &calico.Tier{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Tier), err
}

func (c *FakeTiers) List(opts v1.ListOptions) (result *calico.TierList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(tiersResource, tiersKind, c.ns, opts), &calico.TierList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &calico.TierList{}
	for _, item := range obj.(*calico.TierList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tiers.
func (c *FakeTiers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(tiersResource, c.ns, opts))

}

// Patch applies the patch and returns the patched tier.
func (c *FakeTiers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(tiersResource, c.ns, name, data, subresources...), &calico.Tier{})

	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Tier), err
}
