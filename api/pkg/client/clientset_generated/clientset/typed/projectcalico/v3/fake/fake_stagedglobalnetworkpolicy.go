// Copyright (c) 2023 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeStagedGlobalNetworkPolicies implements StagedGlobalNetworkPolicyInterface
type FakeStagedGlobalNetworkPolicies struct {
	Fake *FakeProjectcalicoV3
}

var stagedglobalnetworkpoliciesResource = v3.SchemeGroupVersion.WithResource("stagedglobalnetworkpolicies")

var stagedglobalnetworkpoliciesKind = v3.SchemeGroupVersion.WithKind("StagedGlobalNetworkPolicy")

// Get takes name of the stagedGlobalNetworkPolicy, and returns the corresponding stagedGlobalNetworkPolicy object, and an error if there is any.
func (c *FakeStagedGlobalNetworkPolicies) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.StagedGlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(stagedglobalnetworkpoliciesResource, name), &v3.StagedGlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.StagedGlobalNetworkPolicy), err
}

// List takes label and field selectors, and returns the list of StagedGlobalNetworkPolicies that match those selectors.
func (c *FakeStagedGlobalNetworkPolicies) List(ctx context.Context, opts v1.ListOptions) (result *v3.StagedGlobalNetworkPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(stagedglobalnetworkpoliciesResource, stagedglobalnetworkpoliciesKind, opts), &v3.StagedGlobalNetworkPolicyList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.StagedGlobalNetworkPolicyList{ListMeta: obj.(*v3.StagedGlobalNetworkPolicyList).ListMeta}
	for _, item := range obj.(*v3.StagedGlobalNetworkPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested stagedGlobalNetworkPolicies.
func (c *FakeStagedGlobalNetworkPolicies) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(stagedglobalnetworkpoliciesResource, opts))
}

// Create takes the representation of a stagedGlobalNetworkPolicy and creates it.  Returns the server's representation of the stagedGlobalNetworkPolicy, and an error, if there is any.
func (c *FakeStagedGlobalNetworkPolicies) Create(ctx context.Context, stagedGlobalNetworkPolicy *v3.StagedGlobalNetworkPolicy, opts v1.CreateOptions) (result *v3.StagedGlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(stagedglobalnetworkpoliciesResource, stagedGlobalNetworkPolicy), &v3.StagedGlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.StagedGlobalNetworkPolicy), err
}

// Update takes the representation of a stagedGlobalNetworkPolicy and updates it. Returns the server's representation of the stagedGlobalNetworkPolicy, and an error, if there is any.
func (c *FakeStagedGlobalNetworkPolicies) Update(ctx context.Context, stagedGlobalNetworkPolicy *v3.StagedGlobalNetworkPolicy, opts v1.UpdateOptions) (result *v3.StagedGlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(stagedglobalnetworkpoliciesResource, stagedGlobalNetworkPolicy), &v3.StagedGlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.StagedGlobalNetworkPolicy), err
}

// Delete takes name of the stagedGlobalNetworkPolicy and deletes it. Returns an error if one occurs.
func (c *FakeStagedGlobalNetworkPolicies) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(stagedglobalnetworkpoliciesResource, name, opts), &v3.StagedGlobalNetworkPolicy{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeStagedGlobalNetworkPolicies) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(stagedglobalnetworkpoliciesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.StagedGlobalNetworkPolicyList{})
	return err
}

// Patch applies the patch and returns the patched stagedGlobalNetworkPolicy.
func (c *FakeStagedGlobalNetworkPolicies) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.StagedGlobalNetworkPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(stagedglobalnetworkpoliciesResource, name, pt, data, subresources...), &v3.StagedGlobalNetworkPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.StagedGlobalNetworkPolicy), err
}
