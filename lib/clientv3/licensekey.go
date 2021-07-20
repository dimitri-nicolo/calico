// Copyright (c) 2017 Tigera, Inc. All rights reserved.

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

package clientv3

import (
	"context"
	"errors"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// LicenseKeyInterface has methods to work with LicenseKey resources.
type LicenseKeyInterface interface {
	Create(ctx context.Context, res *apiv3.LicenseKey, opts options.SetOptions) (*apiv3.LicenseKey, error)
	Update(ctx context.Context, res *apiv3.LicenseKey, opts options.SetOptions) (*apiv3.LicenseKey, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.LicenseKey, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.LicenseKey, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.LicenseKeyList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// LicenseKey implements LicenseKeyInterface
type licenseKey struct {
	client client
}

// Create takes the representation of a LicenseKey and creates it.
// Returns the stored representation of the LicenseKey, and an error
// if there is any.
func (r licenseKey) Create(ctx context.Context, res *apiv3.LicenseKey, opts options.SetOptions) (*apiv3.LicenseKey, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	if res.ObjectMeta.GetName() != "default" {
		return nil, errors.New("Cannot create a License Key resource with a name other than \"default\"")
	}
	out, err := r.client.resources.Create(ctx, opts, apiv3.KindLicenseKey, res)
	if out != nil {
		return out.(*apiv3.LicenseKey), err
	}
	return nil, err
}

// Update takes the representation of a LicenseKey and updates it.
// Returns the stored representation of the LicenseKey, and an error
// if there is any.
func (r licenseKey) Update(ctx context.Context, res *apiv3.LicenseKey, opts options.SetOptions) (*apiv3.LicenseKey, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindLicenseKey, res)
	if out != nil {
		return out.(*apiv3.LicenseKey), err
	}
	return nil, err
}

// Delete takes name of the LicenseKey and deletes it. Returns an
// error if one occurs.
func (r licenseKey) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.LicenseKey, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindLicenseKey, noNamespace, name)
	if out != nil {
		return out.(*apiv3.LicenseKey), err
	}
	return nil, err
}

// Get takes name of the LicenseKey, and returns the corresponding
// LicenseKey object, and an error if there is any.
func (r licenseKey) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.LicenseKey, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindLicenseKey, noNamespace, name)
	if out != nil {
		return out.(*apiv3.LicenseKey), err
	}
	return nil, err
}

// List returns the list of LicenseKey objects that match the supplied options.
func (r licenseKey) List(ctx context.Context, opts options.ListOptions) (*apiv3.LicenseKeyList, error) {
	res := &apiv3.LicenseKeyList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindLicenseKey, apiv3.KindLicenseKeyList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the LicenseKey that
// match the supplied options.
func (r licenseKey) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindLicenseKey, nil)
}
