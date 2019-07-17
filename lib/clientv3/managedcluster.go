// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package clientv3

import (
	"context"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// ManagedClusterInterface has methods to work with ManagedCluster resources.
type ManagedClusterInterface interface {
	Create(ctx context.Context, res *apiv3.ManagedCluster, opts options.SetOptions) (*apiv3.ManagedCluster, error)
	Update(ctx context.Context, res *apiv3.ManagedCluster, opts options.SetOptions) (*apiv3.ManagedCluster, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.ManagedCluster, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.ManagedCluster, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.ManagedClusterList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// managedClusters implements ManagedClusterInterface
type managedClusters struct {
	client client
}

// Create takes the representation of a ManagedCluster and creates it.  Returns the stored
// representation of the ManagedCluster, and an error, if there is any.
func (r managedClusters) Create(ctx context.Context, res *apiv3.ManagedCluster, opts options.SetOptions) (*apiv3.ManagedCluster, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindManagedCluster, res)
	if out != nil {
		return out.(*apiv3.ManagedCluster), err
	}
	return nil, err
}

// Update takes the representation of a ManagedCluster and updates it. Returns the stored
// representation of the ManagedCluster, and an error, if there is any.
func (r managedClusters) Update(ctx context.Context, res *apiv3.ManagedCluster, opts options.SetOptions) (*apiv3.ManagedCluster, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindManagedCluster, res)
	if out != nil {
		return out.(*apiv3.ManagedCluster), err
	}
	return nil, err
}

// Delete takes name of the ManagedCluster and deletes it. Returns an error if one occurs.
func (r managedClusters) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.ManagedCluster, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindManagedCluster, noNamespace, name)
	if out != nil {
		return out.(*apiv3.ManagedCluster), err
	}
	return nil, err
}

// Get takes name of the ManagedCluster, and returns the corresponding ManagedCluster object,
// and an error if there is any.
func (r managedClusters) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.ManagedCluster, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindManagedCluster, noNamespace, name)
	if out != nil {
		return out.(*apiv3.ManagedCluster), err
	}
	return nil, err
}

// List returns the list of ManagedCluster objects that match the supplied options.
func (r managedClusters) List(ctx context.Context, opts options.ListOptions) (*apiv3.ManagedClusterList, error) {
	res := &apiv3.ManagedClusterList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindManagedCluster, apiv3.KindManagedClusterList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the ManagedClusters that match the
// supplied options.
func (r managedClusters) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindManagedCluster, nil)
}
