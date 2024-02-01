// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package enginedata

import "github.com/tigera/api/pkg/lib/numorstring"

const (
	namespace1 = "namespace1"
	namespace2 = "namespace2"
	namespace3 = "namespace3"
	namespace4 = "namespace4"

	networkset2       = "netset-2"
	networkset3       = "netset-3"
	globalNetworkset2 = "global-netset-2"

	service1 = "service1"
	service2 = "service2"
	service3 = "service3"
)

var (
	portsOrdered1 = []numorstring.Port{
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
	}
	portsOrdered2 = []numorstring.Port{
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
	}
	portsOrdered3 = []numorstring.Port{
		{
			MinPort: 8080,
			MaxPort: 8081,
		},
	}

	protocolTCP  = numorstring.ProtocolFromString("TCP")
	protocolUDP  = numorstring.ProtocolFromString("UDP")
	protocolICMP = numorstring.ProtocolFromString("ICMP")
)
