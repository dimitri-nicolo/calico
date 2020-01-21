// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeStagedNetworkPolicies implements StagedNetworkPolicyInterface
type FakeStagedNetworkPolicies struct {
	Fake *FakeProjectcalico
	ns   string
}

var stagednetworkpoliciesResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "stagednetworkpolicies"}

var stagednetworkpoliciesKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "StagedNetworkPolicy"}

// Get takes name of the stagedNetworkPolicy, and returns the corresponding stagedNetworkPolicy object, and an error if there is any.
func (c *FakeStagedNetworkPolicies) Get(name string, options v1.GetOptions) (result *projectcalico.StagedNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(stagednetworkpoliciesResource, c.ns, name), &projectcalico.StagedNetworkPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.StagedNetworkPolicy), err
}

// List takes label and field selectors, and returns the list of StagedNetworkPolicies that match those selectors.
func (c *FakeStagedNetworkPolicies) List(opts v1.ListOptions) (result *projectcalico.StagedNetworkPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(stagednetworkpoliciesResource, stagednetworkpoliciesKind, c.ns, opts), &projectcalico.StagedNetworkPolicyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.StagedNetworkPolicyList{ListMeta: obj.(*projectcalico.StagedNetworkPolicyList).ListMeta}
	for _, item := range obj.(*projectcalico.StagedNetworkPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested stagedNetworkPolicies.
func (c *FakeStagedNetworkPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(stagednetworkpoliciesResource, c.ns, opts))

}

// Create takes the representation of a stagedNetworkPolicy and creates it.  Returns the server's representation of the stagedNetworkPolicy, and an error, if there is any.
func (c *FakeStagedNetworkPolicies) Create(stagedNetworkPolicy *projectcalico.StagedNetworkPolicy) (result *projectcalico.StagedNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(stagednetworkpoliciesResource, c.ns, stagedNetworkPolicy), &projectcalico.StagedNetworkPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.StagedNetworkPolicy), err
}

// Update takes the representation of a stagedNetworkPolicy and updates it. Returns the server's representation of the stagedNetworkPolicy, and an error, if there is any.
func (c *FakeStagedNetworkPolicies) Update(stagedNetworkPolicy *projectcalico.StagedNetworkPolicy) (result *projectcalico.StagedNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(stagednetworkpoliciesResource, c.ns, stagedNetworkPolicy), &projectcalico.StagedNetworkPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.StagedNetworkPolicy), err
}

// Delete takes name of the stagedNetworkPolicy and deletes it. Returns an error if one occurs.
func (c *FakeStagedNetworkPolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(stagednetworkpoliciesResource, c.ns, name), &projectcalico.StagedNetworkPolicy{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeStagedNetworkPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(stagednetworkpoliciesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &projectcalico.StagedNetworkPolicyList{})
	return err
}

// Patch applies the patch and returns the patched stagedNetworkPolicy.
func (c *FakeStagedNetworkPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.StagedNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(stagednetworkpoliciesResource, c.ns, name, pt, data, subresources...), &projectcalico.StagedNetworkPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.StagedNetworkPolicy), err
}
