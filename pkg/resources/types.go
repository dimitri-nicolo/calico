package resources

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

var (
	ResourceTypeNamespaces            = metav1.TypeMeta{APIVersion: "v1", Kind: "namespaces"}
	ResourceTypeServiceAccounts       = metav1.TypeMeta{APIVersion: "v1", Kind: "serviceaccounts"}
	ResourceTypePods                  = metav1.TypeMeta{APIVersion: "v1", Kind: "pods"}
	ResourceTypeEndpoints             = metav1.TypeMeta{APIVersion: "v1", Kind: "endpoints"}
	ResourceTypeServices              = metav1.TypeMeta{APIVersion: "v1", Kind: "services"}
	ResourceTypeK8sNetworkPolicies    = metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "networkpolicies"}
	ResourceTypeHostEndpoints         = metav1.TypeMeta{APIVersion: "projectcalico.org/v3", Kind: "hostendpoints"}
	ResourceTypeGlobalNetworkSets     = metav1.TypeMeta{APIVersion: "projectcalico.org/v3", Kind: "globalnetworksets"}
	ResourceTypeNetworkPolicies       = metav1.TypeMeta{APIVersion: "projectcalico.org/v3", Kind: "networkpolicies"}
	ResourceTypeGlobalNetworkPolicies = metav1.TypeMeta{APIVersion: "projectcalico.org/v3", Kind: "globalnetworkpolicies"}
)

// GetResourceHelper returns the requested ResourceHelper, or nil if not supported.
func GetResourceHelper(tm metav1.TypeMeta) ResourceHelper {
	return resourceHelpersMap[tm]
}

func GetAllResourceHelpers() []ResourceHelper {
	rhs := make([]ResourceHelper, len(resourceHelpers))
	copy(rhs, resourceHelpers)
	return rhs
}

type resourceHelper struct {
	typeMeta     metav1.TypeMeta
	resource     Resource
	resourceList ResourceList
}

var (
	resourceHelpersMap = map[metav1.TypeMeta]ResourceHelper{}
	resourceHelpers    = []ResourceHelper{
		resourceHelper{
			ResourceTypePods, &corev1.Pod{}, &corev1.PodList{},
		},
		resourceHelper{
			ResourceTypeNamespaces, &corev1.Namespace{}, &corev1.NamespaceList{},
		},
		resourceHelper{
			ResourceTypeServiceAccounts, &corev1.ServiceAccount{}, &corev1.ServiceAccountList{},
		},
		resourceHelper{
			ResourceTypeEndpoints, &corev1.Endpoints{}, &corev1.EndpointsList{},
		},
		resourceHelper{
			ResourceTypeServices, &corev1.Service{}, &corev1.ServiceList{},
		},
		resourceHelper{
			ResourceTypeK8sNetworkPolicies, &networkingv1.NetworkPolicy{}, &networkingv1.NetworkPolicyList{},
		},
		resourceHelper{
			ResourceTypeHostEndpoints, &apiv3.HostEndpoint{}, &apiv3.HostEndpointList{},
		},
		resourceHelper{
			ResourceTypeGlobalNetworkSets, &apiv3.GlobalNetworkSet{}, &apiv3.GlobalNetworkSetList{},
		},
		resourceHelper{
			ResourceTypeNetworkPolicies, &apiv3.NetworkPolicy{}, &apiv3.NetworkPolicyList{},
		},
		resourceHelper{
			ResourceTypeGlobalNetworkPolicies, &apiv3.GlobalNetworkPolicy{}, &apiv3.GlobalNetworkPolicyList{},
		},
	}
)

func init() {
	// Populate the resource helpers map.
	for _, rh := range resourceHelpers {
		resourceHelpersMap[rh.TypeMeta()] = rh
	}
}

func (r resourceHelper) TypeMeta() metav1.TypeMeta {
	return r.typeMeta
}

// NewResource implements the ResourceHelper interface, and returns a new instance of the resource type.
func (r resourceHelper) NewResource() Resource {
	return r.resource.DeepCopyObject().(Resource)
}

// NewResourceList implements the ResourceHelper interface, and returns a new instance of the resource list type.
func (r resourceHelper) NewResourceList() ResourceList {
	return r.resourceList.DeepCopyObject().(ResourceList)
}

// NewResource returns a new instance of the requested resource type.
func NewResource(tm metav1.TypeMeta) Resource {
	return resourceHelpersMap[tm].NewResource()
}

// NewResourceList returns a new instance of the requested resource type list.
func NewResourceList(tm metav1.TypeMeta) ResourceList {
	return resourceHelpersMap[tm].NewResourceList()
}
