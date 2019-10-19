// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package internalversion

import (
	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// StagedKubernetesNetworkPoliciesGetter has a method to return a StagedKubernetesNetworkPolicyInterface.
// A group's client should implement this interface.
type StagedKubernetesNetworkPoliciesGetter interface {
	StagedKubernetesNetworkPolicies(namespace string) StagedKubernetesNetworkPolicyInterface
}

// StagedKubernetesNetworkPolicyInterface has methods to work with StagedKubernetesNetworkPolicy resources.
type StagedKubernetesNetworkPolicyInterface interface {
	Create(*projectcalico.StagedKubernetesNetworkPolicy) (*projectcalico.StagedKubernetesNetworkPolicy, error)
	Update(*projectcalico.StagedKubernetesNetworkPolicy) (*projectcalico.StagedKubernetesNetworkPolicy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*projectcalico.StagedKubernetesNetworkPolicy, error)
	List(opts v1.ListOptions) (*projectcalico.StagedKubernetesNetworkPolicyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.StagedKubernetesNetworkPolicy, err error)
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
func (c *stagedKubernetesNetworkPolicies) Get(name string, options v1.GetOptions) (result *projectcalico.StagedKubernetesNetworkPolicy, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicy{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of StagedKubernetesNetworkPolicies that match those selectors.
func (c *stagedKubernetesNetworkPolicies) List(opts v1.ListOptions) (result *projectcalico.StagedKubernetesNetworkPolicyList, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested stagedKubernetesNetworkPolicies.
func (c *stagedKubernetesNetworkPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a stagedKubernetesNetworkPolicy and creates it.  Returns the server's representation of the stagedKubernetesNetworkPolicy, and an error, if there is any.
func (c *stagedKubernetesNetworkPolicies) Create(stagedKubernetesNetworkPolicy *projectcalico.StagedKubernetesNetworkPolicy) (result *projectcalico.StagedKubernetesNetworkPolicy, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicy{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		Body(stagedKubernetesNetworkPolicy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a stagedKubernetesNetworkPolicy and updates it. Returns the server's representation of the stagedKubernetesNetworkPolicy, and an error, if there is any.
func (c *stagedKubernetesNetworkPolicies) Update(stagedKubernetesNetworkPolicy *projectcalico.StagedKubernetesNetworkPolicy) (result *projectcalico.StagedKubernetesNetworkPolicy, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicy{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		Name(stagedKubernetesNetworkPolicy.Name).
		Body(stagedKubernetesNetworkPolicy).
		Do().
		Into(result)
	return
}

// Delete takes name of the stagedKubernetesNetworkPolicy and deletes it. Returns an error if one occurs.
func (c *stagedKubernetesNetworkPolicies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *stagedKubernetesNetworkPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched stagedKubernetesNetworkPolicy.
func (c *stagedKubernetesNetworkPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.StagedKubernetesNetworkPolicy, err error) {
	result = &projectcalico.StagedKubernetesNetworkPolicy{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("stagedkubernetesnetworkpolicies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
