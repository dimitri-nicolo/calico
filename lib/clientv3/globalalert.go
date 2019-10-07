// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// GlobalAlertInterface has methods to work with GlobalAlert resources.
type GlobalAlertInterface interface {
	Create(ctx context.Context, res *apiv3.GlobalAlert, opts options.SetOptions) (*apiv3.GlobalAlert, error)
	Update(ctx context.Context, res *apiv3.GlobalAlert, opts options.SetOptions) (*apiv3.GlobalAlert, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalAlert, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalAlert, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalAlertList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// globalAlerts implements GlobalAlertInterface
type globalAlerts struct {
	client client
}

// Create takes the representation of a GlobalAlert and creates it.  Returns the stored
// representation of the GlobalAlert, and an error, if there is any.
func (r globalAlerts) Create(ctx context.Context, res *apiv3.GlobalAlert, opts options.SetOptions) (*apiv3.GlobalAlert, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindGlobalAlert, res)
	if out != nil {
		return out.(*apiv3.GlobalAlert), err
	}
	return nil, err
}

// Update takes the representation of a GlobalAlert and updates it. Returns the stored
// representation of the GlobalAlert, and an error, if there is any.
func (r globalAlerts) Update(ctx context.Context, res *apiv3.GlobalAlert, opts options.SetOptions) (*apiv3.GlobalAlert, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindGlobalAlert, res)
	if out != nil {
		return out.(*apiv3.GlobalAlert), err
	}
	return nil, err
}

// Delete takes name of the GlobalAlert and deletes it. Returns an error if one occurs.
func (r globalAlerts) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalAlert, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindGlobalAlert, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalAlert), err
	}
	return nil, err
}

// Get takes name of the GlobalAlert, and returns the corresponding GlobalAlert object,
// and an error if there is any.
func (r globalAlerts) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalAlert, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindGlobalAlert, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalAlert), err
	}
	return nil, err
}

// List returns the list of GlobalAlert objects that match the supplied options.
func (r globalAlerts) List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalAlertList, error) {
	res := &apiv3.GlobalAlertList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindGlobalAlert, apiv3.KindGlobalAlertList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the GlobalAlerts that match the
// supplied options.
func (r globalAlerts) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindGlobalAlert, nil)
}
