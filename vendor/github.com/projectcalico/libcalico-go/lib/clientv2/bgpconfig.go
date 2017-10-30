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

	apiv2 "github.com/projectcalico/libcalico-go/lib/apis/v2"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// BGPConfigurationInterface has methods to work with BGPConfiguration resources.
type BGPConfigurationInterface interface {
	Create(ctx context.Context, res *apiv2.BGPConfiguration, opts options.SetOptions) (*apiv2.BGPConfiguration, error)
	Update(ctx context.Context, res *apiv2.BGPConfiguration, opts options.SetOptions) (*apiv2.BGPConfiguration, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv2.BGPConfiguration, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv2.BGPConfiguration, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv2.BGPConfigurationList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// bgpConfigurations implements BGPConfigurationInterface
type bgpConfigurations struct {
	client client
}

// Create takes the representation of a BGPConfiguration and creates it.
// Returns the stored representation of the BGPConfiguration, and an error
// if there is any.
func (r bgpConfigurations) Create(ctx context.Context, res *apiv2.BGPConfiguration, opts options.SetOptions) (*apiv2.BGPConfiguration, error) {
	if err := r.ValidateDefaultOnlyFields(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv2.KindBGPConfiguration, res)
	if out != nil {
		return out.(*apiv2.BGPConfiguration), err
	}
	return nil, err
}

// Update takes the representation of a BGPConfiguration and updates it.
// Returns the stored representation of the BGPConfiguration, and an error
// if there is any.
func (r bgpConfigurations) Update(ctx context.Context, res *apiv2.BGPConfiguration, opts options.SetOptions) (*apiv2.BGPConfiguration, error) {
	// Check that NodeToNodeMeshEnabled and ASNumber are set. Can only be set on "default".
	if err := r.ValidateDefaultOnlyFields(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv2.KindBGPConfiguration, res)
	if out != nil {
		return out.(*apiv2.BGPConfiguration), err
	}
	return nil, err
}

// Delete takes name of the BGPConfiguration and deletes it. Returns an
// error if one occurs.
func (r bgpConfigurations) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv2.BGPConfiguration, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv2.KindBGPConfiguration, noNamespace, name)
	if out != nil {
		return out.(*apiv2.BGPConfiguration), err
	}
	return nil, err
}

// Get takes name of the BGPConfiguration, and returns the corresponding
// BGPConfiguration object, and an error if there is any.
func (r bgpConfigurations) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv2.BGPConfiguration, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv2.KindBGPConfiguration, noNamespace, name)
	if out != nil {
		return out.(*apiv2.BGPConfiguration), err
	}
	return nil, err
}

// List returns the list of BGPConfiguration objects that match the supplied options.
func (r bgpConfigurations) List(ctx context.Context, opts options.ListOptions) (*apiv2.BGPConfigurationList, error) {
	res := &apiv2.BGPConfigurationList{}
	if err := r.client.resources.List(ctx, opts, apiv2.KindBGPConfiguration, apiv2.KindBGPConfigurationList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the BGPConfiguration that
// match the supplied options.
func (r bgpConfigurations) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv2.KindBGPConfiguration)
}

func (r bgpConfigurations) ValidateDefaultOnlyFields(res *apiv2.BGPConfiguration) error {
	errFields := []cerrors.ErroredField{}
	if res.ObjectMeta.GetName() != "default" {
		if res.Spec.NodeToNodeMeshEnabled != nil {
			errFields = append(errFields, cerrors.ErroredField{
				Name:   "BGPConfiguration.Spec.NodeToNodeMeshEnabled",
				Reason: "Cannot set nodeToNodeMeshEnabled on a non default BGP Configuration.",
			})
		}

		if res.Spec.ASNumber != nil {
			errFields = append(errFields, cerrors.ErroredField{
				Name:   "BGPConfiguration.Spec.ASNumber",
				Reason: "Cannot set ASNumber on a non default BGP Configuration.",
			})
		}
	}

	if len(errFields) > 0 {
		return cerrors.ErrorValidation{
			ErroredFields: errFields,
		}
	}

	return nil
}
