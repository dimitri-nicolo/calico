// Copyright (c) 2019 Tigera, Inc. SelectAll rights reserved.
package testutils

import (
	networkingv1 "k8s.io/api/networking/v1"
)

func getK8sNets(n Net) []networkingv1.NetworkPolicyPeer {
	var nets []networkingv1.NetworkPolicyPeer
	if n&Public != 0 {
		nets = append(nets, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: "1.1.1.1/32",
			},
		})
	}
	if n&Private != 0 {
		nets = append(nets, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: "10.0.100.0/24",
			},
		})
	}
	return nets
}

func K8sIngressRuleNets(a Action, e Entity, n Net) networkingv1.NetworkPolicyIngressRule {
	if a != Allow {
		panic("Kubernetes rules can only be allow")
	}
	if e&Destination != 0 {
		panic("Kubernetes ingress rules only support Source")
	}

	return networkingv1.NetworkPolicyIngressRule{
		From: getK8sNets(n),
	}
}

func K8sEgressRuleNets(a Action, e Entity, n Net) networkingv1.NetworkPolicyEgressRule {
	if a != Allow {
		panic("Kubernetes rules can only be allow")
	}
	if e&Source != 0 {
		panic("Kubernetes egress rules only support Destination")
	}

	return networkingv1.NetworkPolicyEgressRule{
		To: getK8sNets(n),
	}
}

func K8sIngressRuleSelectors(a Action, e Entity, sel Selector, nsSel Selector) networkingv1.NetworkPolicyIngressRule {
	if a != Allow {
		panic("Kubernetes rules can only be allow")
	}
	if e&Destination != 0 {
		panic("Kubernetes ingress rules only support Source")
	}
	return networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{{
			PodSelector:       selectorByteToK8sSelector(sel),
			NamespaceSelector: selectorByteToK8sSelector(nsSel),
		}},
	}
}

func K8sEgressRuleSelectors(a Action, e Entity, sel Selector, nsSel Selector) networkingv1.NetworkPolicyEgressRule {
	if a != Allow {
		panic("Kubernetes rules can only be allow")
	}
	if e&Source != 0 {
		panic("Kubernetes egress rules only support Destination")
	}
	return networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{{
			PodSelector:       selectorByteToK8sSelector(sel),
			NamespaceSelector: selectorByteToK8sSelector(nsSel),
		}},
	}
}
