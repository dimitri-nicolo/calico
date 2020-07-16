// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package internalversion

import (
	"time"

	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	scheme "github.com/tigera/apiserver/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// AuthorizationReviewsGetter has a method to return a AuthorizationReviewInterface.
// A group's client should implement this interface.
type AuthorizationReviewsGetter interface {
	AuthorizationReviews() AuthorizationReviewInterface
}

// AuthorizationReviewInterface has methods to work with AuthorizationReview resources.
type AuthorizationReviewInterface interface {
	Create(*projectcalico.AuthorizationReview) (*projectcalico.AuthorizationReview, error)
	Update(*projectcalico.AuthorizationReview) (*projectcalico.AuthorizationReview, error)
	UpdateStatus(*projectcalico.AuthorizationReview) (*projectcalico.AuthorizationReview, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*projectcalico.AuthorizationReview, error)
	List(opts v1.ListOptions) (*projectcalico.AuthorizationReviewList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.AuthorizationReview, err error)
	AuthorizationReviewExpansion
}

// authorizationReviews implements AuthorizationReviewInterface
type authorizationReviews struct {
	client rest.Interface
}

// newAuthorizationReviews returns a AuthorizationReviews
func newAuthorizationReviews(c *ProjectcalicoClient) *authorizationReviews {
	return &authorizationReviews{
		client: c.RESTClient(),
	}
}

// Get takes name of the authorizationReview, and returns the corresponding authorizationReview object, and an error if there is any.
func (c *authorizationReviews) Get(name string, options v1.GetOptions) (result *projectcalico.AuthorizationReview, err error) {
	result = &projectcalico.AuthorizationReview{}
	err = c.client.Get().
		Resource("authorizationreviews").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of AuthorizationReviews that match those selectors.
func (c *authorizationReviews) List(opts v1.ListOptions) (result *projectcalico.AuthorizationReviewList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &projectcalico.AuthorizationReviewList{}
	err = c.client.Get().
		Resource("authorizationreviews").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested authorizationReviews.
func (c *authorizationReviews) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("authorizationreviews").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a authorizationReview and creates it.  Returns the server's representation of the authorizationReview, and an error, if there is any.
func (c *authorizationReviews) Create(authorizationReview *projectcalico.AuthorizationReview) (result *projectcalico.AuthorizationReview, err error) {
	result = &projectcalico.AuthorizationReview{}
	err = c.client.Post().
		Resource("authorizationreviews").
		Body(authorizationReview).
		Do().
		Into(result)
	return
}

// Update takes the representation of a authorizationReview and updates it. Returns the server's representation of the authorizationReview, and an error, if there is any.
func (c *authorizationReviews) Update(authorizationReview *projectcalico.AuthorizationReview) (result *projectcalico.AuthorizationReview, err error) {
	result = &projectcalico.AuthorizationReview{}
	err = c.client.Put().
		Resource("authorizationreviews").
		Name(authorizationReview.Name).
		Body(authorizationReview).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *authorizationReviews) UpdateStatus(authorizationReview *projectcalico.AuthorizationReview) (result *projectcalico.AuthorizationReview, err error) {
	result = &projectcalico.AuthorizationReview{}
	err = c.client.Put().
		Resource("authorizationreviews").
		Name(authorizationReview.Name).
		SubResource("status").
		Body(authorizationReview).
		Do().
		Into(result)
	return
}

// Delete takes name of the authorizationReview and deletes it. Returns an error if one occurs.
func (c *authorizationReviews) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("authorizationreviews").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *authorizationReviews) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("authorizationreviews").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched authorizationReview.
func (c *authorizationReviews) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.AuthorizationReview, err error) {
	result = &projectcalico.AuthorizationReview{}
	err = c.client.Patch(pt).
		Resource("authorizationreviews").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
