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

// GlobalThreatFeedInterface has methods to work with GlobalThreatFeed resources.
type GlobalThreatFeedInterface interface {
	Create(ctx context.Context, res *apiv3.GlobalThreatFeed, opts options.SetOptions) (*apiv3.GlobalThreatFeed, error)
	Update(ctx context.Context, res *apiv3.GlobalThreatFeed, opts options.SetOptions) (*apiv3.GlobalThreatFeed, error)
	UpdateStatus(ctx context.Context, res *apiv3.GlobalThreatFeed, opts options.SetOptions) (*apiv3.GlobalThreatFeed, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalThreatFeed, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalThreatFeed, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalThreatFeedList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// globalThreatFeeds implements GlobalThreatFeedInterface
type globalThreatFeeds struct {
	client client
}

// Create takes the representation of a GlobalThreatFeed and creates it.  Returns the stored
// representation of the GlobalThreatFeed, and an error, if there is any.
func (r globalThreatFeeds) Create(ctx context.Context, res *apiv3.GlobalThreatFeed, opts options.SetOptions) (*apiv3.GlobalThreatFeed, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindGlobalThreatFeed, res)
	if out != nil {
		return out.(*apiv3.GlobalThreatFeed), err
	}
	return nil, err
}

// Update takes the representation of a GlobalThreatFeed and updates it. Returns the stored
// representation of the GlobalThreatFeed, and an error, if there is any.
func (r globalThreatFeeds) Update(ctx context.Context, res *apiv3.GlobalThreatFeed, opts options.SetOptions) (*apiv3.GlobalThreatFeed, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindGlobalThreatFeed, res)
	if out != nil {
		return out.(*apiv3.GlobalThreatFeed), err
	}
	return nil, err
}

// UpdateStatus takes the representation of a GlobalThreatFeed and updates the status section of it. Returns the stored
// representation of the GlobalThreatFeed, and an error, if there is any.
func (r globalThreatFeeds) UpdateStatus(ctx context.Context, res *apiv3.GlobalThreatFeed, opts options.SetOptions) (*apiv3.GlobalThreatFeed, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.UpdateStatus(ctx, opts, apiv3.KindGlobalThreatFeed, res)
	if out != nil {
		return out.(*apiv3.GlobalThreatFeed), err
	}
	return nil, err
}

// Delete takes name of the GlobalThreatFeed and deletes it. Returns an error if one occurs.
func (r globalThreatFeeds) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalThreatFeed, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindGlobalThreatFeed, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalThreatFeed), err
	}
	return nil, err
}

// Get takes name of the GlobalThreatFeed, and returns the corresponding GlobalThreatFeed object,
// and an error if there is any.
func (r globalThreatFeeds) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalThreatFeed, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindGlobalThreatFeed, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalThreatFeed), err
	}
	return nil, err
}

// List returns the list of GlobalThreatFeed objects that match the supplied options.
func (r globalThreatFeeds) List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalThreatFeedList, error) {
	res := &apiv3.GlobalThreatFeedList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindGlobalThreatFeed, apiv3.KindGlobalThreatFeedList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the GlobalThreatFeeds that match the
// supplied options.
func (r globalThreatFeeds) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindGlobalThreatFeed, nil)
}
