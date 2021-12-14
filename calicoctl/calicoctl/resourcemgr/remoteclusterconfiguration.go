// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

package resourcemgr

import (
	"context"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

func init() {
	registerResource(
		api.NewRemoteClusterConfiguration(),
		api.NewRemoteClusterConfigurationList(),
		false,
		[]string{"remoteclusterconfiguration", "remoteclusterconfig", "remoteclusterconfigurations", "remoteclusterconfigs", "rcc"},
		[]string{"NAME"},
		[]string{"NAME", "DATASTORETYPE", "ETCDENDPOINTS", "K8SAPIENDPOINT"},
		map[string]string{
			"NAME":           "{{.ObjectMeta.Name}}",
			"ETCDENDPOINTS":  "{{.Spec.EtcdEndpoints}}",
			"K8SAPIENDPOINT": "{{.Spec.K8sAPIEndpoint}}",
			"DATASTORETYPE":  "{{.Spec.DatastoreType}}",
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.RemoteClusterConfiguration)
			return client.RemoteClusterConfigurations().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.RemoteClusterConfiguration)
			return client.RemoteClusterConfigurations().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.RemoteClusterConfiguration)
			return client.RemoteClusterConfigurations().Delete(ctx, r.Name, options.DeleteOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.RemoteClusterConfiguration)
			return client.RemoteClusterConfigurations().Get(ctx, r.Name, options.GetOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceListObject, error) {
			r := resource.(*api.RemoteClusterConfiguration)
			return client.RemoteClusterConfigurations().List(ctx, options.ListOptions{ResourceVersion: r.ResourceVersion, Name: r.Name})
		},
	)
}
