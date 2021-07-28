// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package resourcemgr

import (
	"context"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

func init() {
	registerResource(
		api.NewGlobalThreatFeed(),
		api.NewGlobalThreatFeedList(),
		false,
		[]string{"globalthreatfeed", "globalthreatfeeds"},
		[]string{"NAME"},
		[]string{"NAME", "PERIOD", "URL"},
		map[string]string{
			"NAME":   "{{.ObjectMeta.Name}}",
			"PERIOD": "{{if .Spec.Pull}}{{if .Spec.Pull.Period }}{{.Spec.Pull.Period}}{{else}}24h{{end}}{{end}}",
			"URL":    "{{if .Spec.Pull}}{{if .Spec.Pull.HTTP}}{{.Spec.Pull.HTTP.URL}}{{end}}{{end}}",
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.GlobalThreatFeed)
			return client.GlobalThreatFeeds().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.GlobalThreatFeed)
			return client.GlobalThreatFeeds().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.GlobalThreatFeed)
			return client.GlobalThreatFeeds().Delete(ctx, r.Name, options.DeleteOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.GlobalThreatFeed)
			return client.GlobalThreatFeeds().Get(ctx, r.Name, options.GetOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceListObject, error) {
			r := resource.(*api.GlobalThreatFeed)
			return client.GlobalThreatFeeds().List(ctx, options.ListOptions{ResourceVersion: r.ResourceVersion, Name: r.Name})
		},
	)
}
