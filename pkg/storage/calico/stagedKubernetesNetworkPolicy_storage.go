// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"golang.org/x/net/context"

	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// NewStagedKubernetesNetworkPolicyStorage creates a new libcalico-based storage.Interface implementation for Policy
func NewStagedKubernetesNetworkPolicyStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.StagedKubernetesNetworkPolicy)
		return c.StagedKubernetesNetworkPolicies().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.StagedKubernetesNetworkPolicy)
		return c.StagedKubernetesNetworkPolicies().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.StagedKubernetesNetworkPolicies().Get(ctx, ns, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.StagedKubernetesNetworkPolicies().Delete(ctx, ns, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.StagedKubernetesNetworkPolicies().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.StagedKubernetesNetworkPolicies().Watch(ctx, olo)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{&etcd.APIObjectVersioner{}},
		aapiType:          reflect.TypeOf(aapi.StagedKubernetesNetworkPolicy{}),
		aapiListType:      reflect.TypeOf(aapi.StagedKubernetesNetworkPolicyList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.StagedKubernetesNetworkPolicy{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.StagedKubernetesNetworkPolicyList{}),
		isNamespaced:      true,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "StagedKubernetesNetworkPolicy",
		converter:         StagedKubernetesNetworkPolicyConverter{},
	}, func() {}
}

type StagedKubernetesNetworkPolicyConverter struct {
}

func (rc StagedKubernetesNetworkPolicyConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiPolicy := aapiObj.(*aapi.StagedKubernetesNetworkPolicy)
	lcgPolicy := &libcalicoapi.StagedKubernetesNetworkPolicy{}
	lcgPolicy.TypeMeta = aapiPolicy.TypeMeta
	lcgPolicy.ObjectMeta = aapiPolicy.ObjectMeta
	lcgPolicy.Spec = aapiPolicy.Spec
	return lcgPolicy
}

func (rc StagedKubernetesNetworkPolicyConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgPolicy := libcalicoObject.(*libcalicoapi.StagedKubernetesNetworkPolicy)
	aapiPolicy := aapiObj.(*aapi.StagedKubernetesNetworkPolicy)
	aapiPolicy.Spec = lcgPolicy.Spec
	aapiPolicy.TypeMeta = lcgPolicy.TypeMeta
	aapiPolicy.ObjectMeta = lcgPolicy.ObjectMeta
	// Labeling Purely for kubectl purposes. ex: kubectl get globalnetworkpolicies -l projectcalico.org/tier=net-sec
	// kubectl 1.9 should come out with support for field selector.
	// Workflows associated with label "projectcalico.org/tier" should be deprecated thereafter.
	if aapiPolicy.Labels == nil {
		aapiPolicy.Labels = make(map[string]string)
	}
}

func (rc StagedKubernetesNetworkPolicyConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgPolicyList := libcalicoListObject.(*libcalicoapi.StagedKubernetesNetworkPolicyList)
	aapiPolicyList := aapiListObj.(*aapi.StagedKubernetesNetworkPolicyList)
	if libcalicoListObject == nil {
		aapiPolicyList.Items = []aapi.StagedKubernetesNetworkPolicy{}
		return
	}
	aapiPolicyList.TypeMeta = lcgPolicyList.TypeMeta
	aapiPolicyList.ListMeta = lcgPolicyList.ListMeta

	for _, item := range lcgPolicyList.Items {
		aapiPolicy := aapi.StagedKubernetesNetworkPolicy{}
		rc.convertToAAPI(&item, &aapiPolicy)
		aapiPolicyList.Items = append(aapiPolicyList.Items, aapiPolicy)
	}
}
