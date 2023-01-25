package calico

import (
	"reflect"

	"golang.org/x/net/context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	features "github.com/projectcalico/calico/licensing/client/features"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// NewStagedGlobalNetworkPolicyStorage creates a new libcalico-based storage.Interface implementation for StagedGlobalNetworkPolicies
func NewStagedGlobalNetworkPolicyStorage(opts Options) (registry.DryRunnableStorage, factory.DestroyFunc) {
	c := CreateClientFromConfig()
	createFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.StagedGlobalNetworkPolicy)
		return c.StagedGlobalNetworkPolicies().Create(ctx, res, oso)
	}
	updateFn := func(ctx context.Context, c clientv3.Interface, obj resourceObject, opts clientOpts) (resourceObject, error) {
		oso := opts.(options.SetOptions)
		res := obj.(*v3.StagedGlobalNetworkPolicy)
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
	hasRestrictionsFn := func(obj resourceObject) bool {
		res := obj.(*v3.StagedGlobalNetworkPolicy)
		return !opts.LicenseMonitor.GetFeatureStatus(features.EgressAccessControl) && rulesHaveDNSDomain(res.Spec.Egress)
	}
	// TODO(doublek): Inject codec, client for nicer testing.
	dryRunnableStorage := registry.DryRunnableStorage{Storage: &resourceStore{
		client:            c,
		codec:             opts.RESTOptions.StorageConfig.Codec,
		versioner:         APIObjectVersioner{},
		aapiType:          reflect.TypeOf(v3.StagedGlobalNetworkPolicy{}),
		aapiListType:      reflect.TypeOf(v3.StagedGlobalNetworkPolicyList{}),
		libCalicoType:     reflect.TypeOf(v3.StagedGlobalNetworkPolicy{}),
		libCalicoListType: reflect.TypeOf(v3.StagedGlobalNetworkPolicyList{}),
		isNamespaced:      false,
		create:            createFn,
		update:            updateFn,
		get:               getFn,
		delete:            deleteFn,
		list:              listFn,
		watch:             watchFn,
		resourceName:      "StagedGlobalNetworkPolicy",
		converter:         StagedGlobalNetworkPolicyConverter{},
		hasRestrictions:   hasRestrictionsFn,
	}, Codec: opts.RESTOptions.StorageConfig.Codec}
	return dryRunnableStorage, func() {}
}

type StagedGlobalNetworkPolicyConverter struct {
}

func (gc StagedGlobalNetworkPolicyConverter) convertToLibcalico(aapiObj runtime.Object) resourceObject {
	aapiStagedGlobalNetworkPolicy := aapiObj.(*v3.StagedGlobalNetworkPolicy)
	lcgStagedGlobalNetworkPolicy := &v3.StagedGlobalNetworkPolicy{}
	lcgStagedGlobalNetworkPolicy.TypeMeta = aapiStagedGlobalNetworkPolicy.TypeMeta
	lcgStagedGlobalNetworkPolicy.ObjectMeta = aapiStagedGlobalNetworkPolicy.ObjectMeta
	lcgStagedGlobalNetworkPolicy.Kind = v3.KindStagedGlobalNetworkPolicy
	lcgStagedGlobalNetworkPolicy.APIVersion = v3.GroupVersionCurrent
	lcgStagedGlobalNetworkPolicy.Spec = aapiStagedGlobalNetworkPolicy.Spec
	return lcgStagedGlobalNetworkPolicy
}

func (gc StagedGlobalNetworkPolicyConverter) convertToAAPI(libcalicoObject resourceObject, aapiObj runtime.Object) {
	lcgStagedGlobalNetworkPolicy := libcalicoObject.(*v3.StagedGlobalNetworkPolicy)
	aapiStagedGlobalNetworkPolicy := aapiObj.(*v3.StagedGlobalNetworkPolicy)
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
	lcgStagedGlobalNetworkPolicyList := libcalicoListObject.(*v3.StagedGlobalNetworkPolicyList)
	aapiStagedGlobalNetworkPolicyList := aapiListObj.(*v3.StagedGlobalNetworkPolicyList)
	if libcalicoListObject == nil {
		aapiStagedGlobalNetworkPolicyList.Items = []v3.StagedGlobalNetworkPolicy{}
		return
	}
	aapiStagedGlobalNetworkPolicyList.TypeMeta = lcgStagedGlobalNetworkPolicyList.TypeMeta
	aapiStagedGlobalNetworkPolicyList.ListMeta = lcgStagedGlobalNetworkPolicyList.ListMeta
	for _, item := range lcgStagedGlobalNetworkPolicyList.Items {
		aapiStagedGlobalNetworkPolicy := v3.StagedGlobalNetworkPolicy{}
		gc.convertToAAPI(&item, &aapiStagedGlobalNetworkPolicy)
		if matched, err := pred.Matches(&aapiStagedGlobalNetworkPolicy); err == nil && matched {
			aapiStagedGlobalNetworkPolicyList.Items = append(aapiStagedGlobalNetworkPolicyList.Items, aapiStagedGlobalNetworkPolicy)
		}
	}
}
