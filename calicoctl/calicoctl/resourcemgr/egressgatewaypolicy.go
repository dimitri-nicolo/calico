// Copyright (c) 2023 Tigera, Inc. All rights reserved.

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
		api.NewEgressGatewayPolicy(),
		NewEgressGatewayPolicyList(),
		false,
		[]string{"egressgatewaypolicy", "egressgatewaypolicies", "egressgatewayp", "egressgwp", "egwpolicy", "egwpolicies", "egwp"},
		[]string{"NAME", "RULES"},
		[]string{"NAME", "RULES"},
		map[string]string{
			"NAME":  "{{.ObjectMeta.Name}}",
			"Rules": "{{ len .Spec.Rules}}",
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.EgressGatewayPolicy)
			return client.EgressGatewayPolicy().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.EgressGatewayPolicy)
			return client.EgressGatewayPolicy().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.EgressGatewayPolicy)
			return client.EgressGatewayPolicy().Delete(ctx, r.Name, options.DeleteOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.EgressGatewayPolicy)
			return client.EgressGatewayPolicy().Get(ctx, r.Name, options.GetOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceListObject, error) {
			r := resource.(*api.EgressGatewayPolicy)
			return client.EgressGatewayPolicy().List(ctx, options.ListOptions{ResourceVersion: r.ResourceVersion, Name: r.Name})
		},
	)
}

// newEgressGatewayPolicyList creates a new (zeroed) EgressGatewayPolicyList struct with the TypeMetadata initialised to the current
// version.
func NewEgressGatewayPolicyList() *api.EgressGatewayPolicyList {
	return &api.EgressGatewayPolicyList{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.KindEgressGatewayPolicyList,
			APIVersion: api.GroupVersionCurrent,
		},
	}
}
