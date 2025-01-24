package utils

import (
	"errors"
	"regexp"
	"strings"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/network-policy-api/apis/v1alpha1"

	"github.com/projectcalico/calico/libcalico-go/lib/names"
	"github.com/projectcalico/calico/queryserver/pkg/querycache/api"
)

// BuildSubstringRegexMatcher creates a regex from a list to help with faster substring searching.
//
// the list should contain at least one value. If the list is empty it fails to create regex pattern.
func BuildSubstringRegexMatcher(list []string) (*regexp.Regexp, error) {
	if len(list) > 0 {
		regexPattern := strings.Join(list, "|")
		epListRegex, err := regexp.Compile(regexPattern)
		if err != nil {
			return nil, err
		}

		return epListRegex, nil
	}
	return nil, errors.New("vague input: cannot create regex pattern from empty list")
}

// GetActualResourceAndTierFromCachedPolicyForRBAC returns the proper resource version/kind and tier for non-tiered
// policies. Kubernetes, StageKubernetes, and AdminNetwork policies are technically non-tiered specially when it comes
// to checking RBAC against them. Before checking authorization to these policies we need to get the correct tier and
// resource type values.
func GetActualResourceAndTierFromCachedPolicyForRBAC(p api.Policy) (api.Resource, string) {
	resource := p.GetResource()
	tier := p.GetTier()
	resourceName := p.GetResource().GetObjectMeta().GetName()
	if strings.HasPrefix(resourceName, names.K8sNetworkPolicyNamePrefix) {
		resource = &v1.NetworkPolicy{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NetworkPolicy",
				APIVersion: "networking.k8s.io/v1",
			},
		}
		tier = ""
	} else if strings.HasPrefix(resourceName, "staged:"+names.K8sNetworkPolicyNamePrefix) {
		resource = &apiv3.StagedKubernetesNetworkPolicy{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StagedKubernetesNetworkPolicy",
				APIVersion: "projectcalico.org/v3",
			},
		}
		tier = ""
	} else if strings.HasPrefix(resourceName, names.K8sAdminNetworkPolicyNamePrefix) {
		resource = &v1alpha1.AdminNetworkPolicy{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AdminNetworkPolicy",
				APIVersion: "policy.networking.k8s.io/v1alpha1",
			},
		}
		tier = ""
	}

	return resource, tier
}
