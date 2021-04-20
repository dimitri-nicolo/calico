// Copyright (c) 2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"

	projectcalico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// FakeAuthorizationReviews implements AuthorizationReviewInterface
type FakeAuthorizationReviews struct {
	Fake *FakeProjectcalico
}

var authorizationreviewsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "authorizationreviews"}

var authorizationreviewsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "AuthorizationReview"}

// Get takes name of the authorizationReview, and returns the corresponding authorizationReview object, and an error if there is any.
func (c *FakeAuthorizationReviews) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.AuthorizationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(authorizationreviewsResource, name), &projectcalico.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.AuthorizationReview), err
}

// List takes label and field selectors, and returns the list of AuthorizationReviews that match those selectors.
func (c *FakeAuthorizationReviews) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.AuthorizationReviewList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(authorizationreviewsResource, authorizationreviewsKind, opts), &projectcalico.AuthorizationReviewList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.AuthorizationReviewList{ListMeta: obj.(*projectcalico.AuthorizationReviewList).ListMeta}
	for _, item := range obj.(*projectcalico.AuthorizationReviewList).Items {
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
func (c *FakeAuthorizationReviews) Create(ctx context.Context, authorizationReview *projectcalico.AuthorizationReview, opts v1.CreateOptions) (result *projectcalico.AuthorizationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(authorizationreviewsResource, authorizationReview), &projectcalico.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.AuthorizationReview), err
}

// Update takes the representation of a authorizationReview and updates it. Returns the server's representation of the authorizationReview, and an error, if there is any.
func (c *FakeAuthorizationReviews) Update(ctx context.Context, authorizationReview *projectcalico.AuthorizationReview, opts v1.UpdateOptions) (result *projectcalico.AuthorizationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(authorizationreviewsResource, authorizationReview), &projectcalico.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.AuthorizationReview), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAuthorizationReviews) UpdateStatus(ctx context.Context, authorizationReview *projectcalico.AuthorizationReview, opts v1.UpdateOptions) (*projectcalico.AuthorizationReview, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(authorizationreviewsResource, "status", authorizationReview), &projectcalico.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.AuthorizationReview), err
}

// Delete takes name of the authorizationReview and deletes it. Returns an error if one occurs.
func (c *FakeAuthorizationReviews) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(authorizationreviewsResource, name), &projectcalico.AuthorizationReview{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAuthorizationReviews) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(authorizationreviewsResource, listOpts)

	_, err := c.Fake.Invokes(action, &projectcalico.AuthorizationReviewList{})
	return err
}

// Patch applies the patch and returns the patched authorizationReview.
func (c *FakeAuthorizationReviews) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.AuthorizationReview, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(authorizationreviewsResource, name, pt, data, subresources...), &projectcalico.AuthorizationReview{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.AuthorizationReview), err
}
