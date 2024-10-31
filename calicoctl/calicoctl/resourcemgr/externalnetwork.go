// Copyright (c) 2016-2017,2021 Tigera, Inc. All rights reserved.

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

package resourcemgr

import (
	"context"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	client "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

func init() {
	registerResource(
		api.NewExternalNetwork(),
		newExternalNetworkList(),
		false,
		[]string{"externalnetwork", "externalnetworks", "externalnet", "enet", "enets"},
		[]string{"NAME", "ROUTETABLEINDEX"},
		[]string{"NAME", "ROUTETABLEINDEX"},
		map[string]string{
			"NAME":            "{{.ObjectMeta.Name}}",
			"ROUTETABLEINDEX": "{{.Spec.RouteTableIndex}}",
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.ExternalNetwork)
			return client.ExternalNetworks().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.ExternalNetwork)
			return client.ExternalNetworks().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.ExternalNetwork)
			return client.ExternalNetworks().Delete(ctx, r.Name, options.DeleteOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.ExternalNetwork)
			return client.ExternalNetworks().Get(ctx, r.Name, options.GetOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceListObject, error) {
			r := resource.(*api.ExternalNetwork)
			return client.ExternalNetworks().List(ctx, options.ListOptions{ResourceVersion: r.ResourceVersion, Name: r.Name})
		},
	)
}

// newExternalNetworkList creates a new (zeroed) ExternalNetworkList struct with the TypeMetadata initialised to the current
// version.
func newExternalNetworkList() *api.ExternalNetworkList {
	return &api.ExternalNetworkList{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.KindExternalNetworkList,
			APIVersion: api.GroupVersionCurrent,
		},
	}
}
