// Copyright (c) 2017-2024 Tigera, Inc. All rights reserved.

package calico

import (
	"context"
	"reflect"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
	"github.com/projectcalico/calico/licensing/client/features"
)

// NewGlobalNetworkPolicyStorage creates a new libcalico-based storage.Interface implementation for GlobalNetworkPolicies
func NewGlobalNetworkPolicyStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.GlobalNetworkPolicy)
		return c.GlobalNetworkPolicies().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.GlobalNetworkPolicy)
		return c.GlobalNetworkPolicies().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.GlobalNetworkPolicies().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.GlobalNetworkPolicies().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalNetworkPolicies().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.GlobalNetworkPolicies().Watch(ctx, olo)
	}
	hasRestrictionsFn := func(obj resourceObject) bool {
		res := obj.(*v3.GlobalNetworkPolicy)
		return !opts.LicenseMonitor.GetFeatureStatus(features.EgressAccessControl) && rulesHaveDNSDomain(res.Spec.Egress)
	}

	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.GlobalNetworkPolicy{}),
		aapiListType:      reflect.TypeOf(v3.GlobalNetworkPolicyList{}),
		libCalicoType:     reflect.TypeOf(v3.GlobalNetworkPolicy{}),
		libCalicoListType: reflect.TypeOf(v3.GlobalNetworkPolicyList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "GlobalNetworkPolicy",
		converter:         GlobalNetworkPolicyConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type GlobalNetworkPolicyConverter struct {
}

func (gc GlobalNetworkPolicyConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiGlobalNetworkPolicy := aapiObj.(*v3.GlobalNetworkPolicy)
	lcgGlobalNetworkPolicy := &v3.GlobalNetworkPolicy{}
	lcgGlobalNetworkPolicy.TypeMeta = aapiGlobalNetworkPolicy.TypeMeta
	lcgGlobalNetworkPolicy.ObjectMeta = aapiGlobalNetworkPolicy.ObjectMeta
	lcgGlobalNetworkPolicy.Kind = v3.KindGlobalNetworkPolicy
	lcgGlobalNetworkPolicy.APIVersion = v3.GroupVersionCurrent
	lcgGlobalNetworkPolicy.Spec = aapiGlobalNetworkPolicy.Spec
	return lcgGlobalNetworkPolicy
}

func (gc GlobalNetworkPolicyConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgGlobalNetworkPolicy := libcalicoObject.(*v3.GlobalNetworkPolicy)
	aapiGlobalNetworkPolicy := aapiObj.(*v3.GlobalNetworkPolicy)
	aapiGlobalNetworkPolicy.Spec = lcgGlobalNetworkPolicy.Spec
	// Default the tier field if not specified
	if aapiGlobalNetworkPolicy.Spec.Tier == "" {
		aapiGlobalNetworkPolicy.Spec.Tier = "default"
	}
	aapiGlobalNetworkPolicy.TypeMeta = lcgGlobalNetworkPolicy.TypeMeta
	aapiGlobalNetworkPolicy.ObjectMeta = lcgGlobalNetworkPolicy.ObjectMeta
	// Workflows associated with label "projectcalico.org/tier" should be deprecated thereafter.
	if aapiGlobalNetworkPolicy.Labels == nil {
		aapiGlobalNetworkPolicy.Labels = make(map[string]string)
	}
	aapiGlobalNetworkPolicy.Labels["projectcalico.org/tier"] = aapiGlobalNetworkPolicy.Spec.Tier
}

func (gc GlobalNetworkPolicyConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgGlobalNetworkPolicyList := libcalicoListObject.(*v3.GlobalNetworkPolicyList)
	aapiGlobalNetworkPolicyList := aapiListObj.(*v3.GlobalNetworkPolicyList)
	if libcalicoListObject == nil {
		aapiGlobalNetworkPolicyList.Items = []v3.GlobalNetworkPolicy{}
		return
	}
	aapiGlobalNetworkPolicyList.TypeMeta = lcgGlobalNetworkPolicyList.TypeMeta
	aapiGlobalNetworkPolicyList.ListMeta = lcgGlobalNetworkPolicyList.ListMeta
	for _, item := range lcgGlobalNetworkPolicyList.Items {
		aapiGlobalNetworkPolicy := v3.GlobalNetworkPolicy{}
		gc.convertToAAPI(&item, &aapiGlobalNetworkPolicy)
		if matched, err := pred.Matches(&aapiGlobalNetworkPolicy); err == nil && matched {
			aapiGlobalNetworkPolicyList.Items = append(aapiGlobalNetworkPolicyList.Items, aapiGlobalNetworkPolicy)
		}
	}
}
