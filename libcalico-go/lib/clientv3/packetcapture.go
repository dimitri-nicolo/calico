// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package clientv3

import (
	"context"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// PacketCaptureInterface has methods to work with PacketCapture resources.
type PacketCaptureInterface interface {
	Create(ctx context.Context, res *apiv3.PacketCapture, opts options.SetOptions) (*apiv3.PacketCapture, error)
	Update(ctx context.Context, res *apiv3.PacketCapture, opts options.SetOptions) (*apiv3.PacketCapture, error)
	Delete(ctx context.Context, namespace, name string, opts options.DeleteOptions) (*apiv3.PacketCapture, error)
	Get(ctx context.Context, namespace, name string, opts options.GetOptions) (*apiv3.PacketCapture, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.PacketCaptureList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// packetCaptures implements PacketCaptureInterface
type packetCaptures struct {
	client client
}

// Create takes the representation of a PacketCapture and creates it.  Returns the stored
// representation of the PacketCapture, and an error, if there is any.
func (r packetCaptures) Create(ctx context.Context, res *apiv3.PacketCapture, opts options.SetOptions) (*apiv3.PacketCapture, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}
	out, err := r.client.resources.Create(ctx, opts, apiv3.KindPacketCapture, res)
	if out != nil {
		return out.(*apiv3.PacketCapture), err
	}
	return nil, err
}

// Update takes the representation of a PacketCapture and updates it. Returns the stored
// representation of the PacketCapture, and an error, if there is any.
func (r packetCaptures) Update(ctx context.Context, res *apiv3.PacketCapture, opts options.SetOptions) (*apiv3.PacketCapture, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}
	out, err := r.client.resources.Update(ctx, opts, apiv3.KindPacketCapture, res)
	if out != nil {
		return out.(*apiv3.PacketCapture), err
	}
	return nil, err
}

// Delete takes name of the PacketCapture and deletes it. Returns an error if one occurs.
func (r packetCaptures) Delete(ctx context.Context, namespace, name string, opts options.DeleteOptions) (*apiv3.PacketCapture, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindPacketCapture, namespace, name)
	if out != nil {
		return out.(*apiv3.PacketCapture), err
	}
	return nil, err
}

// Get takes name of the PacketCapture, and returns the corresponding PacketCapture object,
// and an error if there is any.
func (r packetCaptures) Get(ctx context.Context, namespace, name string, opts options.GetOptions) (*apiv3.PacketCapture, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindPacketCapture, namespace, name)
	if out != nil {
		return out.(*apiv3.PacketCapture), err
	}
	return nil, err
}

// List returns the list of PacketCapture objects that match the supplied options.
func (r packetCaptures) List(ctx context.Context, opts options.ListOptions) (*apiv3.PacketCaptureList, error) {
	res := &apiv3.PacketCaptureList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindPacketCapture, apiv3.KindPacketCaptureList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the PacketCaptures that match the
// supplied options.
func (r packetCaptures) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindPacketCapture, nil)
}
