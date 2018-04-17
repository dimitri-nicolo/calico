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
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	log "github.com/sirupsen/logrus"
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
			licClaims, err := licClient.Decode(*r)
			if err != nil {
				return nil, fmt.Errorf("license is corrupted: %s", err.Error())
			}

			if err = licClaims.Validate(); err != nil {
				// License is already expired. Don't apply it.
				return nil, fmt.Errorf("the license you're trying to creat is already expired on %s", licClaims.Expiry.Time().Local())
			} else {
				log.Debug("License is valid")
			}

			return client.LicenseKey().Create(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			r := resource.(*api.LicenseKey)
			licClaims, err := licClient.Decode(*r)
			if err != nil {
				return nil, fmt.Errorf("license is corrupted: %s", err.Error())
			}

			if err = licClaims.Validate(); err != nil {
				// License is already expired. Don't apply it.
				return nil, fmt.Errorf("the license you're trying to apply is already expired on %s", licClaims.Expiry.Time().Local())
			} else {
				log.Debug("License is valid")
			}

			currentLic, err := client.LicenseKey().Get(ctx, "default", options.GetOptions{})
			if err != nil {
				switch err.(type) {
				case cerrors.ErrorResourceDoesNotExist:
					log.Debugf("Check for an existing LicenseKey: not found. Moving on")
				default:
					log.WithError(err).Debug("Failed to load the existing LicenseKey from datastore. Moving on")
				}
			} else {
				log.Info("License resource found")
				currentLicClaims, err := licClient.Decode(*currentLic)
				if err != nil {
					// Existing license is likely corrupted.
					// Do nothing.
				} else {
					if licClaims.Expiry.Time().Before(currentLicClaims.Expiry.Time()) {
						// The license we're applying expires sooner than the one that's already applied.
						// We reject this change so users don't shoot themselves in the foot.
						return nil, fmt.Errorf("the license you're applying expires on %s, which is sooner than " +
							"the one already applied %s", licClaims.Expiry.Time().Local(), currentLicClaims.Expiry.Time().Local())
					}
				}
			}

			return client.LicenseKey().Update(ctx, r, options.SetOptions{})
		},
		func(ctx context.Context, client client.Interface, resource ResourceObject) (ResourceObject, error) {
			return nil, fmt.Errorf("deleting a license is not supported")
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
