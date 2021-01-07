// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePacketCaptures implements PacketCaptureInterface
type FakePacketCaptures struct {
	Fake *FakeProjectcalico
	ns   string
}

var packetcapturesResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "packetcaptures"}

var packetcapturesKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "PacketCapture"}

// Get takes name of the packetCapture, and returns the corresponding packetCapture object, and an error if there is any.
func (c *FakePacketCaptures) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.PacketCapture, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(packetcapturesResource, c.ns, name), &projectcalico.PacketCapture{})

	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.PacketCapture), err
}

// List takes label and field selectors, and returns the list of PacketCaptures that match those selectors.
func (c *FakePacketCaptures) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.PacketCaptureList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(packetcapturesResource, packetcapturesKind, c.ns, opts), &projectcalico.PacketCaptureList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.PacketCaptureList{ListMeta: obj.(*projectcalico.PacketCaptureList).ListMeta}
	for _, item := range obj.(*projectcalico.PacketCaptureList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested packetCaptures.
func (c *FakePacketCaptures) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(packetcapturesResource, c.ns, opts))

}

// Create takes the representation of a packetCapture and creates it.  Returns the server's representation of the packetCapture, and an error, if there is any.
func (c *FakePacketCaptures) Create(ctx context.Context, packetCapture *projectcalico.PacketCapture, opts v1.CreateOptions) (result *projectcalico.PacketCapture, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(packetcapturesResource, c.ns, packetCapture), &projectcalico.PacketCapture{})

	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.PacketCapture), err
}

// Update takes the representation of a packetCapture and updates it. Returns the server's representation of the packetCapture, and an error, if there is any.
func (c *FakePacketCaptures) Update(ctx context.Context, packetCapture *projectcalico.PacketCapture, opts v1.UpdateOptions) (result *projectcalico.PacketCapture, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(packetcapturesResource, c.ns, packetCapture), &projectcalico.PacketCapture{})

	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.PacketCapture), err
}

// Delete takes name of the packetCapture and deletes it. Returns an error if one occurs.
func (c *FakePacketCaptures) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(packetcapturesResource, c.ns, name), &projectcalico.PacketCapture{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePacketCaptures) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(packetcapturesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &projectcalico.PacketCaptureList{})
	return err
}

// Patch applies the patch and returns the patched packetCapture.
func (c *FakePacketCaptures) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.PacketCapture, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(packetcapturesResource, c.ns, name, pt, data, subresources...), &projectcalico.PacketCapture{})

	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.PacketCapture), err
}
