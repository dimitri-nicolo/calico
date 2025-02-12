// Copyright (c) 2025 Tigera, Inc. All rights reserved.

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

// FakeAuthenticationReviews implements AuthenticationReviewInterface
type FakeAuthenticationReviews struct {
	Fake *FakeProjectcalicoV3
}

var authenticationreviewsResource = v3.SchemeGroupVersion.WithResource("authenticationreviews")

var authenticationreviewsKind = v3.SchemeGroupVersion.WithKind("AuthenticationReview")

// Get takes name of the authenticationReview, and returns the corresponding authenticationReview object, and an error if there is any.
func (c *FakeAuthenticationReviews) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.AuthenticationReview, err error) {
	emptyResult := &v3.AuthenticationReview{}
	obj, err := c.Fake.
		Invokes(testing.NewRootGetActionWithOptions(authenticationreviewsResource, name, options), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.AuthenticationReview), err
}

// List takes label and field selectors, and returns the list of AuthenticationReviews that match those selectors.
func (c *FakeAuthenticationReviews) List(ctx context.Context, opts v1.ListOptions) (result *v3.AuthenticationReviewList, err error) {
	emptyResult := &v3.AuthenticationReviewList{}
	obj, err := c.Fake.
		Invokes(testing.NewRootListActionWithOptions(authenticationreviewsResource, authenticationreviewsKind, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
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
		InvokesWatch(testing.NewRootWatchActionWithOptions(authenticationreviewsResource, opts))
}

// Create takes the representation of a authenticationReview and creates it.  Returns the server's representation of the authenticationReview, and an error, if there is any.
func (c *FakeAuthenticationReviews) Create(ctx context.Context, authenticationReview *v3.AuthenticationReview, opts v1.CreateOptions) (result *v3.AuthenticationReview, err error) {
	emptyResult := &v3.AuthenticationReview{}
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateActionWithOptions(authenticationreviewsResource, authenticationReview, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.AuthenticationReview), err
}

// Update takes the representation of a authenticationReview and updates it. Returns the server's representation of the authenticationReview, and an error, if there is any.
func (c *FakeAuthenticationReviews) Update(ctx context.Context, authenticationReview *v3.AuthenticationReview, opts v1.UpdateOptions) (result *v3.AuthenticationReview, err error) {
	emptyResult := &v3.AuthenticationReview{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateActionWithOptions(authenticationreviewsResource, authenticationReview, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.AuthenticationReview), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAuthenticationReviews) UpdateStatus(ctx context.Context, authenticationReview *v3.AuthenticationReview, opts v1.UpdateOptions) (result *v3.AuthenticationReview, err error) {
	emptyResult := &v3.AuthenticationReview{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceActionWithOptions(authenticationreviewsResource, "status", authenticationReview, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.AuthenticationReview), err
}

// Delete takes name of the authenticationReview and deletes it. Returns an error if one occurs.
func (c *FakeAuthenticationReviews) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(authenticationreviewsResource, name, opts), &v3.AuthenticationReview{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAuthenticationReviews) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionActionWithOptions(authenticationreviewsResource, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v3.AuthenticationReviewList{})
	return err
}

// Patch applies the patch and returns the patched authenticationReview.
func (c *FakeAuthenticationReviews) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.AuthenticationReview, err error) {
	emptyResult := &v3.AuthenticationReview{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(authenticationreviewsResource, name, pt, data, opts, subresources...), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.AuthenticationReview), err
}
