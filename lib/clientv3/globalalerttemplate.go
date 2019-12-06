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

// GlobalAlertTemplateInterface has methods to work with GlobalAlertTemplate resources.
type GlobalAlertTemplateInterface interface {
	Create(ctx context.Context, res *apiv3.GlobalAlertTemplate, opts options.SetOptions) (*apiv3.GlobalAlertTemplate, error)
	Update(ctx context.Context, res *apiv3.GlobalAlertTemplate, opts options.SetOptions) (*apiv3.GlobalAlertTemplate, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalAlertTemplate, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalAlertTemplate, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalAlertTemplateList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// globalAlertTemplates implements GlobalAlertTemplateInterface
type globalAlertTemplates struct {
	client client
}

// Create takes the representation of a GlobalAlertTemplate and creates it.  Returns the stored
// representation of the GlobalAlertTemplate, and an error, if there is any.
func (r globalAlertTemplates) Create(ctx context.Context, res *apiv3.GlobalAlertTemplate, opts options.SetOptions) (*apiv3.GlobalAlertTemplate, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindGlobalAlertTemplate, res)
	if out != nil {
		return out.(*apiv3.GlobalAlertTemplate), err
	}
	return nil, err
}

// Update takes the representation of a GlobalAlertTemplate and updates it. Returns the stored
// representation of the GlobalAlertTemplate, and an error, if there is any.
func (r globalAlertTemplates) Update(ctx context.Context, res *apiv3.GlobalAlertTemplate, opts options.SetOptions) (*apiv3.GlobalAlertTemplate, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindGlobalAlertTemplate, res)
	if out != nil {
		return out.(*apiv3.GlobalAlertTemplate), err
	}
	return nil, err
}

// Delete takes name of the GlobalAlertTemplate and deletes it. Returns an error if one occurs.
func (r globalAlertTemplates) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalAlertTemplate, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindGlobalAlertTemplate, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalAlertTemplate), err
	}
	return nil, err
}

// Get takes name of the GlobalAlertTemplate, and returns the corresponding GlobalAlertTemplate object,
// and an error if there is any.
func (r globalAlertTemplates) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalAlertTemplate, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindGlobalAlertTemplate, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalAlertTemplate), err
	}
	return nil, err
}

// List returns the list of GlobalAlertTemplate objects that match the supplied options.
func (r globalAlertTemplates) List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalAlertTemplateList, error) {
	res := &apiv3.GlobalAlertTemplateList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindGlobalAlertTemplate, apiv3.KindGlobalAlertTemplateList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the GlobalAlertTemplates that match the
// supplied options.
func (r globalAlertTemplates) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindGlobalAlertTemplate, nil)
}
