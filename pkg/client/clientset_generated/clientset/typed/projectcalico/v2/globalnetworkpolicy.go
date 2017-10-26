/*
Copyright 2017 Tigera.
*/package v2

import (
	v2 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v2"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GlobalNetworkPoliciesGetter has a method to return a GlobalNetworkPolicyInterface.
// A group's client should implement this interface.
type GlobalNetworkPoliciesGetter interface {
	GlobalNetworkPolicies() GlobalNetworkPolicyInterface
}

// GlobalNetworkPolicyInterface has methods to work with GlobalNetworkPolicy resources.
type GlobalNetworkPolicyInterface interface {
	Create(*v2.GlobalNetworkPolicy) (*v2.GlobalNetworkPolicy, error)
	Update(*v2.GlobalNetworkPolicy) (*v2.GlobalNetworkPolicy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v2.GlobalNetworkPolicy, error)
	List(opts v1.ListOptions) (*v2.GlobalNetworkPolicyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v2.GlobalNetworkPolicy, err error)
	GlobalNetworkPolicyExpansion
}

// globalNetworkPolicies implements GlobalNetworkPolicyInterface
type globalNetworkPolicies struct {
	client rest.Interface
}

// newGlobalNetworkPolicies returns a GlobalNetworkPolicies
func newGlobalNetworkPolicies(c *ProjectcalicoV2Client) *globalNetworkPolicies {
	return &globalNetworkPolicies{
		client: c.RESTClient(),
	}
}

// Get takes name of the globalNetworkPolicy, and returns the corresponding globalNetworkPolicy object, and an error if there is any.
func (c *globalNetworkPolicies) Get(name string, options v1.GetOptions) (result *v2.GlobalNetworkPolicy, err error) {
	result = &v2.GlobalNetworkPolicy{}
	err = c.client.Get().
		Resource("globalnetworkpolicies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalNetworkPolicies that match those selectors.
func (c *globalNetworkPolicies) List(opts v1.ListOptions) (result *v2.GlobalNetworkPolicyList, err error) {
	result = &v2.GlobalNetworkPolicyList{}
	err = c.client.Get().
		Resource("globalnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalNetworkPolicies.
func (c *globalNetworkPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("globalnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a globalNetworkPolicy and creates it.  Returns the server's representation of the globalNetworkPolicy, and an error, if there is any.
func (c *globalNetworkPolicies) Create(globalNetworkPolicy *v2.GlobalNetworkPolicy) (result *v2.GlobalNetworkPolicy, err error) {
	result = &v2.GlobalNetworkPolicy{}
	err = c.client.Post().
		Resource("globalnetworkpolicies").
		Body(globalNetworkPolicy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a globalNetworkPolicy and updates it. Returns the server's representation of the globalNetworkPolicy, and an error, if there is any.
func (c *globalNetworkPolicies) Update(globalNetworkPolicy *v2.GlobalNetworkPolicy) (result *v2.GlobalNetworkPolicy, err error) {
	result = &v2.GlobalNetworkPolicy{}
	err = c.client.Put().
		Resource("globalnetworkpolicies").
		Name(globalNetworkPolicy.Name).
		Body(globalNetworkPolicy).
		Do().
		Into(result)
	return
}

// Delete takes name of the globalNetworkPolicy and deletes it. Returns an error if one occurs.
func (c *globalNetworkPolicies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalnetworkpolicies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalNetworkPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("globalnetworkpolicies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched globalNetworkPolicy.
func (c *globalNetworkPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v2.GlobalNetworkPolicy, err error) {
	result = &v2.GlobalNetworkPolicy{}
	err = c.client.Patch(pt).
		Resource("globalnetworkpolicies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
