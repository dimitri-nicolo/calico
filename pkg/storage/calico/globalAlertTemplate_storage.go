// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"

	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
)

// NewGlobalAlertTemplateStorage creates a new libcalico-based storage.Interface implementation for GlobalAlertTemplates
func NewGlobalAlertTemplateStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalAlertTemplate)
		return c.GlobalAlertTemplates().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.GlobalAlertTemplate)
		return c.GlobalAlertTemplates().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalAlertTemplates().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.GlobalAlertTemplates().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalAlertTemplates().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalAlertTemplates().Watch(ctx, olo)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.GlobalAlertTemplate{}),
		aapiListType:      reflect.TypeOf(aapi.GlobalAlertTemplateList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.GlobalAlertTemplate{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.GlobalAlertTemplateList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "GlobalAlertTemplate",
		converter:         GlobalAlertTemplateConverter{},
	}, func() {}
}

type GlobalAlertTemplateConverter struct {
}

func (gc GlobalAlertTemplateConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalAlertTemplate := aapiObj.(*aapi.GlobalAlertTemplate)
	lcgGlobalAlertTemplate := &libcalicoapi.GlobalAlertTemplate{}
	lcgGlobalAlertTemplate.TypeMeta = aapiGlobalAlertTemplate.TypeMeta
	lcgGlobalAlertTemplate.ObjectMeta = aapiGlobalAlertTemplate.ObjectMeta
	lcgGlobalAlertTemplate.Spec = aapiGlobalAlertTemplate.Spec
	return lcgGlobalAlertTemplate
}

func (gc GlobalAlertTemplateConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgGlobalAlertTemplate := libcalicoObject.(*libcalicoapi.GlobalAlertTemplate)
	aapiGlobalAlertTemplate := aapiObj.(*aapi.GlobalAlertTemplate)
	aapiGlobalAlertTemplate.Spec = lcgGlobalAlertTemplate.Spec
	aapiGlobalAlertTemplate.TypeMeta = lcgGlobalAlertTemplate.TypeMeta
	aapiGlobalAlertTemplate.ObjectMeta = lcgGlobalAlertTemplate.ObjectMeta
}

func (gc GlobalAlertTemplateConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgGlobalAlertTemplateList := libcalicoListObject.(*libcalicoapi.GlobalAlertTemplateList)
	aapiGlobalAlertTemplateList := aapiListObj.(*aapi.GlobalAlertTemplateList)
	if libcalicoListObject == nil {
		aapiGlobalAlertTemplateList.Items = []aapi.GlobalAlertTemplate{}
		return
	}
	aapiGlobalAlertTemplateList.TypeMeta = lcgGlobalAlertTemplateList.TypeMeta
	aapiGlobalAlertTemplateList.ListMeta = lcgGlobalAlertTemplateList.ListMeta
	for _, item := range lcgGlobalAlertTemplateList.Items {
		aapiGlobalAlertTemplate := aapi.GlobalAlertTemplate{}
		gc.convertToAAPI(&item, &aapiGlobalAlertTemplate)
		if matched, err := pred.Matches(&aapiGlobalAlertTemplate); err == nil && matched {
			aapiGlobalAlertTemplateList.Items = append(aapiGlobalAlertTemplateList.Items, aapiGlobalAlertTemplate)
		}
	}
}
