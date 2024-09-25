// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
package utils

import (
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/names"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

// DoExcludeStagedPolicy return true if staged policy should be filtered out
// Staged policies with StagedAction set to Delete are filtered out.
func DoExcludeStagedPolicy(uv3 *api.Update) bool {
	p3Key := uv3.Key.(model.ResourceKey)

	switch p3Key.Kind {
	case v3.KindStagedNetworkPolicy:
		if p3Value, ok := uv3.Value.(*v3.StagedNetworkPolicy); ok {
			if p3Value.Spec.StagedAction == v3.StagedActionDelete {
				return true
			}
		}
	case v3.KindStagedKubernetesNetworkPolicy:
		if p3Value, ok := uv3.Value.(*v3.StagedKubernetesNetworkPolicy); ok {
			if p3Value.Spec.StagedAction == v3.StagedActionDelete {
				return true
			}
		}
	case v3.KindStagedGlobalNetworkPolicy:
		if p3Value, ok := uv3.Value.(*v3.StagedGlobalNetworkPolicy); ok {
			if p3Value.Spec.StagedAction == v3.StagedActionDelete {
				return true
			}
		}
	}

	return false
}

func StagedToEnforcedConversion(uv1 *api.Update, uv3 *api.Update) {
	p1Key := uv1.Key.(model.PolicyKey)
	p3Key := uv3.Key.(model.ResourceKey)

	switch p3Key.Kind {
	case v3.KindStagedNetworkPolicy:
		p3Key.Kind = v3.KindNetworkPolicy
		p3Key.Name = model.PolicyNamePrefixStaged + p3Key.Name
		if p3Value, ok := uv3.Value.(*v3.StagedNetworkPolicy); ok {
			_, cp3Value := v3.ConvertStagedPolicyToEnforced(p3Value)
			cp3Value.Name = model.PolicyNamePrefixStaged + cp3Value.Name
			uv3.Value = cp3Value
		}
	case v3.KindStagedKubernetesNetworkPolicy:
		p3Key.Kind = v3.KindNetworkPolicy
		p3Key.Name = model.PolicyNamePrefixStaged + names.K8sNetworkPolicyNamePrefix + p3Key.Name
		if p3Value, ok := uv3.Value.(*v3.StagedKubernetesNetworkPolicy); ok {
			//From StagedKubernetesNetworkPolicy to networkingv1 NetworkPolicy
			_, v1NetworkPolicy := v3.ConvertStagedKubernetesPolicyToK8SEnforced(p3Value)
			c := conversion.NewConverter()
			//From networkingv1 NetworkkPolicy to calico model.KVPair
			kvPair, err := c.K8sNetworkPolicyToCalico(v1NetworkPolicy)
			if err == nil {
				if cp3Value, ok := kvPair.Value.(*v3.NetworkPolicy); ok {
					cp3Value.Name = model.PolicyNamePrefixStaged + cp3Value.Name
					uv3.Value = cp3Value
				}
			}
		}
	case v3.KindStagedGlobalNetworkPolicy:
		p3Key.Kind = v3.KindGlobalNetworkPolicy
		p3Key.Name = model.PolicyNamePrefixStaged + p3Key.Name
		if p3Value, ok := uv3.Value.(*v3.StagedGlobalNetworkPolicy); ok {
			_, cp3Value := v3.ConvertStagedGlobalPolicyToEnforced(p3Value)
			cp3Value.Name = model.PolicyNamePrefixStaged + cp3Value.Name
			uv3.Value = cp3Value
		}
	}

	uv1.Key = p1Key
	uv3.Key = p3Key
}
