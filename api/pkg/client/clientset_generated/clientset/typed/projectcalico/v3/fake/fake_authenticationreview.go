// Copyright (c) 2022 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAuthenticationReviews implements AuthenticationReviewInterface
type FakeAuthenticationReviews struct {
	Fake *FakeProjectcalicoV3
}

var authenticationreviewsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "authenticationreviews"}

var authenticationreviewsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "AuthenticationReview"}

// Get takes name of the authenticationReview, and returns the corresponding authenticationReview object, and an error if there is any.
func (c *FakeAuthenticationReviews) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.AuthenticationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(authenticationreviewsResource, name), &v3.AuthenticationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthenticationReview), err
}

// List takes label and field selectors, and returns the list of AuthenticationReviews that match those selectors.
func (c *FakeAuthenticationReviews) List(ctx context.Context, opts v1.ListOptions) (result *v3.AuthenticationReviewList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(authenticationreviewsResource, authenticationreviewsKind, opts), &v3.AuthenticationReviewList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.AuthenticationReviewList{ListMeta: obj.(*v3.AuthenticationReviewList).ListMeta}
	for _, item := range obj.(*v3.AuthenticationReviewList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested authenticationReviews.
func (c *FakeAuthenticationReviews) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(authenticationreviewsResource, opts))
}

// Create takes the representation of a authenticationReview and creates it.  Returns the server's representation of the authenticationReview, and an error, if there is any.
func (c *FakeAuthenticationReviews) Create(ctx context.Context, authenticationReview *v3.AuthenticationReview, opts v1.CreateOptions) (result *v3.AuthenticationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(authenticationreviewsResource, authenticationReview), &v3.AuthenticationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthenticationReview), err
}

// Update takes the representation of a authenticationReview and updates it. Returns the server's representation of the authenticationReview, and an error, if there is any.
func (c *FakeAuthenticationReviews) Update(ctx context.Context, authenticationReview *v3.AuthenticationReview, opts v1.UpdateOptions) (result *v3.AuthenticationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(authenticationreviewsResource, authenticationReview), &v3.AuthenticationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthenticationReview), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAuthenticationReviews) UpdateStatus(ctx context.Context, authenticationReview *v3.AuthenticationReview, opts v1.UpdateOptions) (*v3.AuthenticationReview, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(authenticationreviewsResource, "status", authenticationReview), &v3.AuthenticationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthenticationReview), err
}

// Delete takes name of the authenticationReview and deletes it. Returns an error if one occurs.
func (c *FakeAuthenticationReviews) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(authenticationreviewsResource, name), &v3.AuthenticationReview{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAuthenticationReviews) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(authenticationreviewsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.AuthenticationReviewList{})
	return err
}

// Patch applies the patch and returns the patched authenticationReview.
func (c *FakeAuthenticationReviews) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.AuthenticationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(authenticationreviewsResource, name, pt, data, subresources...), &v3.AuthenticationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthenticationReview), err
}
