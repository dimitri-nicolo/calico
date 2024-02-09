// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package enginedata

import (
	"fmt"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"

	calres "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
)

var (
	IngressNamespaceRulesData = []v3.Rule{
		{
			Action: v3.Allow,
			Source: v3.EntityRule{
				NamespaceSelector: namespace2,
			},
			Destination: v3.EntityRule{
				Ports: portsOrdered1,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Thu, 30 Nov 2022 06:04:05 PST",
					calres.NamespaceKey:   namespace2,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Source: v3.EntityRule{
				NamespaceSelector: namespace2,
			},
			Destination: v3.EntityRule{
				Ports: portsOrdered2,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:04:05 PST",
					calres.NamespaceKey:   namespace2,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolUDP,
		},
		{
			Action: v3.Allow,
			Source: v3.EntityRule{
				NamespaceSelector: namespace3,
			},
			Destination: v3.EntityRule{
				Ports: portsOrdered3,
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
		{
			Action: v3.Allow,
			Source: v3.EntityRule{
				NamespaceSelector: namespace4,
			},
			Destination: v3.EntityRule{
				Ports: portsOrdered3,
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
					calres.NamespaceKey:   namespace4,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolUDP,
		},
	}

	IngressNetworkSetRulesData = []v3.Rule{
		{
			Action: v3.Allow,
			Source: v3.EntityRule{
				NamespaceSelector: "global()",
				Selector: fmt.Sprintf("projectcalico.org/name == '%s' && projectcalico.org/kind == '%s'",
					globalNetworkset2, string(calres.NetworkSetScope)),
			},
			Destination: v3.EntityRule{
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
			Protocol: &protocolUDP,
		},
		{
			Action: v3.Allow,
			Source: v3.EntityRule{
				NamespaceSelector: namespace3,
				Selector: fmt.Sprintf("projectcalico.org/name == '%s' && projectcalico.org/kind == '%s'",
					networkset3, string(calres.NetworkSetScope)),
			},
			Destination: v3.EntityRule{
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
			Source: v3.EntityRule{
				NamespaceSelector: namespace2,
				Selector: fmt.Sprintf("projectcalico.org/name == '%s' && projectcalico.org/kind == '%s'",
					networkset2, string(calres.NetworkSetScope)),
			},
			Destination: v3.EntityRule{
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

	IngressPrivateNetworkRulesData = []v3.Rule{
		{
			Action: v3.Allow,
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
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
					calres.ScopeKey:       string(calres.PrivateNetworkScope),
				},
			},
			Protocol: &protocolICMP,
		},
		{
			Action: v3.Allow,
			Source: v3.EntityRule{
				Nets: []string{
					"10.0.0.0/8",
					"172.16.0.0/12",
					"192.168.0.0/16",
				},
			},
			Destination: v3.EntityRule{
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
					}},
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

	IngressPublicNetworkRulesData = []v3.Rule{
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
