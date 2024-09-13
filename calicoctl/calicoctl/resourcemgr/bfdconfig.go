// Copyright (c) 2024 Tigera, Inc. All rights reserved.

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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	client "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

func init() {
	registerResource(
		api.NewBFDConfiguration(),
		newBFDConfigurationList(),
		false,
		[]string{"bfdconfiguration", "bfdconfigurations", "bfdconfig", "bfdconfigs"},
		[]string{"NAME"},
		[]string{"NAME", "SELECTOR"},
		map[string]string{
			"NAME":     "{{.ObjectMeta.Name}}",
			"SELECTOR": "{{.Spec.NodeSelector}}",
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.BFDConfiguration)
			return client.BFDConfigurations().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.BFDConfiguration)
			return client.BFDConfigurations().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.BFDConfiguration)
			return client.BFDConfigurations().Delete(ctx, r.Name, options.DeleteOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.BFDConfiguration)
			return client.BFDConfigurations().Get(ctx, r.Name, options.GetOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceListObject, error) {
			r := resource.(*api.BFDConfiguration)
			return client.BFDConfigurations().List(ctx, options.ListOptions{ResourceVersion: r.ResourceVersion, Name: r.Name})
		},
	)
}

// NewBFDConfigurationList creates a new zeroed) BFDConfigurationList struct with the TypeMetadata
// initialized to the current version.
func newBFDConfigurationList() *api.BFDConfigurationList {
	return &api.BFDConfigurationList{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.KindBFDConfigurationList,
			APIVersion: api.GroupVersionCurrent,
		},
	}
}
