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

// NewStagedGlobalNetworkPolicyStorage creates a new libcalico-based storage.Interface implementation for StagedGlobalNetworkPolicies
func NewStagedGlobalNetworkPolicyStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	c := createClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.StagedGlobalNetworkPolicy)
		return c.StagedGlobalNetworkPolicies().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*libcalicoapi.StagedGlobalNetworkPolicy)
		return c.StagedGlobalNetworkPolicies().Update(ctx, res, oso)
	}
	getFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		ogo := opts.(options.GetOptions)
		return c.StagedGlobalNetworkPolicies().Get(ctx, name, ogo)
	}
	deleteFn := func(ctx context.Context, c clientv3.Interface, ns string, name string, opts clientOpts) (resourceObject, error) {
		odo := opts.(options.DeleteOptions)
		return c.StagedGlobalNetworkPolicies().Delete(ctx, name, odo)
	}
	listFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (resourceListObject, error) {
		olo := opts.(options.ListOptions)
		return c.StagedGlobalNetworkPolicies().List(ctx, olo)
	}
	watchFn := func(ctx context.Context, c clientv3.Interface, opts clientOpts) (watch.Interface, error) {
		olo := opts.(options.ListOptions)
		return c.StagedGlobalNetworkPolicies().Watch(ctx, olo)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	return &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         etcd.APIObjectVersioner{},
		aapiType:          reflect.TypeOf(aapi.StagedGlobalNetworkPolicy{}),
		aapiListType:      reflect.TypeOf(aapi.StagedGlobalNetworkPolicyList{}),
		libCalicoType:     reflect.TypeOf(libcalicoapi.StagedGlobalNetworkPolicy{}),
		libCalicoListType: reflect.TypeOf(libcalicoapi.StagedGlobalNetworkPolicyList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "StagedGlobalNetworkPolicy",
		converter:         StagedGlobalNetworkPolicyConverter{},
	}, func() {}
}

type StagedGlobalNetworkPolicyConverter struct {
}

func (gc StagedGlobalNetworkPolicyConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiStagedGlobalNetworkPolicy := aapiObj.(*aapi.StagedGlobalNetworkPolicy)
	lcgStagedGlobalNetworkPolicy := &libcalicoapi.StagedGlobalNetworkPolicy{}
	lcgStagedGlobalNetworkPolicy.TypeMeta = aapiStagedGlobalNetworkPolicy.TypeMeta
	lcgStagedGlobalNetworkPolicy.ObjectMeta = aapiStagedGlobalNetworkPolicy.ObjectMeta
	lcgStagedGlobalNetworkPolicy.Spec = aapiStagedGlobalNetworkPolicy.Spec
	return lcgStagedGlobalNetworkPolicy
}

func (gc StagedGlobalNetworkPolicyConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgStagedGlobalNetworkPolicy := libcalicoObject.(*libcalicoapi.StagedGlobalNetworkPolicy)
	aapiStagedGlobalNetworkPolicy := aapiObj.(*aapi.StagedGlobalNetworkPolicy)
	aapiStagedGlobalNetworkPolicy.Spec = lcgStagedGlobalNetworkPolicy.Spec
	// Tier field maybe left blank when policy created vi OS libcalico.
	// Initialize it to default in that case to make work with field selector.
	if aapiStagedGlobalNetworkPolicy.Spec.Tier == "" {
		aapiStagedGlobalNetworkPolicy.Spec.Tier = "default"
	}
	aapiStagedGlobalNetworkPolicy.TypeMeta = lcgStagedGlobalNetworkPolicy.TypeMeta
	aapiStagedGlobalNetworkPolicy.ObjectMeta = lcgStagedGlobalNetworkPolicy.ObjectMeta
	// Labeling Purely for kubectl purposes. ex: kubectl get stagedglobalnetworkpolicies -l projectcalico.org/tier=net-sec
	// kubectl 1.9 should come out with support for field selector.
	// Workflows associated with label "projectcalico.org/tier" should be deprecated thereafter.
	if aapiStagedGlobalNetworkPolicy.Labels == nil {
		aapiStagedGlobalNetworkPolicy.Labels = make(map[string]string)
	}
	aapiStagedGlobalNetworkPolicy.Labels["projectcalico.org/tier"] = aapiStagedGlobalNetworkPolicy.Spec.Tier
}

func (gc StagedGlobalNetworkPolicyConverter) convertToAAPIList(libcalicoListObject resourceListObject, aapiListObj runtime.Object, pred storage.SelectionPredicate) {
	lcgStagedGlobalNetworkPolicyList := libcalicoListObject.(*libcalicoapi.StagedGlobalNetworkPolicyList)
	aapiStagedGlobalNetworkPolicyList := aapiListObj.(*aapi.StagedGlobalNetworkPolicyList)
	if libcalicoListObject == nil {
		aapiStagedGlobalNetworkPolicyList.Items = []aapi.StagedGlobalNetworkPolicy{}
		return
	}
	aapiStagedGlobalNetworkPolicyList.TypeMeta = lcgStagedGlobalNetworkPolicyList.TypeMeta
	aapiStagedGlobalNetworkPolicyList.ListMeta = lcgStagedGlobalNetworkPolicyList.ListMeta
	for _, item := range lcgStagedGlobalNetworkPolicyList.Items {
		aapiStagedGlobalNetworkPolicy := aapi.StagedGlobalNetworkPolicy{}
		gc.convertToAAPI(&item, &aapiStagedGlobalNetworkPolicy)
		if matched, err := pred.Matches(&aapiStagedGlobalNetworkPolicy); err == nil && matched {
			aapiStagedGlobalNetworkPolicyList.Items = append(aapiStagedGlobalNetworkPolicyList.Items, aapiStagedGlobalNetworkPolicy)
		}
	}
}
