// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.

package calico

import (
	"crypto/sha256"
	"fmt"
	"reflect"

	"golang.org/x/net/context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
	"k8s.io/klog"

	"github.com/projectcalico/calico/apiserver/pkg/helpers"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
	"github.com/projectcalico/calico/licensing/client/features"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
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
		res := obj.(*v3.ManagedCluster)

		if resources == nil {
			return nil, cerrors.ErrorValidation{
				ErroredFields: []cerrors.ErroredField{{
					Name:   "Metadata.Name",
					Reason: "ManagementCluster must be configured before adding ManagedClusters",
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
		fingerprint := fmt.Sprintf("%x", sha256.Sum256(certificate.Raw))
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
		// - If the operatorNamespace is not set in the ManagedCluster resource, default to tigera-operator.
		operatorNs := res.Spec.OperatorNamespace
		if operatorNs == "" {
			operatorNs = "tigera-operator"
		}
		out.Spec.InstallationManifest = helpers.InstallationManifest(resources.CACert, certificate, privKey, resources.ManagementClusterAddr, resources.ManagementClusterCAType, operatorNs)
		return out, nil
	}

	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.ManagedCluster)
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
	hasRestrictionsFn := func(obj resourceObject) bool {
		return !opts.LicenseMonitor.GetFeatureStatus(features.MultiClusterManagement)
	}
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.ManagedCluster{}),
		aapiListType:      reflect.TypeOf(v3.ManagedClusterList{}),
		libCalicoType:     reflect.TypeOf(v3.ManagedCluster{}),
		libCalicoListType: reflect.TypeOf(v3.ManagedClusterList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		hasRestrictions:   hasRestrictionsFn,
		resourceName:      "ManagedCluster",
		converter:         ManagedClusterConverter{},
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type ManagedClusterConverter struct {
}

func (gc ManagedClusterConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiManagedCluster := aapiObj.(*v3.ManagedCluster)
	lcgManagedCluster := &v3.ManagedCluster{}
	lcgManagedCluster.TypeMeta = aapiManagedCluster.TypeMeta
	lcgManagedCluster.ObjectMeta = aapiManagedCluster.ObjectMeta
	lcgManagedCluster.Kind = v3.KindManagedCluster
	lcgManagedCluster.APIVersion = v3.GroupVersionCurrent
	lcgManagedCluster.Spec = aapiManagedCluster.Spec
	lcgManagedCluster.Status = aapiManagedCluster.Status
	return lcgManagedCluster
}

func (gc ManagedClusterConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgManagedCluster := libcalicoObject.(*v3.ManagedCluster)
	aapiManagedCluster := aapiObj.(*v3.ManagedCluster)
	aapiManagedCluster.Spec = lcgManagedCluster.Spec
	aapiManagedCluster.Status = lcgManagedCluster.Status
	aapiManagedCluster.TypeMeta = lcgManagedCluster.TypeMeta
	aapiManagedCluster.ObjectMeta = lcgManagedCluster.ObjectMeta
}

func (gc ManagedClusterConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgManagedClusterList := libcalicoListObject.(*v3.ManagedClusterList)
	aapiManagedClusterList := aapiListObj.(*v3.ManagedClusterList)
	if libcalicoListObject == nil {
		aapiManagedClusterList.Items = []v3.ManagedCluster{}
		return
	}
	aapiManagedClusterList.TypeMeta = lcgManagedClusterList.TypeMeta
	aapiManagedClusterList.ListMeta = lcgManagedClusterList.ListMeta
	for _, item := range lcgManagedClusterList.Items {
		aapiManagedCluster := v3.ManagedCluster{}
		gc.convertToAAPI(&item, &aapiManagedCluster)
		if matched, err := pred.Matches(&aapiManagedCluster); err == nil && matched {
			aapiManagedClusterList.Items = append(aapiManagedClusterList.Items, aapiManagedCluster)
		}
	}
}
