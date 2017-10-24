// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apiv2"
	"github.com/projectcalico/libcalico-go/lib/clientv2"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

// NewNetworkPolicyStorage creates a new libcalico-based storage.Interface implementation for Policy
func NewNetworkPolicyStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	create := func(ctx context.Context, c clientv2.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.NetworkPolicy)
		return c.NetworkPolicies().Create(ctx, res, oso)
	}
	update := func(ctx context.Context, c clientv2.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.NetworkPolicy)
		return c.NetworkPolicies().Update(ctx, res, oso)
	}
	get := func(ctx context.Context, c clientv2.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.NetworkPolicies().Get(ctx, ns, name, ogo)
	}
	delete := func(ctx context.Context, c clientv2.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.NetworkPolicies().Delete(ctx, ns, name, odo)
	}
	list := func(ctx context.Context, c clientv2.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.NetworkPolicies().List(ctx, olo)
	}
	watch := func(ctx context.Context, c clientv2.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.NetworkPolicies().Watch(ctx, olo)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.NetworkPolicy{}),
		aapiListType:      reflect.TypeOf(aapi.NetworkPolicyList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.NetworkPolicy{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.NetworkPolicyList{}),
		isNamespaced:      true,
		create:            create,
		update:            update,
		get:               get,
		delete:            delete,
		list:              list,
		watch:             watch,
		resourceName:      "NetworkPolicy",
		converter:         NetworkPolicyConverter{},
	}, func() {}
}

type NetworkPolicyConverter struct {
}

func (rc NetworkPolicyConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiPolicy := aapiObj.(*aapi.NetworkPolicy)
	lcgPolicy := &libcalicoapi.NetworkPolicy{}
	lcgPolicy.TypeMeta = aapiPolicy.TypeMeta
	lcgPolicy.ObjectMeta = aapiPolicy.ObjectMeta
	lcgPolicy.Spec = aapiPolicy.Spec
	return lcgPolicy
}

func (rc NetworkPolicyConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgPolicy := libcalicoObject.(*libcalicoapi.NetworkPolicy)
	aapiPolicy := aapiObj.(*aapi.NetworkPolicy)
	aapiPolicy.Spec = lcgPolicy.Spec
	// Tier field maybe left blank when policy created vi OS libcalico.
	// Initialize it to default in that case to make work with field selector.
	if aapiPolicy.Spec.Tier == "" {
		aapiPolicy.Spec.Tier = "default"
	}
	aapiPolicy.TypeMeta = lcgPolicy.TypeMeta
	aapiPolicy.ObjectMeta = lcgPolicy.ObjectMeta
}

func (rc NetworkPolicyConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, filterFunc storage.FilterFunc) {
	lcgPolicyList := libcalicoListObject.(*libcalicoapi.NetworkPolicyList)
	aapiPolicyList := aapiListObj.(*aapi.NetworkPolicyList)
	if libcalicoListObject == nil {
		aapiPolicyList.Items = []aapi.NetworkPolicy{}
		return
	}
	aapiPolicyList.TypeMeta = lcgPolicyList.TypeMeta
	aapiPolicyList.ListMeta = lcgPolicyList.ListMeta
	for _, item := range lcgPolicyList.Items {
		aapiPolicy := aapi.NetworkPolicy{}
		rc.convertToAAPI(&item, &aapiPolicy)
		if filterFunc != nil && filterFunc(&aapiPolicy) {
			aapiPolicyList.Items = append(aapiPolicyList.Items, aapiPolicy)
		}
	}
}
