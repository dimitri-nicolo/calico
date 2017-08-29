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

// FakeTiers implements TierInterface
type FakeTiers struct {
	Fake *FakeCalicoV1
}

var tiersResource = schema.GroupVersionResource{Group: "calico.tigera.io", Version: "v1", Resource: "tiers"}

func (c *FakeTiers) Create(tier *v1.Tier) (result *v1.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(tiersResource, tier), &v1.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Tier), err
}

func (c *FakeTiers) Update(tier *v1.Tier) (result *v1.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(tiersResource, tier), &v1.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Tier), err
}

func (c *FakeTiers) Delete(name string, options *meta_v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(tiersResource, name), &v1.Tier{})
	return err
}

func (c *FakeTiers) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(tiersResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1.TierList{})
	return err
}

func (c *FakeTiers) Get(name string, options meta_v1.GetOptions) (result *v1.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(tiersResource, name), &v1.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Tier), err
}

func (c *FakeTiers) List(opts meta_v1.ListOptions) (result *v1.TierList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(tiersResource, opts), &v1.TierList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.TierList{}
	for _, item := range obj.(*v1.TierList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tiers.
func (c *FakeTiers) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(tiersResource, opts))
}

// Patch applies the patch and returns the patched tier.
func (c *FakeTiers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(tiersResource, name, data, subresources...), &v1.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Tier), err
}
