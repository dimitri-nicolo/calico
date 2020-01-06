// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package v3

import (
	"time"

	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ManagedClustersGetter has a method to return a ManagedClusterInterface.
// A group's client should implement this interface.
type ManagedClustersGetter interface {
	ManagedClusters() ManagedClusterInterface
}

// ManagedClusterInterface has methods to work with ManagedCluster resources.
type ManagedClusterInterface interface {
	Create(*v3.ManagedCluster) (*v3.ManagedCluster, error)
	Update(*v3.ManagedCluster) (*v3.ManagedCluster, error)
	UpdateStatus(*v3.ManagedCluster) (*v3.ManagedCluster, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v3.ManagedCluster, error)
	List(opts v1.ListOptions) (*v3.ManagedClusterList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.ManagedCluster, err error)
	ManagedClusterExpansion
}

// managedClusters implements ManagedClusterInterface
type managedClusters struct {
	client rest.Interface
}

// newManagedClusters returns a ManagedClusters
func newManagedClusters(c *ProjectcalicoV3Client) *managedClusters {
	return &managedClusters{
		client: c.RESTClient(),
	}
}

// Get takes name of the managedCluster, and returns the corresponding managedCluster object, and an error if there is any.
func (c *managedClusters) Get(name string, options v1.GetOptions) (result *v3.ManagedCluster, err error) {
	result = &v3.ManagedCluster{}
	err = c.client.Get().
		Resource("managedclusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ManagedClusters that match those selectors.
func (c *managedClusters) List(opts v1.ListOptions) (result *v3.ManagedClusterList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.ManagedClusterList{}
	err = c.client.Get().
		Resource("managedclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested managedClusters.
func (c *managedClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("managedclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a managedCluster and creates it.  Returns the server's representation of the managedCluster, and an error, if there is any.
func (c *managedClusters) Create(managedCluster *v3.ManagedCluster) (result *v3.ManagedCluster, err error) {
	result = &v3.ManagedCluster{}
	err = c.client.Post().
		Resource("managedclusters").
		Body(managedCluster).
		Do().
		Into(result)
	return
}

// Update takes the representation of a managedCluster and updates it. Returns the server's representation of the managedCluster, and an error, if there is any.
func (c *managedClusters) Update(managedCluster *v3.ManagedCluster) (result *v3.ManagedCluster, err error) {
	result = &v3.ManagedCluster{}
	err = c.client.Put().
		Resource("managedclusters").
		Name(managedCluster.Name).
		Body(managedCluster).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *managedClusters) UpdateStatus(managedCluster *v3.ManagedCluster) (result *v3.ManagedCluster, err error) {
	result = &v3.ManagedCluster{}
	err = c.client.Put().
		Resource("managedclusters").
		Name(managedCluster.Name).
		SubResource("status").
		Body(managedCluster).
		Do().
		Into(result)
	return
}

// Delete takes name of the managedCluster and deletes it. Returns an error if one occurs.
func (c *managedClusters) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("managedclusters").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *managedClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("managedclusters").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched managedCluster.
func (c *managedClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.ManagedCluster, err error) {
	result = &v3.ManagedCluster{}
	err = c.client.Patch(pt).
		Resource("managedclusters").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
