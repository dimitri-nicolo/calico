// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package clientv3

import (
	"context"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
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
	if err := r.defaultOverlayRoutingMode(ctx, res); err != nil {
		return nil, err
	}
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
	if err := r.defaultOverlayRoutingMode(ctx, res); err != nil {
		return nil, err
	}
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

// defaultOverlayRoutingMode sets OverlayRoutingMode to Enabled if a VXLAN IPPool is present, and to Disabled otherwise.
func (r remoteClusterConfiguration) defaultOverlayRoutingMode(ctx context.Context, res *apiv3.RemoteClusterConfiguration) error {
	if res.Spec.SyncOptions.OverlayRoutingMode != "" {
		return nil
	}

	// Check if Wireguard is globally enabled.
	out, err := r.client.resources.Get(ctx, options.GetOptions{}, apiv3.KindFelixConfiguration, noNamespace, "default")
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
			return err
		}
	} else {
		cfg := out.(*apiv3.FelixConfiguration)
		if (cfg.Spec.WireguardEnabled != nil && *cfg.Spec.WireguardEnabled) || (cfg.Spec.WireguardEnabledV6 != nil && *cfg.Spec.WireguardEnabledV6) {
			res.Spec.SyncOptions.OverlayRoutingMode = apiv3.OverlayRoutingModeEnabled
			return nil
		}
	}

	// Check if VXLAN is enabled.
	ipPoolList := &apiv3.IPPoolList{}
	if err := r.client.resources.List(ctx, options.ListOptions{}, apiv3.KindIPPool, apiv3.KindIPPoolList, ipPoolList); err != nil {
		return err
	}
	for _, ipPool := range ipPoolList.Items {
		if ipPool.Spec.VXLANMode != apiv3.VXLANModeNever {
			res.Spec.SyncOptions.OverlayRoutingMode = apiv3.OverlayRoutingModeEnabled
			return nil
		}
	}
	res.Spec.SyncOptions.OverlayRoutingMode = apiv3.OverlayRoutingModeDisabled
	return nil
}
