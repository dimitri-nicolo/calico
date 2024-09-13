// Copyright (c) 2020 Tigera, Inc. All rights reserved.

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
	"time"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/options"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// BFDConfigurationInterface has methods to work with BFDConfiguration resources.
type BFDConfigurationInterface interface {
	Create(ctx context.Context, res *apiv3.BFDConfiguration, opts options.SetOptions) (*apiv3.BFDConfiguration, error)
	Update(ctx context.Context, res *apiv3.BFDConfiguration, opts options.SetOptions) (*apiv3.BFDConfiguration, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.BFDConfiguration, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.BFDConfiguration, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.BFDConfigurationList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// bfdConfigurations implements BFDConfigurationInterface
type bfdConfigurations struct {
	client client
}

// Create takes the representation of a BFDConfiguration and creates it.
// Returns the stored representation of the BFDConfiguration, and an error
// if there is any.
func (r bfdConfigurations) Create(ctx context.Context, res *apiv3.BFDConfiguration, opts options.SetOptions) (*apiv3.BFDConfiguration, error) {
	if err := r.setDefaults(res); err != nil {
		return nil, err
	}

	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindBFDConfiguration, res)
	if out != nil {
		return out.(*apiv3.BFDConfiguration), err
	}
	return nil, err
}

// Update takes the representation of a BFDConfiguration and updates it.
// Returns the stored representation of the BFDConfiguration, and an error
// if there is any.
func (r bfdConfigurations) Update(ctx context.Context, res *apiv3.BFDConfiguration, opts options.SetOptions) (*apiv3.BFDConfiguration, error) {
	if err := r.setDefaults(res); err != nil {
		return nil, err
	}

	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindBFDConfiguration, res)
	if out != nil {
		return out.(*apiv3.BFDConfiguration), err
	}
	return nil, err
}

// Delete takes name of the BFDConfiguration and deletes it. Returns an
// error if one occurs.
func (r bfdConfigurations) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.BFDConfiguration, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindBFDConfiguration, noNamespace, name)
	if out != nil {
		return out.(*apiv3.BFDConfiguration), err
	}
	return nil, err
}

// Get takes name of the BFDConfiguration, and returns the corresponding
// BFDConfiguration object, and an error if there is any.
func (r bfdConfigurations) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.BFDConfiguration, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindBFDConfiguration, noNamespace, name)
	if out != nil {
		return out.(*apiv3.BFDConfiguration), err
	}
	return nil, err
}

// List returns the list of BFDConfiguration objects that match the supplied options.
func (r bfdConfigurations) List(ctx context.Context, opts options.ListOptions) (*apiv3.BFDConfigurationList, error) {
	res := &apiv3.BFDConfigurationList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindBFDConfiguration, apiv3.KindBFDConfigurationList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the BFDConfiguration that
// match the supplied options.
func (r bfdConfigurations) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindBFDConfiguration, nil)
}

// setDefaults sets the default values for a BFDConfiguration if not set.
// Note that this is largely to make etcd mode tests pass, as default values in Kubernetes datastore mode
// are set by the CRD.
func (r bfdConfigurations) setDefaults(res *apiv3.BFDConfiguration) error {
	for i := range res.Spec.Interfaces {
		iface := &res.Spec.Interfaces[i]
		if iface.MinimumRecvInterval == nil {
			iface.MinimumRecvInterval = &metav1.Duration{Duration: 10 * time.Millisecond}
		}
		if iface.MinimumSendInterval == nil {
			iface.MinimumSendInterval = &metav1.Duration{Duration: 100 * time.Millisecond}
		}
		if iface.IdleSendInterval == nil {
			iface.IdleSendInterval = &metav1.Duration{Duration: time.Minute}
		}
		if iface.Multiplier == 0 {
			iface.Multiplier = 5
		}
		if iface.MatchPattern == "" {
			iface.MatchPattern = "*"
		}
	}
	return nil
}
