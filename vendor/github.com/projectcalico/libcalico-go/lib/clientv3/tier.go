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

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// TierInterface has methods to work with Tier resources.
type TierInterface interface {
	Create(ctx context.Context, res *apiv3.Tier, opts options.SetOptions) (*apiv3.Tier, error)
	Update(ctx context.Context, res *apiv3.Tier, opts options.SetOptions) (*apiv3.Tier, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.Tier, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.Tier, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.TierList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// tiers implements TierInterface
type tiers struct {
	client client
}

// Create takes the representation of a Tier and creates it.  Returns the stored
// representation of the Tier, and an error, if there is any.
func (r tiers) Create(ctx context.Context, res *apiv3.Tier, opts options.SetOptions) (*apiv3.Tier, error) {
	if res.Name == defaultTierName && res.Spec.Order != nil {
		return nil, cerrors.ErrorOperationNotSupported{
			Identifier: defaultTierName,
			Operation:  "Create",
			Reason:     "Default tier should have nil Order",
		}
	}
	out, err := r.client.resources.Create(ctx, opts, apiv3.KindTier, res)
	if out != nil {
		return out.(*apiv3.Tier), err
	}
	return nil, err
}

// Update takes the representation of a Tier and updates it. Returns the stored
// representation of the Tier, and an error, if there is any.
func (r tiers) Update(ctx context.Context, res *apiv3.Tier, opts options.SetOptions) (*apiv3.Tier, error) {
	if res.GetObjectMeta().GetName() == defaultTierName && res.Spec.Order != nil {
		return nil, cerrors.ErrorOperationNotSupported{
			Identifier: defaultTierName,
			Operation:  "Update",
			Reason:     "Cannot update the order of the default tier",
		}
	}
	out, err := r.client.resources.Update(ctx, opts, apiv3.KindTier, res)
	if out != nil {
		return out.(*apiv3.Tier), err
	}
	return nil, err
}

// Delete takes name of the Tier and deletes it. Returns an error if one occurs.
func (r tiers) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.Tier, error) {
	if name == defaultTierName {
		return nil, cerrors.ErrorOperationNotSupported{
			Identifier: defaultTierName,
			Operation:  "Delete",
			Reason:     "Cannot delete default tier",
		}
	}

	// Check if there are any policies associated with the tier first.
	npList, err := r.client.NetworkPolicies().List(ctx, options.ListOptions{
		Prefix: true,
		Name:   name + ".",
	})
	if err != nil {
		return nil, err
	}

	gnpList, err := r.client.GlobalNetworkPolicies().List(ctx, options.ListOptions{
		Prefix: true,
		Name:   name + ".",
	})
	if err != nil {
		return nil, err
	}

	if len(npList.Items) > 0 || len(gnpList.Items) > 0 {
		return nil, cerrors.ErrorOperationNotSupported{
			Operation:  "delete",
			Identifier: name,
			Reason:     "Cannot delete a non-empty tier",
		}
	}

	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindTier, noNamespace, name)
	if out != nil {
		return out.(*apiv3.Tier), err
	}
	return nil, err
}

// Get takes name of the Tier, and returns the corresponding Tier object,
// and an error if there is any.
func (r tiers) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.Tier, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindTier, noNamespace, name)
	if out != nil {
		return out.(*apiv3.Tier), err
	}
	return nil, err
}

// List returns the list of Tier objects that match the supplied options.
func (r tiers) List(ctx context.Context, opts options.ListOptions) (*apiv3.TierList, error) {
	res := &apiv3.TierList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindTier, apiv3.KindTierList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the tiers that match the
// supplied options.
func (r tiers) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindTier, nil)
}
