// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package fv

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	prtypes "github.com/projectcalico/calico/policy-recommendation/pkg/types"
)

var (
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	protocolTCP = numorstring.ProtocolFromString("TCP")
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	protocolUDP = numorstring.ProtocolFromString("UDP")
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	timeAtStep1 = "2002-10-02T10:00:00-05:00"
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	timeAtStep2 = "2002-10-02T10:02:30-05:00"
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	timeAtStep3 = "2002-10-02T10:05:00-05:00"
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	timestampStep8Relearning = "2002-10-02T11:02:01-05:00"
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep1 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},

				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},

				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep2 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},

				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb",
				Namespace: "namespace2",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace2'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.docker.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep3 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb",
				Namespace: "namespace2",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace2'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.docker.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb",
				Namespace: "namespace5",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace5'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep4 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb",
				Namespace: "namespace2",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace2'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.docker.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb",
				Namespace: "namespace5",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace5'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep5 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb",
				Namespace: "namespace2",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace2'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.docker.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb",
				Namespace: "namespace5",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace5'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep6 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-76kle",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb",
				Namespace: "namespace2",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace2'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.docker.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb",
				Namespace: "namespace5",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace5'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep7 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-76kle",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace2-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace2-76kle",
				Namespace: "namespace2",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace2'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-76kle",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.docker.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace5-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace5-76kle",
				Namespace: "namespace5",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep3,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace5'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep8 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-76kle",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace2-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace2-76kle",
				Namespace: "namespace2",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace2'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-76kle",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							Domains: []string{"www.google.com", "www.tigera.io"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Domains: []string{"www.docker.com", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							Domains: []string{"www.calico.org"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace5-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace5-76kle",
				Namespace: "namespace5",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timestampStep8Relearning,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
							Domains: []string{"www.google.com", "www.projectcalico.org", "www.tigera.io", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timestampStep8Relearning,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source: v3.EntityRule{
							Nets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
						},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timestampStep8Relearning,
								"policyrecommendation.tigera.io/scope":       "Private",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source: v3.EntityRule{
							Nets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
						},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 101,
									MaxPort: 101,
								},
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timestampStep8Relearning,
								"policyrecommendation.tigera.io/scope":       "Private",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace5'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToDomainRecommendationsStep1AfterDeletingNamespace3 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							Domains: []string{"www.google.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Domains: []string{"www.google.com", "www.website.com"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Domains",
							},
						},
					},
				},

				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToServiceRecommendationsStep1 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Services: &v3.ServiceMatch{
								Name: "glb-service1a",
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "glb-service1a",
								"policyrecommendation.tigera.io/namespace":   "",
								"policyrecommendation.tigera.io/scope":       "Service",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace4'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace4",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace3'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace3",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Services: &v3.ServiceMatch{
								Name: "glb-service3a",
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "glb-service3a",
								"policyrecommendation.tigera.io/namespace":   "",
								"policyrecommendation.tigera.io/scope":       "Service",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace5'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace5",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace2'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace2",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedEgressToServiceRecommendationsStep2 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							Services: &v3.ServiceMatch{
								Name: "glb-service1a",
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "glb-service1a",
								"policyrecommendation.tigera.io/namespace":   "",
								"policyrecommendation.tigera.io/scope":       "Service",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace4'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace4",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace3'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/namespace":   "namespace3",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace2-xv5fb",
				Namespace: "namespace2",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
							},
							Services: &v3.ServiceMatch{
								Name: "glb-service3a",
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/name":        "glb-service3a",
								"policyrecommendation.tigera.io/namespace":   "",
								"policyrecommendation.tigera.io/scope":       "Service",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 97,
									MaxPort: 97,
								},
								{
									MinPort: 98,
									MaxPort: 98,
								},
								{
									MinPort: 99,
									MaxPort: 99,
								},
								{
									MinPort: 100,
									MaxPort: 100,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace5'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/namespace":   "namespace5",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace2'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							Services: &v3.ServiceMatch{
								Name: "glb-service3a",
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "glb-service3a",
								"policyrecommendation.tigera.io/namespace":   "",
								"policyrecommendation.tigera.io/scope":       "Service",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 99,
									MaxPort: 99,
								},
								{
									MinPort: 666,
									MaxPort: 666,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace5'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep2,
								"policyrecommendation.tigera.io/namespace":   "namespace5",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace2'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace2",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedNamespaceRecommendationsStep1 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace4'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace4",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace2'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace2",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 81,
									MaxPort: 81,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace3'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace3",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace3-xv5fb",
				Namespace: "namespace3",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 666,
									MaxPort: 666,
								},
								{
									MinPort: 667,
									MaxPort: 667,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace5'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace5",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolUDP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 9090,
									MaxPort: 9090,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace2'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace2",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace3'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
		prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace5-xv5fb",
				Namespace: "namespace5",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress:      []v3.Rule{},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 1,
									MaxPort: 1,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'namespace1'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/namespace":   "namespace1",
								"policyrecommendation.tigera.io/scope":       "Namespace",
								"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtoocol",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace5'",
				Types:    []v3.PolicyType{"Egress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedPrivateNetworkRecommendationsStep1 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source: v3.EntityRule{
							Nets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
						},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 5,
									MaxPort: 5,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Private",
							},
						},
					},
				},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 33,
									MaxPort: 33,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
								{
									MinPort: 90,
									MaxPort: 90,
								},
								{
									MinPort: 8080,
									MaxPort: 8080,
								},
								{
									MinPort: 8081,
									MaxPort: 8081,
								},
							},
							Nets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Private",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress", "Ingress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedPrivateNetworkRecommendationsStep2 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source: v3.EntityRule{
							Nets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
						},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 5,
									MaxPort: 5,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Private",
							},
						},
					},
				},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 33,
									MaxPort: 33,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
								{
									MinPort: 90,
									MaxPort: 90,
								},
								{
									MinPort: 8080,
									MaxPort: 8080,
								},
								{
									MinPort: 8081,
									MaxPort: 8081,
								},
							},
							Nets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Private",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress", "Ingress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedPrivateNetworkRecommendationsStep3 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-76kle",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source: v3.EntityRule{
							Nets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
						},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 5,
									MaxPort: 5,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Private",
							},
						},
					},
				},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 33,
									MaxPort: 33,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
								{
									MinPort: 90,
									MaxPort: 90,
								},
								{
									MinPort: 8080,
									MaxPort: 8080,
								},
								{
									MinPort: 8081,
									MaxPort: 8081,
								},
							},
							Nets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/scope":       "Private",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress", "Ingress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedNetworkSetRecommendationsStep1 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Learning",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source: v3.EntityRule{
							NamespaceSelector: "projectcalico.org/name == 'my-netset-namespace'",
							Selector:          "projectcalico.org/name == 'my-netset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 5,
									MaxPort: 5,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-netset",
								"policyrecommendation.tigera.io/namespace":   "my-netset-namespace",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
				},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 33,
									MaxPort: 33,
								},
								{
									MinPort: 90,
									MaxPort: 90,
								},
								{
									MinPort: 8080,
									MaxPort: 8080,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'my-netset-namespace'",
							Selector:          "projectcalico.org/name == 'my-netset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-netset",
								"policyrecommendation.tigera.io/namespace":   "my-netset-namespace",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
								{
									MinPort: 8081,
									MaxPort: 8081,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'global()'",
							Selector:          "projectcalico.org/name == 'my-globalnetset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-globalnetset",
								"policyrecommendation.tigera.io/namespace":   "global()",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
				},
				Selector: "projectcalico.org/namespace == 'namespace1'",
				Types:    []v3.PolicyType{"Egress", "Ingress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedNetworkSetRecommendationsStep2 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-xv5fb",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stabilizing",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source: v3.EntityRule{
							NamespaceSelector: "projectcalico.org/name == 'my-netset-namespace'",
							Selector:          "projectcalico.org/name == 'my-netset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 5,
									MaxPort: 5,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-netset",
								"policyrecommendation.tigera.io/namespace":   "my-netset-namespace",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
				},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 33,
									MaxPort: 33,
								},
								{
									MinPort: 90,
									MaxPort: 90,
								},
								{
									MinPort: 8080,
									MaxPort: 8080,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'my-netset-namespace'",
							Selector:          "projectcalico.org/name == 'my-netset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-netset",
								"policyrecommendation.tigera.io/namespace":   "my-netset-namespace",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
								{
									MinPort: 8081,
									MaxPort: 8081,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'global()'",
							Selector:          "projectcalico.org/name == 'my-globalnetset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-globalnetset",
								"policyrecommendation.tigera.io/namespace":   "global()",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
				}, Selector: "projectcalico.org/namespace == 'namespace1'",
				Types: []v3.PolicyType{"Egress", "Ingress"},
			},
		},
	}
	//lint:ignore U1000 Ignore unused function temporarily for future testing purposes.
	expectedNetworkSetRecommendationsStep3 = map[string]*v3.StagedNetworkPolicy{
		prtypes.PolicyRecommendationTierName + ".namespace1-76kle": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      prtypes.PolicyRecommendationTierName + ".namespace1-76kle",
				Namespace: "namespace1",
				Labels: map[string]string{
					"policyrecommendation.tigera.io/scope":  "namespace",
					"projectcalico.org/tier":                prtypes.PolicyRecommendationTierName,
					"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
					"projectcalico.org/spec.stagedAction":   "Learn",
				},
				Annotations: map[string]string{
					"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
					"policyrecommendation.tigera.io/status":      "Stable",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "projectcalico.org/v3",
						Kind:       "PolicyRecommendationScope",
						Name:       "default",
					},
				},
			},
			TypeMeta: metav1.TypeMeta{},
			Spec: v3.StagedNetworkPolicySpec{
				StagedAction: v3.StagedActionLearn,
				Tier:         prtypes.PolicyRecommendationTierName,
				Ingress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source: v3.EntityRule{
							NamespaceSelector: "projectcalico.org/name == 'my-netset-namespace'",
							Selector:          "projectcalico.org/name == 'my-netset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 5,
									MaxPort: 5,
								},
								{
									MinPort: 80,
									MaxPort: 80,
								},
							},
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-netset",
								"policyrecommendation.tigera.io/namespace":   "my-netset-namespace",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
				},
				Egress: []v3.Rule{
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 33,
									MaxPort: 33,
								},
								{
									MinPort: 90,
									MaxPort: 90,
								},
								{
									MinPort: 8080,
									MaxPort: 8080,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'my-netset-namespace'",
							Selector:          "projectcalico.org/name == 'my-netset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-netset",
								"policyrecommendation.tigera.io/namespace":   "my-netset-namespace",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
					{
						Action:   v3.Allow,
						Protocol: &protocolTCP,
						Source:   v3.EntityRule{},
						Destination: v3.EntityRule{
							Ports: []numorstring.Port{
								{
									MinPort: 80,
									MaxPort: 80,
								},
								{
									MinPort: 8081,
									MaxPort: 8081,
								},
							},
							NamespaceSelector: "projectcalico.org/name == 'global()'",
							Selector:          "projectcalico.org/name == 'my-globalnetset' && projectcalico.org/kind == 'NetworkSet'",
						},
						Metadata: &v3.RuleMetadata{
							Annotations: map[string]string{
								"policyrecommendation.tigera.io/lastUpdated": timeAtStep1,
								"policyrecommendation.tigera.io/name":        "my-globalnetset",
								"policyrecommendation.tigera.io/namespace":   "global()",
								"policyrecommendation.tigera.io/scope":       "NetworkSet",
							},
						},
					},
				}, Selector: "projectcalico.org/namespace == 'namespace1'",
				Types: []v3.PolicyType{"Egress", "Ingress"},
			},
		},
	}
)
