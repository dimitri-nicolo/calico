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

package clientv2

import (
	"context"

	"github.com/projectcalico/libcalico-go/lib/apiv2"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// TierInterface has methods to work with Tier resources.
type TierInterface interface {
	Create(ctx context.Context, res *apiv2.Tier, opts options.SetOptions) (*apiv2.Tier, error)
	Update(ctx context.Context, res *apiv2.Tier, opts options.SetOptions) (*apiv2.Tier, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv2.Tier, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv2.Tier, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv2.TierList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// tiers implements TierInterface
type tiers struct {
	client client
}

// Create takes the representation of a Tier and creates it.  Returns the stored
// representation of the Tier, and an error, if there is any.
func (r tiers) Create(ctx context.Context, res *apiv2.Tier, opts options.SetOptions) (*apiv2.Tier, error) {
	out, err := r.client.resources.Create(ctx, opts, apiv2.KindTier, res)
	if out != nil {
		return out.(*apiv2.Tier), err
	}
	return nil, err
}

// Update takes the representation of a Tier and updates it. Returns the stored
// representation of the Tier, and an error, if there is any.
func (r tiers) Update(ctx context.Context, res *apiv2.Tier, opts options.SetOptions) (*apiv2.Tier, error) {
	out, err := r.client.resources.Update(ctx, opts, apiv2.KindTier, res)
	if out != nil {
		return out.(*apiv2.Tier), err
	}
	return nil, err
}

// Delete takes name of the Tier and deletes it. Returns an error if one occurs.
func (r tiers) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv2.Tier, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv2.KindTier, noNamespace, name)
	if out != nil {
		return out.(*apiv2.Tier), err
	}
	return nil, err
}

// Get takes name of the Tier, and returns the corresponding Tier object,
// and an error if there is any.
func (r tiers) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv2.Tier, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv2.KindTier, noNamespace, name)
	if out != nil {
		return out.(*apiv2.Tier), err
	}
	return nil, err
}

// List returns the list of Tier objects that match the supplied options.
func (r tiers) List(ctx context.Context, opts options.ListOptions) (*apiv2.TierList, error) {
	res := &apiv2.TierList{}
	if err := r.client.resources.List(ctx, opts, apiv2.KindTier, apiv2.KindTierList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the tiers that match the
// supplied options.
func (r tiers) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv2.KindTier)
}
