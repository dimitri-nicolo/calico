/*
Copyright 2017 Tigera.
*/package fake

import (
	v2 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGlobalNetworkPolicies implements GlobalNetworkPolicyInterface
type FakeGlobalNetworkPolicies struct {
	Fake *FakeProjectcalicoV2
}

var globalnetworkpoliciesResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v2", Resource: "globalnetworkpolicies"}

var globalnetworkpoliciesKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v2", Kind: "GlobalNetworkPolicy"}

// Get takes name of the globalNetworkPolicy, and returns the corresponding globalNetworkPolicy object, and an error if there is any.
func (c *FakeGlobalNetworkPolicies) Get(name string, options v1.GetOptions) (result *v2.GlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalnetworkpoliciesResource, name), &v2.GlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2.GlobalNetworkPolicy), err
}

// List takes label and field selectors, and returns the list of GlobalNetworkPolicies that match those selectors.
func (c *FakeGlobalNetworkPolicies) List(opts v1.ListOptions) (result *v2.GlobalNetworkPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalnetworkpoliciesResource, globalnetworkpoliciesKind, opts), &v2.GlobalNetworkPolicyList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v2.GlobalNetworkPolicyList{}
	for _, item := range obj.(*v2.GlobalNetworkPolicyList).Items {
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

// Create takes the representation of a globalNetworkPolicy and creates it.  Returns the server's representation of the globalNetworkPolicy, and an error, if there is any.
func (c *FakeGlobalNetworkPolicies) Create(globalNetworkPolicy *v2.GlobalNetworkPolicy) (result *v2.GlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalnetworkpoliciesResource, globalNetworkPolicy), &v2.GlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2.GlobalNetworkPolicy), err
}

// Update takes the representation of a globalNetworkPolicy and updates it. Returns the server's representation of the globalNetworkPolicy, and an error, if there is any.
func (c *FakeGlobalNetworkPolicies) Update(globalNetworkPolicy *v2.GlobalNetworkPolicy) (result *v2.GlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalnetworkpoliciesResource, globalNetworkPolicy), &v2.GlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2.GlobalNetworkPolicy), err
}

// Delete takes name of the globalNetworkPolicy and deletes it. Returns an error if one occurs.
func (c *FakeGlobalNetworkPolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(globalnetworkpoliciesResource, name), &v2.GlobalNetworkPolicy{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobalNetworkPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalnetworkpoliciesResource, listOptions)

	_, err := c.Fake.Invokes(action, &v2.GlobalNetworkPolicyList{})
	return err
}

// Patch applies the patch and returns the patched globalNetworkPolicy.
func (c *FakeGlobalNetworkPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v2.GlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalnetworkpoliciesResource, name, data, subresources...), &v2.GlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2.GlobalNetworkPolicy), err
}
