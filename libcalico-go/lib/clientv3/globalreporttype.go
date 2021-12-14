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

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// GlobalReportTypeInterface has methods to work with GlobalReportType resources.
type GlobalReportTypeInterface interface {
	Create(ctx context.Context, res *apiv3.GlobalReportType, opts options.SetOptions) (*apiv3.GlobalReportType, error)
	Update(ctx context.Context, res *apiv3.GlobalReportType, opts options.SetOptions) (*apiv3.GlobalReportType, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalReportType, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalReportType, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalReportTypeList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// globalReportTypes implements GlobalReportTypeInterface
type globalReportTypes struct {
	client client
}

// Create takes the representation of a GlobalReportType and creates it.  Returns the stored
// representation of the GlobalReportType, and an error, if there is any.
func (r globalReportTypes) Create(ctx context.Context, res *apiv3.GlobalReportType, opts options.SetOptions) (*apiv3.GlobalReportType, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindGlobalReportType, res)
	if out != nil {
		return out.(*apiv3.GlobalReportType), err
	}
	return nil, err
}

// Update takes the representation of a GlobalReportType and updates it. Returns the stored
// representation of the GlobalReportType, and an error, if there is any.
func (r globalReportTypes) Update(ctx context.Context, res *apiv3.GlobalReportType, opts options.SetOptions) (*apiv3.GlobalReportType, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindGlobalReportType, res)
	if out != nil {
		return out.(*apiv3.GlobalReportType), err
	}
	return nil, err
}

// Delete takes name of the GlobalReportType and deletes it. Returns an error if one occurs.
func (r globalReportTypes) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalReportType, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindGlobalReportType, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalReportType), err
	}
	return nil, err
}

// Get takes name of the GlobalReportType, and returns the corresponding GlobalReportType object,
// and an error if there is any.
func (r globalReportTypes) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalReportType, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindGlobalReportType, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalReportType), err
	}
	return nil, err
}

// List returns the list of GlobalReportType objects that match the supplied options.
func (r globalReportTypes) List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalReportTypeList, error) {
	res := &apiv3.GlobalReportTypeList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindGlobalReportType, apiv3.KindGlobalReportTypeList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the GlobalReportTypes that match the
// supplied options.
func (r globalReportTypes) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindGlobalReportType, nil)
}
