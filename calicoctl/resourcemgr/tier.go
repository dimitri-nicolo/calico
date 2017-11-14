// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package resourcemgr

import (
	"context"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

func init() {
	registerResource(
		api.NewTier(),
		api.NewTierList(),
		false,
		[]string{"tier", "tiers"},
		[]string{"NAME", "ORDER"},
		[]string{"NAME", "ORDER"},
		map[string]string{
			"NAME":  "{{.ObjectMeta.Name}}",
			"ORDER": "{{.Spec.Order}}",
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.Tier)
			return client.Tiers().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.Tier)
			return client.Tiers().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.Tier)
			return client.Tiers().Delete(ctx, r.Name, options.DeleteOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.Tier)
			return client.Tiers().Get(ctx, r.Name, options.GetOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceListObject, error) {
			r := resource.(*api.Tier)
			return client.Tiers().List(ctx, options.ListOptions{ResourceVersion: r.ResourceVersion, Name: r.Name})
		},
	)
}
