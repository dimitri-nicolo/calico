// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package enginedata

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The contents of this file are meant to be used to hold the expected output of the staged network
// policy generation engine. The expected output is used to validate the correctness of the engine's
// output.
var (
	ExpectedSnpNamespace1 = &v3.StagedNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test_tier.name1-xv5fb",
			Namespace: "namespace1",
			Labels: map[string]string{
				"policyrecommendation.tigera.io/scope":  "namespace",
				"projectcalico.org/spec.stagedAction":   "Learn",
				"projectcalico.org/tier":                "test_tier",
				"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
			},
			Annotations: map[string]string{
				"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
				"policyrecommendation.tigera.io/status":      "NoData",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "projectcalico.org/v3",
					Kind:               "PolicyRecommendationScope",
					Name:               "default",
					UID:                "orikr-9df4d-0k43m",
					Controller:         getPtrBool(true),
					BlockOwnerDeletion: getPtrBool(false),
				},
			},
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionLearn,
			Tier:         "test_tier",
			Selector:     "projectcalico.org/namespace == 'namespace1'",
			Types: []v3.PolicyType{
				"Egress", "Ingress",
			},
			Egress: []v3.Rule{
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 1,
								MaxPort: 99,
							},
						},
						Domains: []string{
							"tigera.io",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Thu, 30 Nov 2022 12:30:05 PST",
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
								MinPort: 441,
								MaxPort: 441,
							},
						},
						Domains: []string{
							"www.tigera.io",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
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
								MinPort: 442,
								MaxPort: 442,
							},
						},
						Domains: []string{
							"www.tigera.io",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
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
								MinPort: 443,
								MaxPort: 443,
							},
						},
						Domains: []string{
							"www.google.com",
							"www.projectcalico.org",
							"www.tigera.io",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
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
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
						Domains: []string{
							"calico.org",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:30:05 PST",
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
								MinPort: 5,
								MaxPort: 59,
							},
						},
						Domains: []string{
							"kubernetes.io",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 13:04:05 PST",
							"policyrecommendation.tigera.io/scope":       "Domains",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolICMP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Services: &v3.ServiceMatch{
							Name: "service3",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:05:05 PST",
							"policyrecommendation.tigera.io/name":        "service3",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "Service",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolICMP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Services: &v3.ServiceMatch{
							Name: "service4",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "service4",
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
						Services: &v3.ServiceMatch{
							Name: "service2",
						},
						Ports: []numorstring.Port{
							{
								MinPort: 5,
								MaxPort: 59,
							},
							{
								MinPort: 22,
								MaxPort: 22,
							},
							{
								MinPort: 33,
								MaxPort: 33,
							},
							{
								MinPort: 44,
								MaxPort: 56,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "service2",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "Service",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Services: &v3.ServiceMatch{
							Name: "service2",
						},
						Ports: []numorstring.Port{
							{
								MinPort: 5,
								MaxPort: 59,
							},
							{
								MinPort: 22,
								MaxPort: 22,
							},
							{
								MinPort: 44,
								MaxPort: 56,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:04:05 PST",
							"policyrecommendation.tigera.io/name":        "service2",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "Service",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Services: &v3.ServiceMatch{
							Name: "service3",
						},
						Ports: []numorstring.Port{
							{
								MinPort: 3030,
								MaxPort: 3030,
							},
							{
								MinPort: 3031,
								MaxPort: 3031,
							},
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "service3",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "Service",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Services: &v3.ServiceMatch{
							Name: "service1",
						},
						Ports: []numorstring.Port{
							{
								MinPort: 3434,
								MaxPort: 3434,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "service1",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "Service",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolICMP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Pass,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace1'",
						Ports: []numorstring.Port{
							{
								MinPort: 5,
								MaxPort: 59,
							},
							{
								MinPort: 22,
								MaxPort: 22,
							},
							{
								MinPort: 44,
								MaxPort: 56,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Thu, 30 Nov 2022 06:04:05 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace3'",
						Ports: []numorstring.Port{
							{
								MinPort: 64,
								MaxPort: 64,
							},
							{
								MinPort: 645,
								MaxPort: 645,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/namespace":   "namespace3",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Pass,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace1'",
						Ports: []numorstring.Port{
							{
								MinPort: 5,
								MaxPort: 59,
							},
							{
								MinPort: 22,
								MaxPort: 22,
							},
							{
								MinPort: 44,
								MaxPort: 56,
							},
							{
								MinPort: 6464,
								MaxPort: 6464,
							},
							{
								MinPort: 6465,
								MaxPort: 6465,
							},
							{
								MinPort: 9099,
								MaxPort: 9099,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/namespace":   "namespace1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:05:05 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace3'",
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:05:05 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace3",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'netset-3' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "projectcalico.org/name == 'namespace3'",
						Ports: []numorstring.Port{
							{
								MinPort: 1,
								MaxPort: 99,
							},
							{
								MinPort: 3,
								MaxPort: 3,
							},
							{
								MinPort: 24,
								MaxPort: 35,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:30:05 PST",
							"policyrecommendation.tigera.io/name":        "netset-3",
							"policyrecommendation.tigera.io/namespace":   "namespace3",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},

				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'global-netset-1' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "global()",
						Ports: []numorstring.Port{
							{
								MinPort: 663,
								MaxPort: 663,
							},
							{
								MinPort: 667,
								MaxPort: 667,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "global-netset-1",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Pass,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'netset-1' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "projectcalico.org/name == 'namespace1'",
						Ports: []numorstring.Port{
							{
								MinPort: 666,
								MaxPort: 666,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "netset-1",
							"policyrecommendation.tigera.io/namespace":   "namespace1",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'netset-2' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
						Ports: []numorstring.Port{
							{
								MinPort: 667,
								MaxPort: 667,
							},
							{
								MinPort: 999,
								MaxPort: 999,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "netset-2",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'global-netset-2' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "global()",
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:30:05 PST",
							"policyrecommendation.tigera.io/name":        "global-netset-2",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'global-netset-2' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "global()",
						Ports: []numorstring.Port{
							{
								MinPort: 667,
								MaxPort: 667,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "global-netset-2",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'netset-2' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:30:05 PST",
							"policyrecommendation.tigera.io/name":        "netset-2",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolICMP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Nets: []string{
							"10.0.0.0/8",
							"172.16.0.0/12",
							"192.168.0.0/16",
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Private",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Nets: []string{
							"10.0.0.0/8",
							"172.16.0.0/12",
							"192.168.0.0/16",
						},
						Ports: []numorstring.Port{
							{
								MinPort: 441,
								MaxPort: 441,
							},
							{
								MinPort: 8080,
								MaxPort: 8080,
							},
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
							{
								MinPort: 8081,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Private",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Nets: []string{
							"10.0.0.0/8",
							"172.16.0.0/12",
							"192.168.0.0/16",
						},
						Ports: []numorstring.Port{
							{
								MinPort: 1,
								MaxPort: 99,
							},
							{
								MinPort: 3,
								MaxPort: 3,
							},
							{
								MinPort: 24,
								MaxPort: 35,
							},
							{
								MinPort: 999,
								MaxPort: 999,
							},
							{
								MinPort: 4568,
								MaxPort: 4568,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Private",
						},
					},
				},
				{
					Action:      v3.Allow,
					Protocol:    &protocolICMP,
					Source:      v3.EntityRule{},
					Destination: v3.EntityRule{},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Public",
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
								MinPort: 5,
								MaxPort: 59,
							},
							{
								MinPort: 22,
								MaxPort: 22,
							},
							{
								MinPort: 44,
								MaxPort: 56,
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
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Public",
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
								MinPort: 999,
								MaxPort: 999,
							},
							{
								MinPort: 4568,
								MaxPort: 4568,
							},
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Public",
						},
					},
				},
			},
			Ingress: []v3.Rule{
				{
					Action:   v3.Allow,
					Protocol: &protocolICMP,
					Source: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					},
					Destination: v3.EntityRule{},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 5,
								MaxPort: 59,
							},
							{
								MinPort: 22,
								MaxPort: 22,
							},
							{
								MinPort: 44,
								MaxPort: 56,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Thu, 30 Nov 2022 06:04:05 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace3'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 64,
								MaxPort: 64,
							},
							{
								MinPort: 645,
								MaxPort: 645,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/namespace":   "namespace3",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 1,
								MaxPort: 99,
							},
							{
								MinPort: 3,
								MaxPort: 3,
							},
							{
								MinPort: 24,
								MaxPort: 35,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:04:05 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace3'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:05:05 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace3",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source: v3.EntityRule{
						NamespaceSelector: "projectcalico.org/name == 'namespace4'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:05:05 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace4",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"policyrecommendation.tigera.io/warnings":    "NonServicePortsAndProtocol",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'netset-3' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "projectcalico.org/name == 'namespace3'",
					},
					Destination: v3.EntityRule{

						Ports: []numorstring.Port{
							{
								MinPort: 1,
								MaxPort: 99,
							},
							{
								MinPort: 3,
								MaxPort: 3,
							},
							{
								MinPort: 24,
								MaxPort: 35,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:30:05 PST",
							"policyrecommendation.tigera.io/name":        "netset-3",
							"policyrecommendation.tigera.io/namespace":   "namespace3",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'global-netset-1' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "global()",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 663,
								MaxPort: 663,
							},
							{
								MinPort: 667,
								MaxPort: 667,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "global-netset-1",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'netset-2' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 667,
								MaxPort: 667,
							},
							{
								MinPort: 999,
								MaxPort: 999,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "netset-2",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'global-netset-2' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "global()",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 667,
								MaxPort: 667,
							},
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/name":        "global-netset-2",
							"policyrecommendation.tigera.io/namespace":   "",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/name == 'netset-2' && projectcalico.org/kind == 'NetworkSet'",
						NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:30:05 PST",
							"policyrecommendation.tigera.io/name":        "netset-2",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "NetworkSet",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolICMP,
					Source: v3.EntityRule{
						Nets: []string{
							"10.0.0.0/8",
							"172.16.0.0/12",
							"192.168.0.0/16",
						},
					},
					Destination: v3.EntityRule{},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Private",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						Nets: []string{
							"10.0.0.0/8",
							"172.16.0.0/12",
							"192.168.0.0/16",
						},
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8080,
							},
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
							{
								MinPort: 8081,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Private",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source: v3.EntityRule{
						Nets: []string{
							"10.0.0.0/8",
							"172.16.0.0/12",
							"192.168.0.0/16",
						},
					},
					Destination: v3.EntityRule{
						Ports: []numorstring.Port{
							{
								MinPort: 88,
								MaxPort: 89,
							},
							{
								MinPort: 999,
								MaxPort: 999,
							},
							{
								MinPort: 4568,
								MaxPort: 4568,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Private",
						},
					},
				},
				{
					Action:      v3.Allow,
					Protocol:    &protocolICMP,
					Source:      v3.EntityRule{},
					Destination: v3.EntityRule{},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Public",
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
								MinPort: 5,
								MaxPort: 59,
							},
							{
								MinPort: 22,
								MaxPort: 22,
							},
							{
								MinPort: 44,
								MaxPort: 56,
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
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Public",
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
								MinPort: 999,
								MaxPort: 999,
							},
							{
								MinPort: 4568,
								MaxPort: 4568,
							},
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/scope":       "Public",
						},
					},
				},
			},
		},
	}
)

func getPtrBool(f bool) *bool {
	return &f
}
