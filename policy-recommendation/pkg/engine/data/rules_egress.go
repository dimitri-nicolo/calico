// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package enginedata

import (
	"fmt"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	calres "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
)

var (
	EgressToDomainRulesData = []v3.Rule{
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Domains: []string{"calico.org"},
				Ports:   portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.ScopeKey:       string(calres.EgressToDomainScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Domains: []string{"tigera.io"},
				Ports:   portsOrdered2,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Thu, 30 Nov 2022 12:30:05 PST",
					calres.ScopeKey:       string(calres.EgressToDomainScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Domains: []string{"kubernetes.io"},
				Ports:   portsOrdered1,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 13:04:05 PST",
					calres.ScopeKey:       string(calres.EgressToDomainScope),
				},
			},
			Protocol: &protocolUDP,
		},
	}

	EgressToServiceRulesData = []v3.Rule{
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Services: &v3.ServiceMatch{
					Name:      service3,
					Namespace: namespace3,
				},
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
					calres.NameKey:        service3,
					calres.NamespaceKey:   namespace3,
					calres.ScopeKey:       string(calres.EgressToServiceScope),
				},
			},
			Protocol: &protocolICMP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Services: &v3.ServiceMatch{
					Name:      service2,
					Namespace: namespace2,
				},
				Ports: portsOrdered1,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Thu, 30 Nov 2022 06:04:05 PST",
					calres.NameKey:        service2,
					calres.NamespaceKey:   namespace2,
					calres.ScopeKey:       string(calres.EgressToServiceScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Services: &v3.ServiceMatch{
					Name:      service2,
					Namespace: namespace2,
				},
				Ports: portsOrdered1,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:04:05 PST",
					calres.NameKey:        service2,
					calres.NamespaceKey:   namespace2,
					calres.ScopeKey:       string(calres.EgressToServiceScope),
				},
			},
			Protocol: &protocolUDP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Services: &v3.ServiceMatch{
					Name:      service3,
					Namespace: namespace3,
				},
				Ports: portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
					calres.NameKey:        service3,
					calres.NamespaceKey:   namespace3,
					calres.ScopeKey:       string(calres.EgressToServiceScope),
				},
			},
			Protocol: &protocolUDP,
		},
	}

	EgressNamespaceRulesData = []v3.Rule{
		{
			Action: v3.Pass,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace1,
				Ports:             portsOrdered1,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Thu, 30 Nov 2022 06:04:05 PST",
					calres.NamespaceKey:   namespace1,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Pass,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace1,
				Ports:             portsOrdered1,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:04:05 PST",
					calres.NamespaceKey:   namespace1,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolUDP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace2,
				Ports:             portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
					calres.NamespaceKey:   namespace2,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolUDP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace3,
				Ports:             portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
					calres.NamespaceKey:   namespace3,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolUDP,
		},
	}

	EgressNetworkSetRulesData = []v3.Rule{
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: "global()",
				Selector: fmt.Sprintf("projectcalico.org/name == '%s' && projectcalico.org/kind == '%s'",
					globalNetworkset2, string(calres.NetworkSetScope)),
				Ports: portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.NameKey:        globalNetworkset2,
					calres.NamespaceKey:   "",
					calres.ScopeKey:       string(calres.NetworkSetScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace3,
				Selector: fmt.Sprintf("projectcalico.org/name == '%s' && projectcalico.org/kind == '%s'",
					networkset3, string(calres.NetworkSetScope)),
				Ports: portsOrdered2,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.NameKey:        networkset3,
					calres.NamespaceKey:   namespace3,
					calres.ScopeKey:       string(calres.NetworkSetScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace2,
				Selector: fmt.Sprintf("projectcalico.org/name == '%s' && projectcalico.org/kind == '%s'",
					networkset2, string(calres.NetworkSetScope)),
				Ports: portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.NameKey:        networkset2,
					calres.NamespaceKey:   namespace2,
					calres.ScopeKey:       string(calres.NetworkSetScope),
				},
			},
			Protocol: &protocolUDP,
		},
	}

	EgressPrivateNetworkRulesData = []v3.Rule{
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Nets: []string{
					"10.0.0.0/8",
					"172.16.0.0/12",
					"192.168.0.0/16",
				},
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.ScopeKey:       string(calres.PrivateNetworkScope),
				},
			},
			Protocol: &protocolICMP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Nets: []string{
					"10.0.0.0/8",
					"172.16.0.0/12",
					"192.168.0.0/16",
				},
				Ports: portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.ScopeKey:       string(calres.PrivateNetworkScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Nets: []string{
					"10.0.0.0/8",
					"172.16.0.0/12",
					"192.168.0.0/16",
				},
				Ports: portsOrdered2,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.ScopeKey:       string(calres.PrivateNetworkScope),
				},
			},
			Protocol: &protocolUDP,
		},
	}

	EgressPublicNetworkRulesData = []v3.Rule{
		{
			Action:      v3.Allow,
			Destination: v3.EntityRule{},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.ScopeKey:       string(calres.PublicNetworkScope),
				},
			},
			Protocol: &protocolICMP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Ports: portsOrdered1,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.ScopeKey:       string(calres.PublicNetworkScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				Ports: portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.ScopeKey:       string(calres.PublicNetworkScope),
				},
			},
			Protocol: &protocolUDP,
		},
	}
)
