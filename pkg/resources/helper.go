// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package resources

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

const (
	v1                      = "v1"
	grpVersionProjectcalico = "projectcalico.org/v3"
	grpVersionK8sNetworking = "networking.k8s.io/v1"
	grpVersionExtensions    = "extensions/v1beta1"
)

var (
	TypeCalicoGlobalNetworkPolicies  = metav1.TypeMeta{APIVersion: grpVersionProjectcalico, Kind: apiv3.KindGlobalNetworkPolicy}
	TypeCalicoGlobalNetworkSets      = metav1.TypeMeta{APIVersion: grpVersionProjectcalico, Kind: apiv3.KindGlobalNetworkSet}
	TypeCalicoHostEndpoints          = metav1.TypeMeta{APIVersion: grpVersionProjectcalico, Kind: apiv3.KindHostEndpoint}
	TypeCalicoNetworkPolicies        = metav1.TypeMeta{APIVersion: grpVersionProjectcalico, Kind: apiv3.KindNetworkPolicy}
	TypeCalicoTiers                  = metav1.TypeMeta{APIVersion: grpVersionProjectcalico, Kind: apiv3.KindTier}
	TypeK8sServices                  = metav1.TypeMeta{APIVersion: v1, Kind: "Service"}
	TypeK8sEndpoints                 = metav1.TypeMeta{APIVersion: v1, Kind: "Endpoints"}
	TypeK8sNamespaces                = metav1.TypeMeta{APIVersion: v1, Kind: "Namespace"}
	TypeK8sNetworkPolicies           = metav1.TypeMeta{APIVersion: grpVersionK8sNetworking, Kind: "NetworkPolicy"}
	TypeK8sNetworkPoliciesExtensions = metav1.TypeMeta{APIVersion: grpVersionExtensions, Kind: "NetworkPolicy"}
	TypeK8sPods                      = metav1.TypeMeta{APIVersion: v1, Kind: "Pod"}
	TypeK8sServiceAccounts           = metav1.TypeMeta{APIVersion: v1, Kind: "ServiceAccount"}
)

type ResourceHelper interface {
	TypeMeta() metav1.TypeMeta
	NewResource() Resource
	NewResourceList() ResourceList
}

// GetTypeMeta extracts the group version kind from the resource unless
//   it is using a deprecated apiVersion
func GetTypeMeta(res Resource) metav1.TypeMeta {
	gvk := res.GetObjectKind().GroupVersionKind()
	tm := metav1.TypeMeta{Kind: gvk.Kind, APIVersion: gvk.GroupVersion().String()}
	if tm == TypeK8sNetworkPoliciesExtensions {
		tm = TypeK8sNetworkPolicies
	}
	return tm
}

// GetResourceHelper returns the requested ResourceHelper, or nil if not supported.
func GetResourceHelper(gvk metav1.TypeMeta) ResourceHelper {
	if gvk == TypeK8sNetworkPoliciesExtensions {
		return resourceHelpersMap[TypeK8sNetworkPolicies]
	}
	return resourceHelpersMap[gvk]
}

// GetAllResourceHelpers returns a list of all supported ResourceHelpers.
func GetAllResourceHelpers() []ResourceHelper {
	rhs := make([]ResourceHelper, len(resourceHelpers))
	copy(rhs, resourceHelpers)
	return rhs
}

// NewResource returns a new instance of the requested resource type.
func NewResource(gvk metav1.TypeMeta) Resource {
	helper := resourceHelpersMap[gvk]
	if helper == nil {
		return nil
	}
	return helper.NewResource()
}

// NewResourceList returns a new instance of the requested resource type list.
func NewResourceList(gvk metav1.TypeMeta) ResourceList {
	helper := resourceHelpersMap[gvk]
	if helper == nil {
		return nil
	}
	return helper.NewResourceList()
}

type resourceHelper struct {
	kind         metav1.TypeMeta
	resource     Resource
	resourceList ResourceList
}

func (h *resourceHelper) TypeMeta() metav1.TypeMeta {
	return h.kind
}

func (h *resourceHelper) NewResource() Resource {
	return h.resource.DeepCopyObject().(Resource)
}

func (h *resourceHelper) NewResourceList() ResourceList {
	return h.resourceList.DeepCopyObject().(ResourceList)
}

//TODO(rlb): Need to normalize the output from the parsed data. The xref cache
var (
	resourceHelpersMap = map[metav1.TypeMeta]ResourceHelper{}
	resourceHelpers    = []ResourceHelper{
		&resourceHelper{
			TypeK8sPods, &corev1.Pod{}, &corev1.PodList{},
		},
		&resourceHelper{
			TypeK8sNamespaces, &corev1.Namespace{}, &corev1.NamespaceList{},
		},
		&resourceHelper{
			TypeK8sServiceAccounts, &corev1.ServiceAccount{}, &corev1.ServiceAccountList{},
		},
		&resourceHelper{
			TypeK8sEndpoints, &corev1.Endpoints{}, &corev1.EndpointsList{},
		},
		&resourceHelper{
			TypeK8sServices, &corev1.Service{}, &corev1.ServiceList{},
		},
		&resourceHelper{
			TypeK8sNetworkPolicies, &networkingv1.NetworkPolicy{}, &networkingv1.NetworkPolicyList{},
		},
		&resourceHelper{
			TypeCalicoTiers, &apiv3.Tier{}, &apiv3.TierList{},
		},
		&resourceHelper{
			TypeCalicoHostEndpoints, &apiv3.HostEndpoint{}, &apiv3.HostEndpointList{},
		},
		&resourceHelper{
			TypeCalicoGlobalNetworkSets, &apiv3.GlobalNetworkSet{}, &apiv3.GlobalNetworkSetList{},
		},
		&resourceHelper{
			TypeCalicoNetworkPolicies, &apiv3.NetworkPolicy{}, &apiv3.NetworkPolicyList{},
		},
		&resourceHelper{
			TypeCalicoGlobalNetworkPolicies, &apiv3.GlobalNetworkPolicy{}, &apiv3.GlobalNetworkPolicyList{},
		},
	}
)

func init() {
	for _, rh := range resourceHelpers {
		resourceHelpersMap[rh.TypeMeta()] = rh
	}
}
