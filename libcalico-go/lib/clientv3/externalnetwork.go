// Copyright (c) 2022 Tigera, Inc. All rights reserved.

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

	"github.com/projectcalico/calico/libcalico-go/lib/options"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// ExternalNetworkInterface has methods to work with ExternalNetwork resources.
type ExternalNetworkInterface interface {
	Create(ctx context.Context, res *apiv3.ExternalNetwork, opts options.SetOptions) (*apiv3.ExternalNetwork, error)
	Update(ctx context.Context, res *apiv3.ExternalNetwork, opts options.SetOptions) (*apiv3.ExternalNetwork, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.ExternalNetwork, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.ExternalNetwork, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.ExternalNetworkList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// ExternalNetwork implements ExternalNetworkInterface
type ExternalNetwork struct {
	client client
}

// Create takes the representation of a ExternalNetwork and creates it.  Returns the stored
// representation of the ExternalNetwork, and an error, if there is any.
func (r ExternalNetwork) Create(ctx context.Context, res *apiv3.ExternalNetwork, opts options.SetOptions) (*apiv3.ExternalNetwork, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindExternalNetwork, res)
	if out != nil {
		return out.(*apiv3.ExternalNetwork), err
	}
	return nil, err
}

// Update takes the representation of a ExternalNetwork and updates it. Returns the stored
// representation of the ExternalNetwork, and an error, if there is any.
func (r ExternalNetwork) Update(ctx context.Context, res *apiv3.ExternalNetwork, opts options.SetOptions) (*apiv3.ExternalNetwork, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindExternalNetwork, res)
	if out != nil {
		return out.(*apiv3.ExternalNetwork), err
	}
	return nil, err
}

// Delete takes name of the ExternalNetwork and deletes it. Returns an error if one occurs.
func (r ExternalNetwork) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.ExternalNetwork, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindExternalNetwork, noNamespace, name)
	if out != nil {
		return out.(*apiv3.ExternalNetwork), err
	}
	return nil, err
}

// Get takes name of the ExternalNetwork, and returns the corresponding ExternalNetwork object,
// and an error if there is any.
func (r ExternalNetwork) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.ExternalNetwork, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindExternalNetwork, noNamespace, name)
	if out != nil {
		return out.(*apiv3.ExternalNetwork), err
	}
	return nil, err
}

// List returns the list of ExternalNetwork objects that match the supplied options.
func (r ExternalNetwork) List(ctx context.Context, opts options.ListOptions) (*apiv3.ExternalNetworkList, error) {
	res := &apiv3.ExternalNetworkList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindExternalNetwork, apiv3.KindExternalNetworkList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the BGPPeers that match the
// supplied options.
func (r ExternalNetwork) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindExternalNetwork, nil)
}
