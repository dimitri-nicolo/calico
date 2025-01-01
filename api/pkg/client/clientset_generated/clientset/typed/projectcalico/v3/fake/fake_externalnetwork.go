// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeExternalNetworks implements ExternalNetworkInterface
type FakeExternalNetworks struct {
	Fake *FakeProjectcalicoV3
}

var externalnetworksResource = v3.SchemeGroupVersion.WithResource("externalnetworks")

var externalnetworksKind = v3.SchemeGroupVersion.WithKind("ExternalNetwork")

// Get takes name of the externalNetwork, and returns the corresponding externalNetwork object, and an error if there is any.
func (c *FakeExternalNetworks) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.ExternalNetwork, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(externalnetworksResource, name), &v3.ExternalNetwork{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.ExternalNetwork), err
}

// List takes label and field selectors, and returns the list of ExternalNetworks that match those selectors.
func (c *FakeExternalNetworks) List(ctx context.Context, opts v1.ListOptions) (result *v3.ExternalNetworkList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(externalnetworksResource, externalnetworksKind, opts), &v3.ExternalNetworkList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.ExternalNetworkList{ListMeta: obj.(*v3.ExternalNetworkList).ListMeta}
	for _, item := range obj.(*v3.ExternalNetworkList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested externalNetworks.
func (c *FakeExternalNetworks) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(externalnetworksResource, opts))
}

// Create takes the representation of a externalNetwork and creates it.  Returns the server's representation of the externalNetwork, and an error, if there is any.
func (c *FakeExternalNetworks) Create(ctx context.Context, externalNetwork *v3.ExternalNetwork, opts v1.CreateOptions) (result *v3.ExternalNetwork, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(externalnetworksResource, externalNetwork), &v3.ExternalNetwork{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.ExternalNetwork), err
}

// Update takes the representation of a externalNetwork and updates it. Returns the server's representation of the externalNetwork, and an error, if there is any.
func (c *FakeExternalNetworks) Update(ctx context.Context, externalNetwork *v3.ExternalNetwork, opts v1.UpdateOptions) (result *v3.ExternalNetwork, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(externalnetworksResource, externalNetwork), &v3.ExternalNetwork{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.ExternalNetwork), err
}

// Delete takes name of the externalNetwork and deletes it. Returns an error if one occurs.
func (c *FakeExternalNetworks) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(externalnetworksResource, name, opts), &v3.ExternalNetwork{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeExternalNetworks) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(externalnetworksResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.ExternalNetworkList{})
	return err
}

// Patch applies the patch and returns the patched externalNetwork.
func (c *FakeExternalNetworks) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.ExternalNetwork, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(externalnetworksResource, name, pt, data, subresources...), &v3.ExternalNetwork{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.ExternalNetwork), err
}
