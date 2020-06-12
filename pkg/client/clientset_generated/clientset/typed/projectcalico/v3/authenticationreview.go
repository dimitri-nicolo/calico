// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package v3

import (
	"time"

	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	scheme "github.com/tigera/apiserver/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// AuthenticationReviewsGetter has a method to return a AuthenticationReviewInterface.
// A group's client should implement this interface.
type AuthenticationReviewsGetter interface {
	AuthenticationReviews() AuthenticationReviewInterface
}

// AuthenticationReviewInterface has methods to work with AuthenticationReview resources.
type AuthenticationReviewInterface interface {
	Create(*v3.AuthenticationReview) (*v3.AuthenticationReview, error)
	Update(*v3.AuthenticationReview) (*v3.AuthenticationReview, error)
	UpdateStatus(*v3.AuthenticationReview) (*v3.AuthenticationReview, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v3.AuthenticationReview, error)
	List(opts v1.ListOptions) (*v3.AuthenticationReviewList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.AuthenticationReview, err error)
	AuthenticationReviewExpansion
}

// authenticationReviews implements AuthenticationReviewInterface
type authenticationReviews struct {
	client rest.Interface
}

// newAuthenticationReviews returns a AuthenticationReviews
func newAuthenticationReviews(c *ProjectcalicoV3Client) *authenticationReviews {
	return &authenticationReviews{
		client: c.RESTClient(),
	}
}

// Get takes name of the authenticationReview, and returns the corresponding authenticationReview object, and an error if there is any.
func (c *authenticationReviews) Get(name string, options v1.GetOptions) (result *v3.AuthenticationReview, err error) {
	result = &v3.AuthenticationReview{}
	err = c.client.Get().
		Resource("authenticationreviews").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of AuthenticationReviews that match those selectors.
func (c *authenticationReviews) List(opts v1.ListOptions) (result *v3.AuthenticationReviewList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.AuthenticationReviewList{}
	err = c.client.Get().
		Resource("authenticationreviews").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested authenticationReviews.
func (c *authenticationReviews) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("authenticationreviews").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a authenticationReview and creates it.  Returns the server's representation of the authenticationReview, and an error, if there is any.
func (c *authenticationReviews) Create(authenticationReview *v3.AuthenticationReview) (result *v3.AuthenticationReview, err error) {
	result = &v3.AuthenticationReview{}
	err = c.client.Post().
		Resource("authenticationreviews").
		Body(authenticationReview).
		Do().
		Into(result)
	return
}

// Update takes the representation of a authenticationReview and updates it. Returns the server's representation of the authenticationReview, and an error, if there is any.
func (c *authenticationReviews) Update(authenticationReview *v3.AuthenticationReview) (result *v3.AuthenticationReview, err error) {
	result = &v3.AuthenticationReview{}
	err = c.client.Put().
		Resource("authenticationreviews").
		Name(authenticationReview.Name).
		Body(authenticationReview).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *authenticationReviews) UpdateStatus(authenticationReview *v3.AuthenticationReview) (result *v3.AuthenticationReview, err error) {
	result = &v3.AuthenticationReview{}
	err = c.client.Put().
		Resource("authenticationreviews").
		Name(authenticationReview.Name).
		SubResource("status").
		Body(authenticationReview).
		Do().
		Into(result)
	return
}

// Delete takes name of the authenticationReview and deletes it. Returns an error if one occurs.
func (c *authenticationReviews) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("authenticationreviews").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *authenticationReviews) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("authenticationreviews").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched authenticationReview.
func (c *authenticationReviews) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.AuthenticationReview, err error) {
	result = &v3.AuthenticationReview{}
	err = c.client.Patch(pt).
		Resource("authenticationreviews").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
