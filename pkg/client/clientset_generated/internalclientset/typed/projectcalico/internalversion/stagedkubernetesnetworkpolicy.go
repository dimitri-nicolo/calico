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

package internalversion

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"

	projectcalico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
	scheme "github.com/projectcalico/apiserver/pkg/client/clientset_generated/internalclientset/scheme"
)

// StagedKubernetesNetworkPoliciesGetter has a method to return a StagedKubernetesNetworkPolicyInterface.
// A group's client should implement this interface.
type StagedKubernetesNetworkPoliciesGetter interface {
	StagedKubernetesNetworkPolicies(namespace string) StagedKubernetesNetworkPolicyInterface
}

// StagedKubernetesNetworkPolicyInterface has methods to work with StagedKubernetesNetworkPolicy resources.
type StagedKubernetesNetworkPolicyInterface interface {
	Create(ctx context.Context, stagedKubernetesNetworkPolicy *projectcalico.StagedKubernetesNetworkPolicy, opts v1.CreateOptions) (*projectcalico.StagedKubernetesNetworkPolicy, error)
	Update(ctx context.Context, stagedKubernetesNetworkPolicy *projectcalico.StagedKubernetesNetworkPolicy, opts v1.UpdateOptions) (*projectcalico.StagedKubernetesNetworkPolicy, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*projectcalico.StagedKubernetesNetworkPolicy, error)
	List(ctx context.Context, opts v1.ListOptions) (*projectcalico.StagedKubernetesNetworkPolicyList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.StagedKubernetesNetworkPolicy, err error)
	StagedKubernetesNetworkPolicyExpansion
}

// stagedKubernetesNetworkPolicies implements StagedKubernetesNetworkPolicyInterface
type stagedKubernetesNetworkPolicies struct {
	client rest.Interface
	ns     string
}

// newStagedKubernetesNetworkPolicies returns a StagedKubernetesNetworkPolicies
func newStagedKubernetesNetworkPolicies(c *ProjectcalicoClient, namespace string) *stagedKubernetesNetworkPolicies {
	return &stagedKubernetesNetworkPolicies{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the stagedKubernetesNetworkPolicy, and returns the corresponding stagedKubernetesNetworkPolicy object, and an error if there is any.
func (c *stagedKubernetesNetworkPolicies) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.StagedKubernetesNetworkPolicy, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicy{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of StagedKubernetesNetworkPolicies that match those selectors.
func (c *stagedKubernetesNetworkPolicies) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.StagedKubernetesNetworkPolicyList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &projectcalico.StagedKubernetesNetworkPolicyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested stagedKubernetesNetworkPolicies.
func (c *stagedKubernetesNetworkPolicies) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a stagedKubernetesNetworkPolicy and creates it.  Returns the server's representation of the stagedKubernetesNetworkPolicy, and an error, if there is any.
func (c *stagedKubernetesNetworkPolicies) Create(ctx context.Context, stagedKubernetesNetworkPolicy *projectcalico.StagedKubernetesNetworkPolicy, opts v1.CreateOptions) (result *projectcalico.StagedKubernetesNetworkPolicy, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicy{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(stagedKubernetesNetworkPolicy).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a stagedKubernetesNetworkPolicy and updates it. Returns the server's representation of the stagedKubernetesNetworkPolicy, and an error, if there is any.
func (c *stagedKubernetesNetworkPolicies) Update(ctx context.Context, stagedKubernetesNetworkPolicy *projectcalico.StagedKubernetesNetworkPolicy, opts v1.UpdateOptions) (result *projectcalico.StagedKubernetesNetworkPolicy, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicy{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		Name(stagedKubernetesNetworkPolicy.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(stagedKubernetesNetworkPolicy).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the stagedKubernetesNetworkPolicy and deletes it. Returns an error if one occurs.
func (c *stagedKubernetesNetworkPolicies) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *stagedKubernetesNetworkPolicies) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched stagedKubernetesNetworkPolicy.
func (c *stagedKubernetesNetworkPolicies) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.StagedKubernetesNetworkPolicy, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicy{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
