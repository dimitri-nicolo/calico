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

// FakePacketCaptures implements PacketCaptureInterface
type FakePacketCaptures struct {
	Fake *FakeProjectcalicoV3
	ns   string
}

var packetcapturesResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "packetcaptures"}

var packetcapturesKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "PacketCapture"}

// Get takes name of the packetCapture, and returns the corresponding packetCapture object, and an error if there is any.
func (c *FakePacketCaptures) Get(name string, options v1.GetOptions) (result *v3.PacketCapture, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(packetcapturesResource, c.ns, name), &v3.PacketCapture{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PacketCapture), err
}

// List takes label and field selectors, and returns the list of PacketCaptures that match those selectors.
func (c *FakePacketCaptures) List(opts v1.ListOptions) (result *v3.PacketCaptureList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(packetcapturesResource, packetcapturesKind, c.ns, opts), &v3.PacketCaptureList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.PacketCaptureList{ListMeta: obj.(*v3.PacketCaptureList).ListMeta}
	for _, item := range obj.(*v3.PacketCaptureList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested packetCaptures.
func (c *FakePacketCaptures) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(packetcapturesResource, c.ns, opts))

}

// Create takes the representation of a packetCapture and creates it.  Returns the server's representation of the packetCapture, and an error, if there is any.
func (c *FakePacketCaptures) Create(packetCapture *v3.PacketCapture) (result *v3.PacketCapture, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(packetcapturesResource, c.ns, packetCapture), &v3.PacketCapture{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PacketCapture), err
}

// Update takes the representation of a packetCapture and updates it. Returns the server's representation of the packetCapture, and an error, if there is any.
func (c *FakePacketCaptures) Update(packetCapture *v3.PacketCapture) (result *v3.PacketCapture, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(packetcapturesResource, c.ns, packetCapture), &v3.PacketCapture{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PacketCapture), err
}

// Delete takes name of the packetCapture and deletes it. Returns an error if one occurs.
func (c *FakePacketCaptures) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(packetcapturesResource, c.ns, name), &v3.PacketCapture{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePacketCaptures) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(packetcapturesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v3.PacketCaptureList{})
	return err
}

// Patch applies the patch and returns the patched packetCapture.
func (c *FakePacketCaptures) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.PacketCapture, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(packetcapturesResource, c.ns, name, pt, data, subresources...), &v3.PacketCapture{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.PacketCapture), err
}
