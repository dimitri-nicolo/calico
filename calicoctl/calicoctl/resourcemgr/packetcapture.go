// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package resourcemgr

import (
	"context"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

func init() {
	registerResource(
		api.NewPacketCapture(),
		api.NewPacketCaptureList(),
		true,
		[]string{"packetcapture", "packetcaptures"},
		[]string{"NAME"},
		[]string{"NAME", "SELECTOR"},
		map[string]string{
			"NAME":      "{{.ObjectMeta.Name}}",
			"NAMESPACE": "{{.ObjectMeta.Namespace}}",
			"SELECTOR":  "{{.Spec.Selector}}",
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.PacketCapture)
			return client.PacketCaptures().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.PacketCapture)
			return client.PacketCaptures().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.PacketCapture)
			return client.PacketCaptures().Delete(ctx, r.Namespace, r.Name, options.DeleteOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.PacketCapture)
			return client.PacketCaptures().Get(ctx, r.Namespace, r.Name, options.GetOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceListObject, error) {
			r := resource.(*api.PacketCapture)
			return client.PacketCaptures().List(ctx, options.ListOptions{ResourceVersion: r.ResourceVersion, Namespace: r.Namespace, Name: r.Name})
		},
	)
}
