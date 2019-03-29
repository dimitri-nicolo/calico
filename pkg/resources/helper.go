// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package resources

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

const (
	v1               = "v1"
	v1beta1          = "v1beta1"
	v3               = "v3"
	grpProjectcalico = "projectcalico.org"
	grpK8sNetworking = "networking.k8s.io"
	grpExtensions    = "extensions"
)

var (
	ResourceTypeGlobalNetworkPolicies = schema.GroupVersionKind{Group: grpProjectcalico, Version: v3, Kind: apiv3.KindGlobalNetworkPolicy}
	ResourceTypeGlobalNetworkSets     = schema.GroupVersionKind{Group: grpProjectcalico, Version: v3, Kind: apiv3.KindGlobalNetworkSet}
	ResourceTypeHostEndpoints         = schema.GroupVersionKind{Group: grpProjectcalico, Version: v3, Kind: apiv3.KindHostEndpoint}
	ResourceTypeNetworkPolicies       = schema.GroupVersionKind{Group: grpProjectcalico, Version: v3, Kind: apiv3.KindNetworkPolicy}
	ResourceTypeTiers                 = schema.GroupVersionKind{Group: grpProjectcalico, Version: v3, Kind: apiv3.KindTier}
	ResourceTypeServices              = schema.GroupVersionKind{Version: v1, Kind: "Service"}
	ResourceTypeEndpoints             = schema.GroupVersionKind{Version: v1, Kind: "Endpoints"}
	ResourceTypeNamespaces            = schema.GroupVersionKind{Version: v1, Kind: "Namespace"}
	ResourceTypeK8sNetworkPolicies    = schema.GroupVersionKind{Group: grpK8sNetworking, Version: v1, Kind: "NetworkPolicy"}
	ResourceTypeK8sNetworkPoliciesDep = schema.GroupVersionKind{Group: grpExtensions, Version: v1beta1, Kind: "NetworkPolicy"}
	ResourceTypePods                  = schema.GroupVersionKind{Version: v1, Kind: "Pod"}
	ResourceTypeServiceAccounts       = schema.GroupVersionKind{Version: v1, Kind: "ServiceAccount"}
)

type ResourceHelper interface {
	GroupVersionKind() schema.GroupVersionKind
	NewResource() Resource
	NewResourceList() ResourceList
}

// GetGroupVersionKind extracts the group version kind from the resource unless
//   it is using a deprecated apiVersion
func GetGroupVersionKind(res Resource) schema.GroupVersionKind {
	gvk := res.GetObjectKind().GroupVersionKind()
	if gvk == ResourceTypeK8sNetworkPoliciesDep {
		gvk = ResourceTypeK8sNetworkPolicies
	}
	return gvk
}

// GetResourceHelper returns the requested ResourceHelper, or nil if not supported.
func GetResourceHelper(gvk schema.GroupVersionKind) ResourceHelper {
	if gvk == ResourceTypeK8sNetworkPoliciesDep {
		return resourceHelpersMap[ResourceTypeK8sNetworkPolicies]
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
func NewResource(gvk schema.GroupVersionKind) Resource {
	helper := resourceHelpersMap[gvk]
	if helper == nil {
		return nil
	}
	return helper.NewResource()
}

// NewResourceList returns a new instance of the requested resource type list.
func NewResourceList(gvk schema.GroupVersionKind) ResourceList {
	helper := resourceHelpersMap[gvk]
	if helper == nil {
		return nil
	}
	return helper.NewResourceList()
}

type resourceHelper struct {
	kind         schema.GroupVersionKind
	resource     Resource
	resourceList ResourceList
}

func (h *resourceHelper) GroupVersionKind() schema.GroupVersionKind {
	return h.kind
}

func (h *resourceHelper) NewResource() Resource {
	return h.resource.DeepCopyObject().(Resource)
}

func (h *resourceHelper) NewResourceList() ResourceList {
	return h.resourceList.DeepCopyObject().(ResourceList)
}

var (
	resourceHelpersMap = map[schema.GroupVersionKind]ResourceHelper{}
	resourceHelpers    = []ResourceHelper{
		&resourceHelper{
			ResourceTypePods, &corev1.Pod{}, &corev1.PodList{},
		},
		&resourceHelper{
			ResourceTypeNamespaces, &corev1.Namespace{}, &corev1.NamespaceList{},
		},
		&resourceHelper{
			ResourceTypeServiceAccounts, &corev1.ServiceAccount{}, &corev1.ServiceAccountList{},
		},
		&resourceHelper{
			ResourceTypeEndpoints, &corev1.Endpoints{}, &corev1.EndpointsList{},
		},
		&resourceHelper{
			ResourceTypeServices, &corev1.Service{}, &corev1.ServiceList{},
		},
		&resourceHelper{
			ResourceTypeK8sNetworkPolicies, &networkingv1.NetworkPolicy{}, &networkingv1.NetworkPolicyList{},
		},
		&resourceHelper{
			ResourceTypeTiers, &apiv3.Tier{}, &apiv3.TierList{},
		},
		&resourceHelper{
			ResourceTypeHostEndpoints, &apiv3.HostEndpoint{}, &apiv3.HostEndpointList{},
		},
		&resourceHelper{
			ResourceTypeGlobalNetworkSets, &apiv3.GlobalNetworkSet{}, &apiv3.GlobalNetworkSetList{},
		},
		&resourceHelper{
			ResourceTypeNetworkPolicies, &apiv3.NetworkPolicy{}, &apiv3.NetworkPolicyList{},
		},
		&resourceHelper{
			ResourceTypeGlobalNetworkPolicies, &apiv3.GlobalNetworkPolicy{}, &apiv3.GlobalNetworkPolicyList{},
		},
	}
)

func init() {
	for _, rh := range resourceHelpers {
		resourceHelpersMap[rh.GroupVersionKind()] = rh
	}
}
