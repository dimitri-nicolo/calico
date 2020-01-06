// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package internalversion

import (
	"time"

	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GlobalAlertsGetter has a method to return a GlobalAlertInterface.
// A group's client should implement this interface.
type GlobalAlertsGetter interface {
	GlobalAlerts() GlobalAlertInterface
}

// GlobalAlertInterface has methods to work with GlobalAlert resources.
type GlobalAlertInterface interface {
	Create(*projectcalico.GlobalAlert) (*projectcalico.GlobalAlert, error)
	Update(*projectcalico.GlobalAlert) (*projectcalico.GlobalAlert, error)
	UpdateStatus(*projectcalico.GlobalAlert) (*projectcalico.GlobalAlert, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*projectcalico.GlobalAlert, error)
	List(opts v1.ListOptions) (*projectcalico.GlobalAlertList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.GlobalAlert, err error)
	GlobalAlertExpansion
}

// globalAlerts implements GlobalAlertInterface
type globalAlerts struct {
	client rest.Interface
}

// newGlobalAlerts returns a GlobalAlerts
func newGlobalAlerts(c *ProjectcalicoClient) *globalAlerts {
	return &globalAlerts{
		client: c.RESTClient(),
	}
}

// Get takes name of the globalAlert, and returns the corresponding globalAlert object, and an error if there is any.
func (c *globalAlerts) Get(name string, options v1.GetOptions) (result *projectcalico.GlobalAlert, err error) {
	result = &projectcalico.GlobalAlert{}
	err = c.client.Get().
		Resource("globalalerts").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalAlerts that match those selectors.
func (c *globalAlerts) List(opts v1.ListOptions) (result *projectcalico.GlobalAlertList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &projectcalico.GlobalAlertList{}
	err = c.client.Get().
		Resource("globalalerts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalAlerts.
func (c *globalAlerts) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("globalalerts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a globalAlert and creates it.  Returns the server's representation of the globalAlert, and an error, if there is any.
func (c *globalAlerts) Create(globalAlert *projectcalico.GlobalAlert) (result *projectcalico.GlobalAlert, err error) {
	result = &projectcalico.GlobalAlert{}
	err = c.client.Post().
		Resource("globalalerts").
		Body(globalAlert).
		Do().
		Into(result)
	return
}

// Update takes the representation of a globalAlert and updates it. Returns the server's representation of the globalAlert, and an error, if there is any.
func (c *globalAlerts) Update(globalAlert *projectcalico.GlobalAlert) (result *projectcalico.GlobalAlert, err error) {
	result = &projectcalico.GlobalAlert{}
	err = c.client.Put().
		Resource("globalalerts").
		Name(globalAlert.Name).
		Body(globalAlert).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *globalAlerts) UpdateStatus(globalAlert *projectcalico.GlobalAlert) (result *projectcalico.GlobalAlert, err error) {
	result = &projectcalico.GlobalAlert{}
	err = c.client.Put().
		Resource("globalalerts").
		Name(globalAlert.Name).
		SubResource("status").
		Body(globalAlert).
		Do().
		Into(result)
	return
}

// Delete takes name of the globalAlert and deletes it. Returns an error if one occurs.
func (c *globalAlerts) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalalerts").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalAlerts) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("globalalerts").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched globalAlert.
func (c *globalAlerts) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.GlobalAlert, err error) {
	result = &projectcalico.GlobalAlert{}
	err = c.client.Patch(pt).
		Resource("globalalerts").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
