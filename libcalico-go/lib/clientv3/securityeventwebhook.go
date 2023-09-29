// Copyright (c) 2023 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/projectcalico/calico/libcalico-go/lib/options"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// SecurityEventWebhookInterface has methods to work with SecurityEventWebhook resources.
type SecurityEventWebhookInterface interface {
	Create(ctx context.Context, res *apiv3.SecurityEventWebhook, opts options.SetOptions) (*apiv3.SecurityEventWebhook, error)
	Update(ctx context.Context, res *apiv3.SecurityEventWebhook, opts options.SetOptions) (*apiv3.SecurityEventWebhook, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.SecurityEventWebhook, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.SecurityEventWebhook, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.SecurityEventWebhookList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// SecurityEventWebhooks implements SecurityEventWebhookInterface
type SecurityEventWebhooks struct {
	client client
}

// Create takes the representation of a SecurityEventWebhook and creates it.
// Returns the stored representation of the SecurityEventWebhook and an error, if there is any.
func (r SecurityEventWebhooks) Create(ctx context.Context, res *apiv3.SecurityEventWebhook, opts options.SetOptions) (*apiv3.SecurityEventWebhook, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindSecurityEventWebhook, res)
	if out != nil {
		return out.(*apiv3.SecurityEventWebhook), err
	}
	return nil, err
}

// Update takes the representation of a SecurityEventWebhook and updates it.
// Returns the stored representation of the SecurityEventWebhook, and an error, if there is any.
func (r SecurityEventWebhooks) Update(ctx context.Context, res *apiv3.SecurityEventWebhook, opts options.SetOptions) (*apiv3.SecurityEventWebhook, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindSecurityEventWebhook, res)
	if out != nil {
		return out.(*apiv3.SecurityEventWebhook), err
	}
	return nil, err
}

// Delete takes name of the SecurityEventWebhook and deletes it. Returns an error if one occurs.
func (r SecurityEventWebhooks) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.SecurityEventWebhook, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindSecurityEventWebhook, noNamespace, name)
	if out != nil {
		return out.(*apiv3.SecurityEventWebhook), err
	}
	return nil, err
}

// Get takes name of the SecurityEventWebhook.
// Returns the corresponding SecurityEventWebhook object and an error if there is any.
func (r SecurityEventWebhooks) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.SecurityEventWebhook, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindSecurityEventWebhook, noNamespace, name)
	if out != nil {
		return out.(*apiv3.SecurityEventWebhook), err
	}
	return nil, err
}

// List returns the list of SecurityEventWebhook objects that match the supplied options.
func (r SecurityEventWebhooks) List(ctx context.Context, opts options.ListOptions) (*apiv3.SecurityEventWebhookList, error) {
	res := &apiv3.SecurityEventWebhookList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindSecurityEventWebhook, apiv3.KindSecurityEventWebhookList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the SecurityEventWebhooks that match the supplied options.
func (r SecurityEventWebhooks) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindSecurityEventWebhook, nil)
}
