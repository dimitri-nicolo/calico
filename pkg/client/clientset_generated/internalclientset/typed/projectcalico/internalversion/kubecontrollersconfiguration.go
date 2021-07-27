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

// KubeControllersConfigurationsGetter has a method to return a KubeControllersConfigurationInterface.
// A group's client should implement this interface.
type KubeControllersConfigurationsGetter interface {
	KubeControllersConfigurations() KubeControllersConfigurationInterface
}

// KubeControllersConfigurationInterface has methods to work with KubeControllersConfiguration resources.
type KubeControllersConfigurationInterface interface {
	Create(ctx context.Context, kubeControllersConfiguration *projectcalico.KubeControllersConfiguration, opts v1.CreateOptions) (*projectcalico.KubeControllersConfiguration, error)
	Update(ctx context.Context, kubeControllersConfiguration *projectcalico.KubeControllersConfiguration, opts v1.UpdateOptions) (*projectcalico.KubeControllersConfiguration, error)
	UpdateStatus(ctx context.Context, kubeControllersConfiguration *projectcalico.KubeControllersConfiguration, opts v1.UpdateOptions) (*projectcalico.KubeControllersConfiguration, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*projectcalico.KubeControllersConfiguration, error)
	List(ctx context.Context, opts v1.ListOptions) (*projectcalico.KubeControllersConfigurationList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.KubeControllersConfiguration, err error)
	KubeControllersConfigurationExpansion
}

// kubeControllersConfigurations implements KubeControllersConfigurationInterface
type kubeControllersConfigurations struct {
	client rest.Interface
}

// newKubeControllersConfigurations returns a KubeControllersConfigurations
func newKubeControllersConfigurations(c *ProjectcalicoClient) *kubeControllersConfigurations {
	return &kubeControllersConfigurations{
		client: c.RESTClient(),
	}
}

// Get takes name of the kubeControllersConfiguration, and returns the corresponding kubeControllersConfiguration object, and an error if there is any.
func (c *kubeControllersConfigurations) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.KubeControllersConfiguration, err error) {
	result = &projectcalico.KubeControllersConfiguration{}
	err = c.client.Get().
		Resource("kubecontrollersconfigurations").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of KubeControllersConfigurations that match those selectors.
func (c *kubeControllersConfigurations) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.KubeControllersConfigurationList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &projectcalico.KubeControllersConfigurationList{}
	err = c.client.Get().
		Resource("kubecontrollersconfigurations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested kubeControllersConfigurations.
func (c *kubeControllersConfigurations) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("kubecontrollersconfigurations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a kubeControllersConfiguration and creates it.  Returns the server's representation of the kubeControllersConfiguration, and an error, if there is any.
func (c *kubeControllersConfigurations) Create(ctx context.Context, kubeControllersConfiguration *projectcalico.KubeControllersConfiguration, opts v1.CreateOptions) (result *projectcalico.KubeControllersConfiguration, err error) {
	result = &projectcalico.KubeControllersConfiguration{}
	err = c.client.Post().
		Resource("kubecontrollersconfigurations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(kubeControllersConfiguration).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a kubeControllersConfiguration and updates it. Returns the server's representation of the kubeControllersConfiguration, and an error, if there is any.
func (c *kubeControllersConfigurations) Update(ctx context.Context, kubeControllersConfiguration *projectcalico.KubeControllersConfiguration, opts v1.UpdateOptions) (result *projectcalico.KubeControllersConfiguration, err error) {
	result = &projectcalico.KubeControllersConfiguration{}
	err = c.client.Put().
		Resource("kubecontrollersconfigurations").
		Name(kubeControllersConfiguration.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(kubeControllersConfiguration).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *kubeControllersConfigurations) UpdateStatus(ctx context.Context, kubeControllersConfiguration *projectcalico.KubeControllersConfiguration, opts v1.UpdateOptions) (result *projectcalico.KubeControllersConfiguration, err error) {
	result = &projectcalico.KubeControllersConfiguration{}
	err = c.client.Put().
		Resource("kubecontrollersconfigurations").
		Name(kubeControllersConfiguration.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(kubeControllersConfiguration).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the kubeControllersConfiguration and deletes it. Returns an error if one occurs.
func (c *kubeControllersConfigurations) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("kubecontrollersconfigurations").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *kubeControllersConfigurations) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("kubecontrollersconfigurations").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched kubeControllersConfiguration.
func (c *kubeControllersConfigurations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.KubeControllersConfiguration, err error) {
	result = &projectcalico.KubeControllersConfiguration{}
	err = c.client.Patch(pt).
		Resource("kubecontrollersconfigurations").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
