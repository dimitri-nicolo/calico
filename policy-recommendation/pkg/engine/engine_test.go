// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.
package engine

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"

	"github.com/projectcalico/calico/lma/pkg/api"
	calres "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	enginedata "github.com/projectcalico/calico/policy-recommendation/pkg/engine/data"
	"github.com/projectcalico/calico/policy-recommendation/pkg/types"
	testutils "github.com/projectcalico/calico/policy-recommendation/tests/utils"
	"github.com/projectcalico/calico/policy-recommendation/utils"
)

const (
	testDataFile = "../../tests/data/flows.json"

	timeNowRFC3339 = "2022-11-30T09:01:38Z"
)

var (
	protocolTCP  = numorstring.ProtocolFromString("TCP")
	protocolUDP  = numorstring.ProtocolFromString("UDP")
	protocolICMP = numorstring.ProtocolFromString("ICMP")
)

type mockRealClock struct{}

func (mockRealClock) NowRFC3339() string { return timeNowRFC3339 }

var mrc mockRealClock

var _ = DescribeTable("processFlow",
	func(eng *recommendationEngine, flow *api.Flow, expectedEgress, expectedIngress engineRules) {
		eng.processFlow(flow)

		Expect(eng.egress.size).To(Equal(expectedEgress.size))
		for key, val := range eng.egress.egressToDomainRules {
			Expect(expectedEgress.egressToDomainRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.egressToServiceRules {
			Expect(expectedEgress.egressToServiceRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.namespaceRules {
			Expect(expectedEgress.namespaceRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.networkSetRules {
			Expect(expectedEgress.networkSetRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.privateNetworkRules {
			Expect(expectedEgress.privateNetworkRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.publicNetworkRules {
			Expect(expectedEgress.publicNetworkRules).To(HaveKeyWithValue(key, val))
		}

		Expect(eng.ingress.size).To(Equal(expectedIngress.size))
		for key, val := range eng.ingress.egressToDomainRules {
			Expect(expectedIngress.egressToDomainRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.ingress.egressToServiceRules {
			Expect(expectedIngress.egressToServiceRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.ingress.namespaceRules {
			Expect(expectedIngress.namespaceRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.ingress.networkSetRules {
			Expect(expectedIngress.networkSetRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.ingress.privateNetworkRules {
			Expect(expectedIngress.privateNetworkRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.ingress.publicNetworkRules {
			Expect(expectedIngress.publicNetworkRules).To(HaveKeyWithValue(key, val))
		}
	},
	Entry("egress-to-public-domain",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:      "net",
				Domains:   "www.mydomain.com",
				Namespace: "",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			egressToDomainRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "",
					protocol:  protocolTCP,
					port:      numorstring.Port{MinPort: 8081, MaxPort: 8081},
				}: {
					Action:    v3.Allow,
					Domains:   []string{"www.mydomain.com"},
					Namespace: "",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			publicNetworkRules: map[engineRuleKey]*types.FlowLogData{},
			size:               1,
		},
		engineRules{},
	),
	Entry("egress-to-service",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:        "wep",
				Name:        "",
				Namespace:   "",
				ServiceName: "some-service",
				Port:        getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			egressToServiceRules: map[engineRuleKey]*types.FlowLogData{
				{
					name:      "some-service",
					namespace: "",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Name:      "some-service",
					Namespace: "",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("egress-to-local-service",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:        "wep",
				Name:        "my-service.namespace2",
				Namespace:   "namespace2",
				ServiceName: "my-service.namespace2.svc.cluster.local",
				Port:        getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace2",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Name:      "",
					Namespace: "namespace2",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("egress-to-namespace-allow",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace2",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace2",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Name:      "",
					Namespace: "namespace2",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("egress-to-namespace-pass",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace2",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace2",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Name:      "",
					Namespace: "namespace2",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("egress-to-intra-namespace-allow",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace1",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Name:      "",
					Namespace: "namespace1",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace1",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Name:      "",
					Namespace: "namespace1",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 80}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
		},
	),
	Entry("egress-to-intra-namespace-pass",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1a-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1b-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace1",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Pass,
					Name:      "",
					Namespace: "namespace1",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("egress-to-networkset",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:      "ns",
				Name:      "netset-1-*",
				Namespace: "namespace2",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			networkSetRules: map[engineRuleKey]*types.FlowLogData{
				{
					name:      "netset-1-*",
					namespace: "namespace2",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Name:      "netset-1-*",
					Namespace: "namespace2",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("egress-to-global-networkset",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type: "ns",
				Name: "global-netset-1-*",
				Port: getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			networkSetRules: map[engineRuleKey]*types.FlowLogData{
				{
					global:   true,
					name:     "global-netset-1-*",
					protocol: protocolTCP,
				}: {
					Action:    v3.Allow,
					Global:    true,
					Name:      "global-netset-1-*",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("egress-to-private-network",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type: "net",
				Name: "pvt",
				Port: getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			privateNetworkRules: map[engineRuleKey]*types.FlowLogData{
				{
					protocol: protocolTCP,
				}: {
					Action:    v3.Allow,
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("egress-to-public-network",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "src",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type: "net",
				Name: "pub",
				Port: getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{
			publicNetworkRules: map[engineRuleKey]*types.FlowLogData{
				{
					protocol: protocolTCP,
				}: {
					Action:    v3.Allow,
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("ingress-from-namespace-allow",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "dst",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-2-*",
				Namespace: "namespace2",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace2",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Namespace: "namespace2",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("ingress-from-namespace-pass",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "dst",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-2-*",
				Namespace: "namespace2",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace2",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Namespace: "namespace2",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("ingress-from-intra-namespace-allow",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "dst",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-2-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace1",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Namespace: "namespace1",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("ingress-from-intra-namespace-pass",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "dst",
			Source: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-2-*",
				Namespace: "namespace1",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace1",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Pass,
					Namespace: "namespace1",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("ingress-from-networkset",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "dst",
			Source: api.FlowEndpointData{
				Type:      "ns",
				Name:      "networkset-1-*",
				Namespace: "namespace2",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{},
		engineRules{
			networkSetRules: map[engineRuleKey]*types.FlowLogData{
				{
					name:      "networkset-1-*",
					namespace: "namespace2",
					protocol:  protocolTCP,
				}: {
					Action:    v3.Allow,
					Name:      "networkset-1-*",
					Namespace: "namespace2",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("ingress-from-global-networkset",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "dst",
			Source: api.FlowEndpointData{
				Type: "ns",
				Name: "global-networkset-1-*",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{},
		engineRules{
			networkSetRules: map[engineRuleKey]*types.FlowLogData{
				{
					global:   true,
					name:     "global-networkset-1-*",
					protocol: protocolTCP,
				}: {
					Action:    v3.Allow,
					Global:    true,
					Name:      "global-networkset-1-*",
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("ingress-from-private-network",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "dst",
			Source: api.FlowEndpointData{
				Type: "net",
				Name: "pvt",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{},
		engineRules{
			privateNetworkRules: map[engineRuleKey]*types.FlowLogData{
				{
					protocol: protocolTCP,
				}: {
					Action:    v3.Allow,
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("ingress-from-public-network",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		&api.Flow{
			Reporter: "dst",
			Source: api.FlowEndpointData{
				Type: "net",
				Name: "pub",
			},
			Destination: api.FlowEndpointData{
				Type:      "wep",
				Name:      "pod-1-*",
				Namespace: "namespace1",
				Port:      getPtrUint16(8081),
			},
			ActionFlag: 1,
			Proto:      getPtrUint8(6),
		},
		engineRules{},
		engineRules{
			publicNetworkRules: map[engineRuleKey]*types.FlowLogData{
				{
					protocol: protocolTCP,
				}: {
					Action:    v3.Allow,
					Protocol:  protocolTCP,
					Ports:     []numorstring.Port{{MinPort: 8081, MaxPort: 8081}},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
)

var _ = DescribeTable("buildRules",
	func(eng *recommendationEngine, dir calres.DirectionType, rules []v3.Rule, expectedEgress, expectedIngress engineRules) {
		eng.buildRules(dir, rules)

		Expect(eng.egress.size).To(Equal(expectedEgress.size))
		for key, val := range eng.egress.egressToDomainRules {
			Expect(expectedEgress.egressToDomainRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.egressToServiceRules {
			Expect(expectedEgress.egressToServiceRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.namespaceRules {
			Expect(expectedEgress.namespaceRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.networkSetRules {
			Expect(expectedEgress.networkSetRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.privateNetworkRules {
			Expect(expectedEgress.privateNetworkRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.egress.publicNetworkRules {
			Expect(expectedEgress.publicNetworkRules).To(HaveKeyWithValue(key, val))
		}

		Expect(eng.ingress.size).To(Equal(expectedIngress.size))
		Expect(eng.ingress.egressToDomainRules).To(HaveLen(0))
		Expect(eng.ingress.egressToServiceRules).To(HaveLen(0))
		for key, val := range eng.ingress.namespaceRules {
			Expect(expectedIngress.namespaceRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.ingress.networkSetRules {
			Expect(expectedIngress.networkSetRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.ingress.privateNetworkRules {
			Expect(expectedIngress.privateNetworkRules).To(HaveKeyWithValue(key, val))
		}
		for key, val := range eng.ingress.publicNetworkRules {
			Expect(expectedIngress.publicNetworkRules).To(HaveKeyWithValue(key, val))
		}
	},
	Entry("build-egress-to-domains",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.EgressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					Domains: []string{"www.my-domain1.com", "my-domain2.com"},
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.EgressToDomainScope),
					},
				},
			},
		},
		engineRules{
			egressToDomainRules: map[engineRuleKey]*types.FlowLogData{
				{
					name:      "",
					namespace: "",
					protocol:  numorstring.ProtocolFromInt(6),
					port:      numorstring.Port{MinPort: 80, MaxPort: 80},
				}: {
					Action:    v3.Allow,
					Domains:   []string{"www.my-domain1.com", "my-domain2.com"},
					Name:      "",
					Namespace: "",
					Protocol:  numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			publicNetworkRules: map[engineRuleKey]*types.FlowLogData{},
			size:               1,
		},
		engineRules{},
	),
	Entry("build-egress-to-service",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.EgressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					Services: &v3.ServiceMatch{
						Name: "external-service",
					},
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "external-service",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.EgressToServiceScope),
					},
				},
			},
		},
		engineRules{
			egressToServiceRules: map[engineRuleKey]*types.FlowLogData{
				{
					name:     "external-service",
					protocol: numorstring.ProtocolFromInt(6),
				}: {
					Action:   v3.Allow,
					Name:     "external-service",
					Protocol: numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("build-egress-to-namespace-allow",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.EgressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "pod-2-*",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "namespace2",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.NamespaceScope),
					},
				},
			},
		},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace2",
					protocol:  numorstring.ProtocolFromInt(6),
				}: {
					Action:    v3.Allow,
					Namespace: "namespace2",
					Protocol:  numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("build-egress-to-namespace-pass",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.EgressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Pass,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					NamespaceSelector: "projectcalico.org/name == 'namespace1'",
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "pod-2-*",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "namespace1",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.NamespaceScope),
					},
				},
			},
		},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace1",
					protocol:  numorstring.ProtocolFromInt(6),
				}: {
					Action:    v3.Pass,
					Namespace: "namespace1",
					Protocol:  numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("build-egress-to-networkset",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.EgressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "networkset-1",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "namespace2",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.NetworkSetScope),
					},
				},
			},
		},
		engineRules{
			networkSetRules: map[engineRuleKey]*types.FlowLogData{
				{
					name:      "networkset-1",
					namespace: "namespace2",
					protocol:  numorstring.ProtocolFromInt(6),
				}: {
					Action:    v3.Allow,
					Name:      "networkset-1",
					Namespace: "namespace2",
					Protocol:  numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("build-egress-to-global-networkset",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.EgressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					NamespaceSelector: "global()",
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "global-networkset-1",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.NetworkSetScope),
					},
				},
			},
		},
		engineRules{
			networkSetRules: map[engineRuleKey]*types.FlowLogData{
				{
					global:   true,
					name:     "global-networkset-1",
					protocol: numorstring.ProtocolFromInt(6),
				}: {
					Action:   v3.Allow,
					Global:   true,
					Name:     "global-networkset-1",
					Protocol: numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("build-egress-to-private-network",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.EgressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.PrivateNetworkScope),
					},
				},
			},
		},
		engineRules{
			privateNetworkRules: map[engineRuleKey]*types.FlowLogData{
				{
					protocol: numorstring.ProtocolFromInt(6),
				}: {
					Action:   v3.Allow,
					Protocol: numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("build-egress-to-public-network",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), true, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.EgressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.PublicNetworkScope),
					},
				},
			},
		},
		engineRules{
			publicNetworkRules: map[engineRuleKey]*types.FlowLogData{
				{
					protocol: numorstring.ProtocolFromInt(6),
				}: {
					Action:   v3.Allow,
					Protocol: numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
		engineRules{},
	),
	Entry("build-ingress-from-namespace",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.IngressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source: v3.EntityRule{
					NamespaceSelector: "projectcalico.org/name == 'namespace2'",
				},
				Destination: v3.EntityRule{
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "pod-2-*",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "namespace2",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.NamespaceScope),
					},
				},
			},
		},
		engineRules{},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace2",
					protocol:  numorstring.ProtocolFromInt(6),
				}: {
					Action:    v3.Allow,
					Namespace: "namespace2",
					Protocol:  numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("build-ingress-from-intra-namespace",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.IngressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source: v3.EntityRule{
					NamespaceSelector: "projectcalico.org/name == 'namespace1'",
				},
				Destination: v3.EntityRule{
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "pod-2-*",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "namespace1",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.NamespaceScope),
					},
				},
			},
		},
		engineRules{},
		engineRules{
			namespaceRules: map[engineRuleKey]*types.FlowLogData{
				{
					namespace: "namespace1",
					protocol:  numorstring.ProtocolFromInt(6),
				}: {
					Action:    v3.Allow,
					Namespace: "namespace1",
					Protocol:  numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("build-ingress-from-networkset",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.IngressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source: v3.EntityRule{
					NamespaceSelector: "projectcalico.org/name == 'namespace2'",
					Selector:          fmt.Sprintf("projectcalico.org/name == 'networkset-1' && projectcalico.org/kind == '%s'", string(calres.NetworkSetScope)),
				},
				Destination: v3.EntityRule{
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "networkset-1",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "namespace2",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.NetworkSetScope),
					},
				},
			},
		},
		engineRules{},
		engineRules{
			networkSetRules: map[engineRuleKey]*types.FlowLogData{
				{
					name:      "networkset-1",
					namespace: "namespace2",
					protocol:  numorstring.ProtocolFromInt(6),
				}: {
					Action:    v3.Allow,
					Name:      "networkset-1",
					Namespace: "namespace2",
					Protocol:  numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("build-ingress-from-global-networkset",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.IngressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source: v3.EntityRule{
					NamespaceSelector: "global()",
					Selector:          fmt.Sprintf("projectcalico.org/name == 'global-networkset-1' && projectcalico.org/kind == '%s'", string(calres.NetworkSetScope)),
				},
				Destination: v3.EntityRule{
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/name", calres.PolicyRecKeyName):        "global-networkset-1",
						fmt.Sprintf("%s/namespace", calres.PolicyRecKeyName):   "namespace1",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.NetworkSetScope),
					},
				},
			},
		},
		engineRules{},
		engineRules{
			networkSetRules: map[engineRuleKey]*types.FlowLogData{
				{
					global:   true,
					name:     "global-networkset-1",
					protocol: numorstring.ProtocolFromInt(6),
				}: {
					Action:   v3.Allow,
					Global:   true,
					Name:     "global-networkset-1",
					Protocol: numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("build-ingress-from-private-network",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.IngressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.PrivateNetworkScope),
					},
				},
			},
		},
		engineRules{},
		engineRules{
			privateNetworkRules: map[engineRuleKey]*types.FlowLogData{
				{
					protocol: numorstring.ProtocolFromInt(6),
				}: {
					Action:   v3.Allow,
					Protocol: numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
	Entry("build-ingress-from-public-network",
		newRecommendationEngine("", "namespace1", "", nil, mrc, time.Duration(0), time.Duration(0), false, "svc.cluster.local", *log.WithField("cluster", "my-cluster")),
		calres.IngressTraffic,
		[]v3.Rule{
			{
				Action:   v3.Allow,
				Protocol: protocolFromInt(uint8(6)),
				Source:   v3.EntityRule{},
				Destination: v3.EntityRule{
					Ports: []numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
					},
				},
				Metadata: &v3.RuleMetadata{
					Annotations: map[string]string{
						fmt.Sprintf("%s/lastUpdated", calres.PolicyRecKeyName): "2022-11-30T09:01:38Z",
						fmt.Sprintf("%s/scope", calres.PolicyRecKeyName):       string(calres.PublicNetworkScope),
					},
				},
			},
		},
		engineRules{},
		engineRules{
			publicNetworkRules: map[engineRuleKey]*types.FlowLogData{
				{
					protocol: numorstring.ProtocolFromInt(6),
				}: {
					Action:   v3.Allow,
					Protocol: numorstring.ProtocolFromInt(6),
					Ports: []numorstring.Port{
						{MinPort: 80, MaxPort: 80},
					},
					Timestamp: "2022-11-30T09:01:38Z",
				},
			},
			size: 1,
		},
	),
)

func protocolFromInt(i uint8) *numorstring.Protocol {
	p := numorstring.ProtocolFromInt(i)
	return &p
}

var _ = Describe("processFlow", func() {
	const serviceNameSuffix = "svc.cluster.local"

	var (
		eng *recommendationEngine

		flowData []api.Flow

		name      = "test_name"
		namespace = "namespace1"
		tier      = "test_tier"
		order     = float64(1)

		interval      = time.Duration(150 * time.Second)
		stabilization = time.Duration(10 * time.Minute)

		clock = mrc
	)

	BeforeEach(func() {
		eng = newRecommendationEngine(
			name,
			namespace,
			tier,
			&order,
			clock,
			interval,
			stabilization,
			false,
			serviceNameSuffix,
			*log.WithField("cluster", "my-cluster"),
		)

		err := testutils.LoadData(testDataFile, &flowData)
		Expect(err).To(BeNil())
	})

	It("Test valid engine rule generation", func() {
		for _, data := range flowData {
			eng.processFlow(&data)
		}

		Expect(len(eng.egress.namespaceRules)).To(Equal(2))
		Expect(eng.egress.namespaceRules[engineRuleKey{namespace: "namespace1", protocol: protocolTCP}]).
			To(Equal(&types.FlowLogData{Action: v3.Allow, Namespace: "namespace1", Protocol: protocolTCP, Ports: ports1, Timestamp: "2022-11-30T09:01:38Z"}))
		Expect(eng.egress.namespaceRules[engineRuleKey{namespace: "namespace2", protocol: protocolTCP}]).
			To(Equal(&types.FlowLogData{Action: v3.Allow, Namespace: "namespace2", Protocol: protocolTCP, Ports: ports2, Timestamp: "2022-11-30T09:01:38Z"}))

		Expect(len(eng.ingress.namespaceRules)).To(Equal(2))
		Expect(eng.ingress.namespaceRules[engineRuleKey{namespace: "namespace1", protocol: protocolTCP}]).
			To(Equal(&types.FlowLogData{Action: v3.Allow, Namespace: "namespace1", Protocol: protocolTCP, Ports: ports1, Timestamp: "2022-11-30T09:01:38Z"}))
		Expect(eng.ingress.namespaceRules[engineRuleKey{namespace: "namespace2", protocol: protocolTCP}]).
			To(Equal(&types.FlowLogData{Action: v3.Allow, Namespace: "namespace2", Protocol: protocolTCP, Ports: ports1, Timestamp: "2022-11-30T09:01:38Z"}))
	})

	It("Test flow with ActionFlagDeny", func() {
		flow := &api.Flow{
			ActionFlag: api.ActionFlagDeny,
		}

		eng.processFlow(flow)
		Expect(eng.egress.size).To(Equal(0))
		Expect(eng.ingress.size).To(Equal(0))
	})

	It("Test flow with ActionFlagEndOfTierDeny", func() {
		flow := &api.Flow{
			ActionFlag: api.ActionFlagEndOfTierDeny,
		}

		eng.processFlow(flow)
		Expect(eng.egress.size).To(Equal(0))
		Expect(eng.ingress.size).To(Equal(0))
	})

	It("Test 'src' reported flow that matches", func() {
		namespace := "namespace1"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeSource,
			Source: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeWEP,
				Namespace: namespace,
			},
		}

		eng.processFlow(flow)
		Expect(eng.egress.size).To(Equal(0))
		Expect(eng.ingress.size).To(Equal(0))
	})

	It("Test 'src' reported flow that is not WEP", func() {
		namespace := "namespace1"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeSource,
			Source: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeHEP,
				Namespace: namespace,
			},
		}

		eng.processFlow(flow)
		Expect(eng.egress.size).To(Equal(0))
		Expect(eng.ingress.size).To(Equal(0))
	})

	It("Test 'src' reported flow where the source flow is not equal to the rec engine namespace", func() {
		namespace := "not-the-engine-namespace"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeSource,
			Source: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeWEP,
				Namespace: namespace,
			},
		}

		eng.processFlow(flow)
		Expect(eng.egress.size).To(Equal(0))
		Expect(eng.ingress.size).To(Equal(0))
	})

	It("Test 'dst' reported flow that matches", func() {
		namespace := "namespace1"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeDestination,
			Destination: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeWEP,
				Namespace: namespace,
			},
		}

		eng.processFlow(flow)
		Expect(eng.egress.size).To(Equal(0))
		Expect(eng.ingress.size).To(Equal(0))
	})

	It("Test 'dst' reported flow that is not WEP", func() {
		namespace := "namespace1"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeDestination,
			Destination: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeHEP,
				Namespace: namespace,
			},
		}

		eng.processFlow(flow)
		Expect(eng.egress.size).To(Equal(0))
		Expect(eng.ingress.size).To(Equal(0))
	})

	It("Test 'dst' reported flow where the source flow is not equal to the rec engine namespace", func() {
		namespace := "not-the-engine-namespace"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeDestination,
			Destination: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeWEP,
				Namespace: namespace,
			},
		}

		eng.processFlow(flow)
		Expect(eng.egress.size).To(Equal(0))
		Expect(eng.ingress.size).To(Equal(0))

	})
})

var _ = Describe("ProcessRecommendation", func() {
	const (
		serviceNameSuffix = "svc.cluster.local"
		tier              = "test_tier"
	)

	var (
		flowsEgress, flowsIngress []*api.Flow

		interval      = time.Duration(150 * time.Second)
		stabilization = time.Duration(10 * time.Minute)

		clock = mrc
	)

	BeforeEach(func() {
		data := []api.Flow{}
		err := testutils.LoadData("./data/flows_egress.json", &data)
		Expect(err).To(BeNil())

		for i := range data {
			flowsEgress = append(flowsEgress, &data[i])
		}

		data = []api.Flow{}
		err = testutils.LoadData("./data/flows_ingress.json", &data)
		Expect(err).To(BeNil())

		for i := range data {
			flowsIngress = append(flowsIngress, &data[i])
		}
	})

	It("Test new rule injection", func() {
		owner := metav1.OwnerReference{
			APIVersion:         "projectcalico.org/v3",
			Kind:               "PolicyRecommendationScope",
			Name:               "default",
			UID:                "orikr-9df4d-0k43m",
			Controller:         getPtrBool(true),
			BlockOwnerDeletion: getPtrBool(false),
		}
		snp := calres.NewStagedNetworkPolicy(
			utils.GetPolicyName(tier, "name1", func() string { return "xv5fb" }),
			"namespace1",
			tier,
			owner,
		)

		snp.Spec.Egress = append(snp.Spec.Egress, enginedata.EgressToDomainRulesData...)
		snp.Spec.Egress = append(snp.Spec.Egress, enginedata.EgressToServiceRulesData...)
		snp.Spec.Egress = append(snp.Spec.Egress, enginedata.EgressNamespaceRulesData...)
		snp.Spec.Egress = append(snp.Spec.Egress, enginedata.EgressNetworkSetRulesData...)
		snp.Spec.Egress = append(snp.Spec.Egress, enginedata.EgressPrivateNetworkRulesData...)
		snp.Spec.Egress = append(snp.Spec.Egress, enginedata.EgressPublicNetworkRulesData...)

		snp.Spec.Ingress = append(snp.Spec.Ingress, enginedata.IngressNamespaceRulesData...)
		snp.Spec.Ingress = append(snp.Spec.Ingress, enginedata.IngressNetworkSetRulesData...)
		snp.Spec.Ingress = append(snp.Spec.Ingress, enginedata.IngressPrivateNetworkRulesData...)
		snp.Spec.Ingress = append(snp.Spec.Ingress, enginedata.IngressPublicNetworkRulesData...)

		eng := getRecommendationEngine(
			*snp,
			clock,
			interval,
			stabilization,
			true,
			serviceNameSuffix,
			*log.WithField("cluster", "my-cluster"),
		)

		eng.processRecommendation(flowsEgress, snp)
		log.Infof("actual egress: %s,\nexpected egress: %s", prettyRules(snp.Spec.Egress), prettyRules(enginedata.ExpectedSnpNamespace1.Spec.Egress))

		eng.processRecommendation(flowsIngress, snp)
		log.Infof("actual ingress: %s,\nexpected ingress: %s", prettyRules(snp.Spec.Ingress), prettyRules(enginedata.ExpectedSnpNamespace1.Spec.Ingress))

		Expect(equality.Semantic.Equalities.DeepDerivative(*snp, *enginedata.ExpectedSnpNamespace1)).To(BeTrue())
	})
})

var _ = DescribeTable("lessPorts",
	func(a, b []numorstring.Port, expected int) {
		Expect(lessPorts(a, b)).To(Equal(expected))
	},
	Entry("less-ports-1",
		[]numorstring.Port{{MinPort: 0, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		-1,
	),
	Entry("less-ports-2",
		[]numorstring.Port{{MinPort: 1, MaxPort: 1, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		-1,
	),
	Entry("less-ports-3",
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 3, PortName: "A"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		-1,
	),
	Entry("less-ports-4",
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		0,
	),
	Entry("less-ports-5",
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 5, PortName: "C"}},
		1,
	),
	Entry("less-ports-6",
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "A"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		-1,
	),
	Entry("less-ports-7",
		[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		[]numorstring.Port{{MinPort: 1, MaxPort: 1, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}},
		1,
	),
)

var _ = DescribeTable("lessStringArrays",
	func(a, b []string, expected bool) {
		Expect(lessStringArrays(a, b)).To(Equal(expected))
	},
	Entry("less-string-arrays-1",
		[]string{"apple"},
		[]string{"Apple"},
		false,
	),
	Entry("less-string-arrays-2",
		[]string{"apple", "banana"},
		[]string{"apple", "banana", "cherry"},
		true,
	),
	Entry("less-string-arrays-3",
		[]string{"apple", "banana", "cherry"},
		[]string{"apple", "banana", "cherry"},
		false,
	),
	Entry("less-string-arrays-4",
		[]string{"apple", "banana", "cherry"},
		[]string{"apple", "banana", "apple"},
		false,
	),
	Entry("less-string-arrays-5",
		[]string{"apple", "banana", "cherry"},
		[]string{"banana", "cherry", "date"},
		true,
	),
	Entry("less-string-arrays-6",
		[]string{"grape", "kiwi", "mango"},
		[]string{"grape", "kiwi", "cherry"},
		false,
	),
)

var (
	ports1 = []numorstring.Port{
		{
			MinPort: 443,
			MaxPort: 443,
		},
	}

	ports2 = []numorstring.Port{
		{
			MinPort: 8080,
			MaxPort: 8080,
		},
		{
			MinPort: 5432,
			MaxPort: 5432,
		},
	}
)

// prettyRules logs a pretty version of map[string]string.
func prettyRules(rules []v3.Rule) string {
	value, err := json.MarshalIndent(rules, "", " ")
	Expect(err).NotTo(HaveOccurred())

	return string(value)
}

func getPtrBool(f bool) *bool {
	return &f
}

func getPtrUint8(i uint8) *uint8 {
	return &i
}

func getPtrUint16(i uint16) *uint16 {
	return &i
}
