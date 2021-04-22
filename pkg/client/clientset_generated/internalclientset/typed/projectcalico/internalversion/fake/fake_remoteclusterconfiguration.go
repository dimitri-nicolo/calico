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

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"

	projectcalico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// FakeRemoteClusterConfigurations implements RemoteClusterConfigurationInterface
type FakeRemoteClusterConfigurations struct {
	Fake *FakeProjectcalico
}

var remoteclusterconfigurationsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "remoteclusterconfigurations"}

var remoteclusterconfigurationsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "RemoteClusterConfiguration"}

// Get takes name of the remoteClusterConfiguration, and returns the corresponding remoteClusterConfiguration object, and an error if there is any.
func (c *FakeRemoteClusterConfigurations) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.RemoteClusterConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(remoteclusterconfigurationsResource, name), &projectcalico.RemoteClusterConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.RemoteClusterConfiguration), err
}

// List takes label and field selectors, and returns the list of RemoteClusterConfigurations that match those selectors.
func (c *FakeRemoteClusterConfigurations) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.RemoteClusterConfigurationList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(remoteclusterconfigurationsResource, remoteclusterconfigurationsKind, opts), &projectcalico.RemoteClusterConfigurationList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.RemoteClusterConfigurationList{ListMeta: obj.(*projectcalico.RemoteClusterConfigurationList).ListMeta}
	for _, item := range obj.(*projectcalico.RemoteClusterConfigurationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested remoteClusterConfigurations.
func (c *FakeRemoteClusterConfigurations) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(remoteclusterconfigurationsResource, opts))
}

// Create takes the representation of a remoteClusterConfiguration and creates it.  Returns the server's representation of the remoteClusterConfiguration, and an error, if there is any.
func (c *FakeRemoteClusterConfigurations) Create(ctx context.Context, remoteClusterConfiguration *projectcalico.RemoteClusterConfiguration, opts v1.CreateOptions) (result *projectcalico.RemoteClusterConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(remoteclusterconfigurationsResource, remoteClusterConfiguration), &projectcalico.RemoteClusterConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.RemoteClusterConfiguration), err
}

// Update takes the representation of a remoteClusterConfiguration and updates it. Returns the server's representation of the remoteClusterConfiguration, and an error, if there is any.
func (c *FakeRemoteClusterConfigurations) Update(ctx context.Context, remoteClusterConfiguration *projectcalico.RemoteClusterConfiguration, opts v1.UpdateOptions) (result *projectcalico.RemoteClusterConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(remoteclusterconfigurationsResource, remoteClusterConfiguration), &projectcalico.RemoteClusterConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.RemoteClusterConfiguration), err
}

// Delete takes name of the remoteClusterConfiguration and deletes it. Returns an error if one occurs.
func (c *FakeRemoteClusterConfigurations) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(remoteclusterconfigurationsResource, name), &projectcalico.RemoteClusterConfiguration{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRemoteClusterConfigurations) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(remoteclusterconfigurationsResource, listOpts)

	_, err := c.Fake.Invokes(action, &projectcalico.RemoteClusterConfigurationList{})
	return err
}

// Patch applies the patch and returns the patched remoteClusterConfiguration.
func (c *FakeRemoteClusterConfigurations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.RemoteClusterConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(remoteclusterconfigurationsResource, name, pt, data, subresources...), &projectcalico.RemoteClusterConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.RemoteClusterConfiguration), err
}
