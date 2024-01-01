// Copyright (c) 2024 Tigera, Inc. All rights reserved.

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

// FakePolicyRecommendationScopes implements PolicyRecommendationScopeInterface
type FakePolicyRecommendationScopes struct {
	Fake *FakeProjectcalicoV3
}

var policyrecommendationscopesResource = v3.SchemeGroupVersion.WithResource("policyrecommendationscopes")

var policyrecommendationscopesKind = v3.SchemeGroupVersion.WithKind("PolicyRecommendationScope")

// Get takes name of the policyRecommendationScope, and returns the corresponding policyRecommendationScope object, and an error if there is any.
func (c *FakePolicyRecommendationScopes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.PolicyRecommendationScope, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(policyrecommendationscopesResource, name), &v3.PolicyRecommendationScope{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PolicyRecommendationScope), err
}

// List takes label and field selectors, and returns the list of PolicyRecommendationScopes that match those selectors.
func (c *FakePolicyRecommendationScopes) List(ctx context.Context, opts v1.ListOptions) (result *v3.PolicyRecommendationScopeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(policyrecommendationscopesResource, policyrecommendationscopesKind, opts), &v3.PolicyRecommendationScopeList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.PolicyRecommendationScopeList{ListMeta: obj.(*v3.PolicyRecommendationScopeList).ListMeta}
	for _, item := range obj.(*v3.PolicyRecommendationScopeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested policyRecommendationScopes.
func (c *FakePolicyRecommendationScopes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(policyrecommendationscopesResource, opts))
}

// Create takes the representation of a policyRecommendationScope and creates it.  Returns the server's representation of the policyRecommendationScope, and an error, if there is any.
func (c *FakePolicyRecommendationScopes) Create(ctx context.Context, policyRecommendationScope *v3.PolicyRecommendationScope, opts v1.CreateOptions) (result *v3.PolicyRecommendationScope, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(policyrecommendationscopesResource, policyRecommendationScope), &v3.PolicyRecommendationScope{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PolicyRecommendationScope), err
}

// Update takes the representation of a policyRecommendationScope and updates it. Returns the server's representation of the policyRecommendationScope, and an error, if there is any.
func (c *FakePolicyRecommendationScopes) Update(ctx context.Context, policyRecommendationScope *v3.PolicyRecommendationScope, opts v1.UpdateOptions) (result *v3.PolicyRecommendationScope, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(policyrecommendationscopesResource, policyRecommendationScope), &v3.PolicyRecommendationScope{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PolicyRecommendationScope), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePolicyRecommendationScopes) UpdateStatus(ctx context.Context, policyRecommendationScope *v3.PolicyRecommendationScope, opts v1.UpdateOptions) (*v3.PolicyRecommendationScope, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(policyrecommendationscopesResource, "status", policyRecommendationScope), &v3.PolicyRecommendationScope{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PolicyRecommendationScope), err
}

// Delete takes name of the policyRecommendationScope and deletes it. Returns an error if one occurs.
func (c *FakePolicyRecommendationScopes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(policyrecommendationscopesResource, name, opts), &v3.PolicyRecommendationScope{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePolicyRecommendationScopes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(policyrecommendationscopesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.PolicyRecommendationScopeList{})
	return err
}

// Patch applies the patch and returns the patched policyRecommendationScope.
func (c *FakePolicyRecommendationScopes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.PolicyRecommendationScope, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(policyrecommendationscopesResource, name, pt, data, subresources...), &v3.PolicyRecommendationScope{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PolicyRecommendationScope), err
}
