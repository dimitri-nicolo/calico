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

// PolicyRecommendationScopeInterface has methods to work with PolicyRecommendationScope resources.
type PolicyRecommendationScopeInterface interface {
	Create(ctx context.Context, res *apiv3.PolicyRecommendationScope, opts options.SetOptions) (*apiv3.PolicyRecommendationScope, error)
	Update(ctx context.Context, res *apiv3.PolicyRecommendationScope, opts options.SetOptions) (*apiv3.PolicyRecommendationScope, error)
	UpdateStatus(ctx context.Context, res *apiv3.PolicyRecommendationScope, opts options.SetOptions) (*apiv3.PolicyRecommendationScope, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.PolicyRecommendationScope, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.PolicyRecommendationScope, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.PolicyRecommendationScopeList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// policyRecommendationScopes implements PolicyRecommendationScopeInterface
type policyRecommendationScopes struct {
	client client
}

// Create takes the representation of a PolicyRecommendationScope and creates it.  Returns the stored
// representation of the PolicyRecommendationScope, and an error, if there is any.
func (r policyRecommendationScopes) Create(ctx context.Context, res *apiv3.PolicyRecommendationScope, opts options.SetOptions) (*apiv3.PolicyRecommendationScope, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindPolicyRecommendationScope, res)
	if out != nil {
		return out.(*apiv3.PolicyRecommendationScope), err
	}
	return nil, err
}

// Update takes the representation of a PolicyRecommendationScope and updates it. Returns the stored
// representation of the PolicyRecommendationScope, and an error, if there is any.
func (r policyRecommendationScopes) Update(ctx context.Context, res *apiv3.PolicyRecommendationScope, opts options.SetOptions) (*apiv3.PolicyRecommendationScope, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindPolicyRecommendationScope, res)
	if out != nil {
		return out.(*apiv3.PolicyRecommendationScope), err
	}
	return nil, err
}

// UpdateStatus takes the representation of a PolicyRecommendationScope and updates the status section of it. Returns the stored
// representation of the PolicyRecommendationScope, and an error, if there is any.
func (r policyRecommendationScopes) UpdateStatus(ctx context.Context, res *apiv3.PolicyRecommendationScope, opts options.SetOptions) (*apiv3.PolicyRecommendationScope, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.UpdateStatus(ctx, opts, apiv3.KindPolicyRecommendationScope, res)
	if out != nil {
		return out.(*apiv3.PolicyRecommendationScope), err
	}
	return nil, err
}

// Delete takes name of the PolicyRecommendationScope and deletes it. Returns an error if one occurs.
func (r policyRecommendationScopes) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.PolicyRecommendationScope, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindPolicyRecommendationScope, noNamespace, name)
	if out != nil {
		return out.(*apiv3.PolicyRecommendationScope), err
	}
	return nil, err
}

// Get takes name of the PolicyRecommendationScope, and returns the corresponding PolicyRecommendationScope object,
// and an error if there is any.open
func (r policyRecommendationScopes) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.PolicyRecommendationScope, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindPolicyRecommendationScope, noNamespace, name)
	if out != nil {
		return out.(*apiv3.PolicyRecommendationScope), err
	}
	return nil, err
}

// List returns the list of PolicyRecommendationScope objects that match the supplied options.
func (r policyRecommendationScopes) List(ctx context.Context, opts options.ListOptions) (*apiv3.PolicyRecommendationScopeList, error) {
	res := &apiv3.PolicyRecommendationScopeList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindPolicyRecommendationScope, apiv3.KindPolicyRecommendationScopeList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the PolicyRecommendationScopes that match the
// supplied options.
func (r policyRecommendationScopes) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindPolicyRecommendationScope, nil)
}
