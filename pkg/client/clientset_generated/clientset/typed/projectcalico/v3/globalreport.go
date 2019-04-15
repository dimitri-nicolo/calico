// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GlobalReportsGetter has a method to return a GlobalReportInterface.
// A group's client should implement this interface.
type GlobalReportsGetter interface {
	GlobalReports() GlobalReportInterface
}

// GlobalReportInterface has methods to work with GlobalReport resources.
type GlobalReportInterface interface {
	Create(*v3.GlobalReport) (*v3.GlobalReport, error)
	Update(*v3.GlobalReport) (*v3.GlobalReport, error)
	UpdateStatus(*v3.GlobalReport) (*v3.GlobalReport, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v3.GlobalReport, error)
	List(opts v1.ListOptions) (*v3.GlobalReportList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalReport, err error)
	GlobalReportExpansion
}

// globalReports implements GlobalReportInterface
type globalReports struct {
	client rest.Interface
}

// newGlobalReports returns a GlobalReports
func newGlobalReports(c *ProjectcalicoV3Client) *globalReports {
	return &globalReports{
		client: c.RESTClient(),
	}
}

// Get takes name of the globalReport, and returns the corresponding globalReport object, and an error if there is any.
func (c *globalReports) Get(name string, options v1.GetOptions) (result *v3.GlobalReport, err error) {
	result = &v3.GlobalReport{}
	err = c.client.Get().
		Resource("globalreports").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalReports that match those selectors.
func (c *globalReports) List(opts v1.ListOptions) (result *v3.GlobalReportList, err error) {
	result = &v3.GlobalReportList{}
	err = c.client.Get().
		Resource("globalreports").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalReports.
func (c *globalReports) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("globalreports").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a globalReport and creates it.  Returns the server's representation of the globalReport, and an error, if there is any.
func (c *globalReports) Create(globalReport *v3.GlobalReport) (result *v3.GlobalReport, err error) {
	result = &v3.GlobalReport{}
	err = c.client.Post().
		Resource("globalreports").
		Body(globalReport).
		Do().
		Into(result)
	return
}

// Update takes the representation of a globalReport and updates it. Returns the server's representation of the globalReport, and an error, if there is any.
func (c *globalReports) Update(globalReport *v3.GlobalReport) (result *v3.GlobalReport, err error) {
	result = &v3.GlobalReport{}
	err = c.client.Put().
		Resource("globalreports").
		Name(globalReport.Name).
		Body(globalReport).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *globalReports) UpdateStatus(globalReport *v3.GlobalReport) (result *v3.GlobalReport, err error) {
	result = &v3.GlobalReport{}
	err = c.client.Put().
		Resource("globalreports").
		Name(globalReport.Name).
		SubResource("status").
		Body(globalReport).
		Do().
		Into(result)
	return
}

// Delete takes name of the globalReport and deletes it. Returns an error if one occurs.
func (c *globalReports) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalreports").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalReports) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("globalreports").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched globalReport.
func (c *globalReports) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalReport, err error) {
	result = &v3.GlobalReport{}
	err = c.client.Patch(pt).
		Resource("globalreports").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
