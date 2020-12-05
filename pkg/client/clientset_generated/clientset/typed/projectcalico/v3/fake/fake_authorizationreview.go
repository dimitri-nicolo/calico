// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAuthorizationReviews implements AuthorizationReviewInterface
type FakeAuthorizationReviews struct {
	Fake *FakeProjectcalicoV3
}

var authorizationreviewsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "authorizationreviews"}

var authorizationreviewsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "AuthorizationReview"}

// Get takes name of the authorizationReview, and returns the corresponding authorizationReview object, and an error if there is any.
func (c *FakeAuthorizationReviews) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.AuthorizationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(authorizationreviewsResource, name), &v3.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthorizationReview), err
}

// List takes label and field selectors, and returns the list of AuthorizationReviews that match those selectors.
func (c *FakeAuthorizationReviews) List(ctx context.Context, opts v1.ListOptions) (result *v3.AuthorizationReviewList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(authorizationreviewsResource, authorizationreviewsKind, opts), &v3.AuthorizationReviewList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.AuthorizationReviewList{ListMeta: obj.(*v3.AuthorizationReviewList).ListMeta}
	for _, item := range obj.(*v3.AuthorizationReviewList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested authorizationReviews.
func (c *FakeAuthorizationReviews) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(authorizationreviewsResource, opts))
}

// Create takes the representation of a authorizationReview and creates it.  Returns the server's representation of the authorizationReview, and an error, if there is any.
func (c *FakeAuthorizationReviews) Create(ctx context.Context, authorizationReview *v3.AuthorizationReview, opts v1.CreateOptions) (result *v3.AuthorizationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(authorizationreviewsResource, authorizationReview), &v3.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthorizationReview), err
}

// Update takes the representation of a authorizationReview and updates it. Returns the server's representation of the authorizationReview, and an error, if there is any.
func (c *FakeAuthorizationReviews) Update(ctx context.Context, authorizationReview *v3.AuthorizationReview, opts v1.UpdateOptions) (result *v3.AuthorizationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(authorizationreviewsResource, authorizationReview), &v3.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthorizationReview), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAuthorizationReviews) UpdateStatus(ctx context.Context, authorizationReview *v3.AuthorizationReview, opts v1.UpdateOptions) (*v3.AuthorizationReview, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(authorizationreviewsResource, "status", authorizationReview), &v3.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthorizationReview), err
}

// Delete takes name of the authorizationReview and deletes it. Returns an error if one occurs.
func (c *FakeAuthorizationReviews) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(authorizationreviewsResource, name), &v3.AuthorizationReview{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAuthorizationReviews) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(authorizationreviewsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.AuthorizationReviewList{})
	return err
}

// Patch applies the patch and returns the patched authorizationReview.
func (c *FakeAuthorizationReviews) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.AuthorizationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(authorizationreviewsResource, name, pt, data, subresources...), &v3.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthorizationReview), err
}
