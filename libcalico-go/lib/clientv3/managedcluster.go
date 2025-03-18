// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package clientv3

import (
	"context"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// ManagedClusterInterface has methods to work with ManagedCluster resources.
type ManagedClusterInterface interface {
	Create(ctx context.Context, res *apiv3.ManagedCluster, opts options.SetOptions) (*apiv3.ManagedCluster, error)
	Update(ctx context.Context, res *apiv3.ManagedCluster, opts options.SetOptions) (*apiv3.ManagedCluster, error)

	// Delete and Get honors the namespace in multitenant mode and ignores it in the single tenant mode.
	// multitenant mode is enabled by setting a CalicoAPIConfigSpec's MultiTenantEnabled to be true.
	Delete(ctx context.Context, namespace, name string, opts options.DeleteOptions) (*apiv3.ManagedCluster, error)
	Get(ctx context.Context, namespace, name string, opts options.GetOptions) (*apiv3.ManagedCluster, error)

	List(ctx context.Context, opts options.ListOptions) (*apiv3.ManagedClusterList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// managedClusters implements ManagedClusterInterface
type managedClusters struct {
	client client
}

const (
	// ErrMsgNotEmpty is the error message returned when editing InstallationManifest field for a ManagedCluster
	ErrMsgNotEmpty             = "InstallationManifest is a reserved field and is not editable"
	ErrMsgEmptyTenantNamespace = "Tenant namespace is empty in multitenant mode"
)

// Create takes the representation of a ManagedCluster and creates it.  Returns the stored
// representation of the ManagedCluster, and an error, if there is any.
func (r managedClusters) Create(ctx context.Context, res *apiv3.ManagedCluster, opts options.SetOptions) (*apiv3.ManagedCluster, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	// Management and standalone cluster use *.cluster.* as Elasticsearch index name. Do not allow
	// the word "cluster" as managed cluster name to maintain separate index for each cluster.
	var invalidManagedClusterName = "cluster"
	if res.ObjectMeta.Name == invalidManagedClusterName {
		return nil, cerrors.ErrorValidation{
			ErroredFields: []cerrors.ErroredField{{
				Name:   "Metadata.Name",
				Reason: "Invalid name for managed cluster, \"cluster\" is a reserved value.",
				Value:  res.ObjectMeta.Name,
			}},
		}
	}

	// InstallationManifest is a reserved field that will be populated by the API server
	// when generating a managed cluster resource. This field contains the manifest
	// that will be applied on the managed cluster to setup a TLS connection
	if len(res.Spec.InstallationManifest) != 0 {
		return nil, cerrors.ErrorValidation{
			ErroredFields: []cerrors.ErroredField{{
				Name:   "Metadata.Name",
				Reason: ErrMsgNotEmpty,
				Value:  res.ObjectMeta.Name,
			}},
		}
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

	// InstallationManifest is a reserved field that will be populated by the API server
	// when generating a managed cluster resource. This field contains the manifest
	// that will be applied on the managed cluster to setup a TLS connection
	if len(res.Spec.InstallationManifest) != 0 {
		return nil, cerrors.ErrorValidation{
			ErroredFields: []cerrors.ErroredField{{
				Name:   "Metadata.Name",
				Reason: ErrMsgNotEmpty,
				Value:  res.ObjectMeta.Name,
			}},
		}
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindManagedCluster, res)
	if out != nil {
		return out.(*apiv3.ManagedCluster), err
	}
	return nil, err
}

// Delete takes name of the ManagedCluster and deletes it. Returns an error if one occurs.
func (r managedClusters) Delete(ctx context.Context, namespace, name string, opts options.DeleteOptions) (*apiv3.ManagedCluster, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindManagedCluster, namespace, name)
	if out != nil {
		return out.(*apiv3.ManagedCluster), err
	}
	return nil, err
}

// Get takes name of the ManagedCluster, and returns the corresponding ManagedCluster object,
// and an error if there is any.
func (r managedClusters) Get(ctx context.Context, namespace, name string, opts options.GetOptions) (*apiv3.ManagedCluster, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindManagedCluster, namespace, name)
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
