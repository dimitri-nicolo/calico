// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	log "github.com/sirupsen/logrus"
	licClient "github.com/tigera/licensing/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	copier "github.com/jinzhu/copier"

	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
)

// NewUISettingsGroupStorage creates a new storage. Interface implementation for UISettingsGroups.
func NewUISettingsGroupStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	getFn := func(ctx context.Context, c clientv3.Interface, ns, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.UISettingsGroups().Get(ctx, name, ogo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.UISettingsGroups().List(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject, claims *licClient.LicenseClaims) bool {
		return false
	}

	// TODO(doublek): Inject codec, client for nicer testing.
	// TODO(dimitrin): Remove aapi and libCalico fields once resource has been refactored. Types
	// refer to the same
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(api.UISettingsGroup{}),
		aapiListType:      reflect.TypeOf(api.UISettingsGroupList{}),
		libCalicoType:     reflect.TypeOf(api.UISettingsGroup{}),
		libCalicoListType: reflect.TypeOf(api.UISettingsGroupList{}),
		isNamespaced:      true,
		get:               getFn,
		list:              listFn,
		resourceName:      "UISettingsGroup",
		converter:         UISettingsGroupConverter{},
		licenseCache:      opts.LicenseCache,
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

// TODO(dimitrin): Deprecated functionality meant to convert to/from libcacilo and apiserver
// resource structures. Remove converter once resourceStore has been refactored.
type UISettingsGroupConverter struct {
}

// TODO(dimitrin): Deprecated functionality meant to convert to/from libcacilo and apiserver
// resource structures. Remove converter once resourceStore has been refactored.
func (gc UISettingsGroupConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	uiSettingsGroup := aapiObj.(*api.UISettingsGroup)
	return uiSettingsGroup
}

// TODO(dimitrin): Deprecated functionality meant to convert to/from libcacilo and apiserver
// resource structures. Remove converter once resourceStore has been refactored.
func (gc UISettingsGroupConverter) convertToAAPI(apiObject resourceObject, aapiObj runtime.Object) {
	inUISettingsGroup := apiObject.(*api.UISettingsGroup)
	outUISettingsGroup := aapiObj.(*api.UISettingsGroup)
	err := copier.Copy(outUISettingsGroup, inUISettingsGroup)
	if err != nil {
		log.WithError(err).Errorf("failed to copy type %v.", reflect.TypeOf(inUISettingsGroup))
	}
}

// TODO(dimitrin): Deprecated functionality meant to convert to/from libcacilo and apiserver
// resource structures. Remove converter once resourceStore has been refactored.
func (gc UISettingsGroupConverter) convertToAAPIList(apiListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	aapiUISettingsGroupList := aapiListObj.(*api.UISettingsGroupList)
	if apiListObject == nil {
		aapiUISettingsGroupList.Items = []api.UISettingsGroup{}
		return
	}
	for _, item := range aapiUISettingsGroupList.Items {
		aapiUISettingsGroup := api.UISettingsGroup{}
		gc.convertToAAPI(&item, &aapiUISettingsGroup)
		if matched, err := pred.Matches(&aapiUISettingsGroup); err == nil && matched {
			aapiUISettingsGroupList.Items = append(aapiUISettingsGroupList.Items, aapiUISettingsGroup)
		}
	}
}
