// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package clientv3

import (
	"context"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// RemoteClusterConfigurationInterface has methods to work with RemoteClusterConfiguration resources.
type RemoteClusterConfigurationInterface interface {
	Create(ctx context.Context, res *apiv3.RemoteClusterConfiguration, opts options.SetOptions) (*apiv3.RemoteClusterConfiguration, error)
	Update(ctx context.Context, res *apiv3.RemoteClusterConfiguration, opts options.SetOptions) (*apiv3.RemoteClusterConfiguration, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.RemoteClusterConfiguration, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.RemoteClusterConfiguration, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.RemoteClusterConfigurationList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// remoteClusterConfiguration implements RemoteClusterConfigurationInterface
type remoteClusterConfiguration struct {
	client client
}

// Create takes the representation of a RemoteClusterConfiguration and creates it.  Returns the stored
// representation of the RemoteClusterConfiguration, and an error, if there is any.
func (r remoteClusterConfiguration) Create(ctx context.Context, res *apiv3.RemoteClusterConfiguration, opts options.SetOptions) (*apiv3.RemoteClusterConfiguration, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindRemoteClusterConfiguration, res)
	if out != nil {
		return out.(*apiv3.RemoteClusterConfiguration), err
	}
	return nil, err
}

// Update takes the representation of a RemoteClusterConfiguration and updates it. Returns the stored
// representation of the RemoteClusterConfiguration, and an error, if there is any.
func (r remoteClusterConfiguration) Update(ctx context.Context, res *apiv3.RemoteClusterConfiguration, opts options.SetOptions) (*apiv3.RemoteClusterConfiguration, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindRemoteClusterConfiguration, res)
	if out != nil {
		return out.(*apiv3.RemoteClusterConfiguration), err
	}
	return nil, err
}

// Delete takes name of the RemoteClusterConfiguration and deletes it. Returns an error if one occurs.
func (r remoteClusterConfiguration) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.RemoteClusterConfiguration, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindRemoteClusterConfiguration, noNamespace, name)
	if out != nil {
		return out.(*apiv3.RemoteClusterConfiguration), err
	}
	return nil, err
}

// Get takes name of the RemoteClusterConfiguration, and returns the corresponding RemoteClusterConfiguration object,
// and an error if there is any.
func (r remoteClusterConfiguration) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.RemoteClusterConfiguration, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindRemoteClusterConfiguration, noNamespace, name)
	if out != nil {
		return out.(*apiv3.RemoteClusterConfiguration), err
	}
	return nil, err
}

// List returns the list of RemoteClusterConfiguration objects that match the supplied options.
func (r remoteClusterConfiguration) List(ctx context.Context, opts options.ListOptions) (*apiv3.RemoteClusterConfigurationList, error) {
	res := &apiv3.RemoteClusterConfigurationList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindRemoteClusterConfiguration, apiv3.KindRemoteClusterConfigurationList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the RemoteClusterConfigurations that match the
// supplied options.
func (r remoteClusterConfiguration) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindRemoteClusterConfiguration, nil)
}
