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

// GlobalAlertTemplatesGetter has a method to return a GlobalAlertTemplateInterface.
// A group's client should implement this interface.
type GlobalAlertTemplatesGetter interface {
	GlobalAlertTemplates() GlobalAlertTemplateInterface
}

// GlobalAlertTemplateInterface has methods to work with GlobalAlertTemplate resources.
type GlobalAlertTemplateInterface interface {
	Create(ctx context.Context, globalAlertTemplate *projectcalico.GlobalAlertTemplate, opts v1.CreateOptions) (*projectcalico.GlobalAlertTemplate, error)
	Update(ctx context.Context, globalAlertTemplate *projectcalico.GlobalAlertTemplate, opts v1.UpdateOptions) (*projectcalico.GlobalAlertTemplate, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*projectcalico.GlobalAlertTemplate, error)
	List(ctx context.Context, opts v1.ListOptions) (*projectcalico.GlobalAlertTemplateList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.GlobalAlertTemplate, err error)
	GlobalAlertTemplateExpansion
}

// globalAlertTemplates implements GlobalAlertTemplateInterface
type globalAlertTemplates struct {
	client rest.Interface
}

// newGlobalAlertTemplates returns a GlobalAlertTemplates
func newGlobalAlertTemplates(c *ProjectcalicoClient) *globalAlertTemplates {
	return &globalAlertTemplates{
		client: c.RESTClient(),
	}
}

// Get takes name of the globalAlertTemplate, and returns the corresponding globalAlertTemplate object, and an error if there is any.
func (c *globalAlertTemplates) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.GlobalAlertTemplate, err error) {
	result = &projectcalico.GlobalAlertTemplate{}
	err = c.client.Get().
		Resource("globalalerttemplates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalAlertTemplates that match those selectors.
func (c *globalAlertTemplates) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.GlobalAlertTemplateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &projectcalico.GlobalAlertTemplateList{}
	err = c.client.Get().
		Resource("globalalerttemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalAlertTemplates.
func (c *globalAlertTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("globalalerttemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a globalAlertTemplate and creates it.  Returns the server's representation of the globalAlertTemplate, and an error, if there is any.
func (c *globalAlertTemplates) Create(ctx context.Context, globalAlertTemplate *projectcalico.GlobalAlertTemplate, opts v1.CreateOptions) (result *projectcalico.GlobalAlertTemplate, err error) {
	result = &projectcalico.GlobalAlertTemplate{}
	err = c.client.Post().
		Resource("globalalerttemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalAlertTemplate).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a globalAlertTemplate and updates it. Returns the server's representation of the globalAlertTemplate, and an error, if there is any.
func (c *globalAlertTemplates) Update(ctx context.Context, globalAlertTemplate *projectcalico.GlobalAlertTemplate, opts v1.UpdateOptions) (result *projectcalico.GlobalAlertTemplate, err error) {
	result = &projectcalico.GlobalAlertTemplate{}
	err = c.client.Put().
		Resource("globalalerttemplates").
		Name(globalAlertTemplate.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalAlertTemplate).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the globalAlertTemplate and deletes it. Returns an error if one occurs.
func (c *globalAlertTemplates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalalerttemplates").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalAlertTemplates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("globalalerttemplates").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched globalAlertTemplate.
func (c *globalAlertTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.GlobalAlertTemplate, err error) {
	result = &projectcalico.GlobalAlertTemplate{}
	err = c.client.Patch(pt).
		Resource("globalalerttemplates").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
