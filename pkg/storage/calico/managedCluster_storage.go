// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"crypto/md5"
	"fmt"
	"reflect"

	"k8s.io/klog"

	cerrors "github.com/projectcalico/libcalico-go/lib/errors"

	licClient "github.com/tigera/licensing/client"
	"github.com/tigera/licensing/client/features"

	"github.com/projectcalico/apiserver/pkg/helpers"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	libcalicoapi "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"

	aapi "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
)

// AnnotationActiveCertificateFingerprint is an annotation that is used to store the fingerprint for
// managed cluster certificate that is allowed to initiate connections.
const AnnotationActiveCertificateFingerprint = "certs.tigera.io/active-fingerprint"

// NewManagedClusterStorage creates a new libcalico-based storage.Interface implementation for ManagedClusters
func NewManagedClusterStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	resources := opts.ManagedClusterResources
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.ManagedCluster)

		if resources == nil {
			return nil, cerrors.ErrorValidation{
				ErroredFields: []cerrors.ErroredField{{
					Name:   "Metadata.Name",
					Reason: fmt.Sprintf("This API is not available"),
					Value:  res.ObjectMeta.Name,
				}},
			}
		}

		// Generate x509 certificate and private key for the managed cluster
		certificate, privKey, err := helpers.Generate(resources.CACert, resources.CAKey, res.ObjectMeta.Name)
		if err != nil {
			klog.Errorf("Cannot generate managed cluster certificate and key due to %s", err)
			return nil, cerrors.ErrorValidation{
				ErroredFields: []cerrors.ErroredField{{
					Name:   "Metadata.Name",
					Reason: "Failed to generate client credentials",
					Value:  res.ObjectMeta.Name,
				}},
			}
		}
		// Store the hash of the certificate as an annotation
		fingerprint := fmt.Sprintf("%x", md5.Sum(certificate.Raw))
		if res.Annotations == nil {
			res.Annotations = make(map[string]string)
		}
		res.Annotations[AnnotationActiveCertificateFingerprint] = fingerprint

		// Create the managed cluster resource
		out, err := c.ManagedClusters().Create(ctx, res, oso)
		if err != nil {
			return nil, err
		}

		// Populate the installation manifest in the response
		out.Spec.InstallationManifest = helpers.InstallationManifest(resources.CACert, certificate, privKey, resources.ManagementClusterAddr)
		return out, nil
	}

	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.ManagedCluster)
		return c.ManagedClusters().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.ManagedClusters().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.ManagedClusters().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.ManagedClusters().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.ManagedClusters().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return !claims.ValidateFeature(features.MultiClusterManagement)
	}

	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.ManagedCluster{}),
		aapiListType:      reflect.TypeOf(aapi.ManagedClusterList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.ManagedCluster{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.ManagedClusterList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "ManagedCluster",
		converter:         ManagedClusterConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type ManagedClusterConverter struct {
}

func (gc ManagedClusterConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiManagedCluster := aapiObj.(*aapi.ManagedCluster)
	lcgManagedCluster := &libcalicoapi.ManagedCluster{}
	lcgManagedCluster.TypeMeta = aapiManagedCluster.TypeMeta
	lcgManagedCluster.ObjectMeta = aapiManagedCluster.ObjectMeta
	lcgManagedCluster.Kind = libcalicoapi.KindManagedCluster
	lcgManagedCluster.APIVersion = libcalicoapi.GroupVersionCurrent
	lcgManagedCluster.Spec = aapiManagedCluster.Spec
	lcgManagedCluster.Status = aapiManagedCluster.Status
	return lcgManagedCluster
}

func (gc ManagedClusterConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgManagedCluster := libcalicoObject.(*libcalicoapi.ManagedCluster)
	aapiManagedCluster := aapiObj.(*aapi.ManagedCluster)
	aapiManagedCluster.Spec = lcgManagedCluster.Spec
	aapiManagedCluster.Status = lcgManagedCluster.Status
	aapiManagedCluster.TypeMeta = lcgManagedCluster.TypeMeta
	aapiManagedCluster.ObjectMeta = lcgManagedCluster.ObjectMeta
}

func (gc ManagedClusterConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgManagedClusterList := libcalicoListObject.(*libcalicoapi.ManagedClusterList)
	aapiManagedClusterList := aapiListObj.(*aapi.ManagedClusterList)
	if libcalicoListObject == nil {
		aapiManagedClusterList.Items = []aapi.ManagedCluster{}
		return
	}
	aapiManagedClusterList.TypeMeta = lcgManagedClusterList.TypeMeta
	aapiManagedClusterList.ListMeta = lcgManagedClusterList.ListMeta
	for _, item := range lcgManagedClusterList.Items {
		aapiManagedCluster := aapi.ManagedCluster{}
		gc.convertToAAPI(&item, &aapiManagedCluster)
		if matched, err := pred.Matches(&aapiManagedCluster); err == nil && matched {
			aapiManagedClusterList.Items = append(aapiManagedClusterList.Items, aapiManagedCluster)
		}
	}
}
