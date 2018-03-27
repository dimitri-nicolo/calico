// Copyright (c) 2018 Tigera, Inc. All rights reserved.

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
	"fmt"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	licClient "github.com/tigera/licensing/client"
)

func init() {
	registerResource(
		api.NewLicenseKey(),
		api.NewLicenseKeyList(),
		false,
		[]string{"license", "licensekey", "lic", "licenses", "licensekeys"},
		[]string{"LICENSEID", "EXPIRATION", "NODES"},
		[]string{"LICENSEID", "EXPIRATION", "NODES", "FEATURES"},
		map[string]string{
			"LICENSEID":   "{{.LicenseID}}",
			"EXPIRATION":  "{{localtime .Claims.Expiry}}",
			"NODES":       "{{.Nodes}}",
			"FEATURES":    "{{.Features}}",
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.LicenseKey)
			_, err := licClient.Decode(*r)
			if err != nil {
				return nil, fmt.Errorf("LicenseKey is corrupted: %s", err.Error())
			}

			return client.LicenseKey().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.LicenseKey)
			_, err := licClient.Decode(*r)
			if err != nil {
				return nil, fmt.Errorf("LicenseKey is corrupted: %s", err.Error())
			}

			return client.LicenseKey().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			return nil, fmt.Errorf("deleting a LicenseKey is not supported")
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.LicenseKey)
			return client.LicenseKey().Get(ctx, r.Name, options.GetOptions{ResourceVersion: r.ResourceVersion})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceListObject, error) {
			r := resource.(*api.LicenseKey)
			return client.LicenseKey().List(ctx, options.ListOptions{ResourceVersion: r.ResourceVersion, Name: r.Name})
		},
	)
}
