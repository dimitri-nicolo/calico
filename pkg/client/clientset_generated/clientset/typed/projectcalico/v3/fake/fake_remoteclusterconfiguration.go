// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRemoteClusterConfigurations implements RemoteClusterConfigurationInterface
type FakeRemoteClusterConfigurations struct {
	Fake *FakeProjectcalicoV3
}

var remoteclusterconfigurationsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "remoteclusterconfigurations"}

var remoteclusterconfigurationsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "RemoteClusterConfiguration"}

// Get takes name of the remoteClusterConfiguration, and returns the corresponding remoteClusterConfiguration object, and an error if there is any.
func (c *FakeRemoteClusterConfigurations) Get(name string, options v1.GetOptions) (result *v3.RemoteClusterConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(remoteclusterconfigurationsResource, name), &v3.RemoteClusterConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.RemoteClusterConfiguration), err
}

// List takes label and field selectors, and returns the list of RemoteClusterConfigurations that match those selectors.
func (c *FakeRemoteClusterConfigurations) List(opts v1.ListOptions) (result *v3.RemoteClusterConfigurationList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(remoteclusterconfigurationsResource, remoteclusterconfigurationsKind, opts), &v3.RemoteClusterConfigurationList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.RemoteClusterConfigurationList{ListMeta: obj.(*v3.RemoteClusterConfigurationList).ListMeta}
	for _, item := range obj.(*v3.RemoteClusterConfigurationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested remoteClusterConfigurations.
func (c *FakeRemoteClusterConfigurations) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(remoteclusterconfigurationsResource, opts))
}

// Create takes the representation of a remoteClusterConfiguration and creates it.  Returns the server's representation of the remoteClusterConfiguration, and an error, if there is any.
func (c *FakeRemoteClusterConfigurations) Create(remoteClusterConfiguration *v3.RemoteClusterConfiguration) (result *v3.RemoteClusterConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(remoteclusterconfigurationsResource, remoteClusterConfiguration), &v3.RemoteClusterConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.RemoteClusterConfiguration), err
}

// Update takes the representation of a remoteClusterConfiguration and updates it. Returns the server's representation of the remoteClusterConfiguration, and an error, if there is any.
func (c *FakeRemoteClusterConfigurations) Update(remoteClusterConfiguration *v3.RemoteClusterConfiguration) (result *v3.RemoteClusterConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(remoteclusterconfigurationsResource, remoteClusterConfiguration), &v3.RemoteClusterConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.RemoteClusterConfiguration), err
}

// Delete takes name of the remoteClusterConfiguration and deletes it. Returns an error if one occurs.
func (c *FakeRemoteClusterConfigurations) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(remoteclusterconfigurationsResource, name), &v3.RemoteClusterConfiguration{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRemoteClusterConfigurations) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(remoteclusterconfigurationsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v3.RemoteClusterConfigurationList{})
	return err
}

// Patch applies the patch and returns the patched remoteClusterConfiguration.
func (c *FakeRemoteClusterConfigurations) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.RemoteClusterConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(remoteclusterconfigurationsResource, name, pt, data, subresources...), &v3.RemoteClusterConfiguration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.RemoteClusterConfiguration), err
}
