// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "github.com/tigera/es-proxy/pkg/apis/v1"

	. "github.com/tigera/es-proxy/pkg/middleware/servicegraph"
)

var _ = Describe("Elasticsearch script interface tests", func() {
	var dummySg = &ServiceGroup{
		ID: GetServiceGroupID([]v1.NamespacedName{{
			Namespace: "my-service-namespace",
			Name:      "my-service-name",
		}}),
	}
	var dummyServiceGroups = mockServiceGroups{sg: dummySg}

	DescribeTable("Test ID normalization",
		func(id string, splitIngressEgress bool, expNormalizedFirst, expNormalizedSecond string) {
			nid, err := GetNormalizedIDs(v1.GraphNodeID(id), dummyServiceGroups, splitIngressEgress)
			Expect(err).NotTo(HaveOccurred())

			if expNormalizedFirst == "" {
				Expect(nid).To(HaveLen(0))
			} else if expNormalizedSecond == "" {
				Expect(nid).To(HaveLen(1))
			}

			if expNormalizedFirst != "" {
				Expect(nid[0]).To(BeEquivalentTo(expNormalizedFirst))
			}
			if expNormalizedSecond != "" {
				Expect(nid[1]).To(BeEquivalentTo(expNormalizedSecond))
			}
		},
		Entry("Layer; not split",
			"layer/my-layer", false,
			"layer/my-layer", "",
		),
		Entry("Layer; split",
			"layer/my-layer", true,
			"layer/my-layer", "",
		),
		Entry("Hosts; not split",
			"hosts/*", false,
			"hosts/*", "",
		),
		Entry("Hosts; split",
			"hosts/*", true,
			"hosts/*", "",
		),
		Entry("NetworkSet; split",
			"ns/netset", true,
			"ns/netset;dir/ingress", "ns/netset;dir/egress",
		),
		Entry("NetworkSet with direction; not split",
			"ns/netset;dir/egress", false,
			"ns/netset", "",
		),
		Entry("NetworkSet with direction; split",
			"ns/netset;dir/egress", true,
			"ns/netset;dir/egress", "",
		),
		Entry("Host; not split",
			"host/host1/*", false,
			"host/host1/*", "",
		),
		Entry("Host; split",
			"host/host1/*", true,
			"host/host1/*", "",
		),
	)

	DescribeTable("Test possible data sets returned by elasticsearch",
		func(idi IDInfo,
			layer, namespace,
			serviceGp, service, servicePort,
			aggrEndpoint, endpoint, aggrEndpointPort, endpointPort string,
		) {
			Expect(idi.GetLayerID()).To(BeEquivalentTo(layer))
			Expect(idi.GetNamespaceID()).To(BeEquivalentTo(namespace))
			Expect(idi.GetServiceGroupID()).To(BeEquivalentTo(serviceGp))
			Expect(idi.GetServiceID()).To(BeEquivalentTo(service))
			Expect(idi.GetServicePortID()).To(BeEquivalentTo(servicePort))
			Expect(idi.GetAggrEndpointID()).To(BeEquivalentTo(aggrEndpoint))
			Expect(idi.GetEndpointID()).To(BeEquivalentTo(endpoint))
			Expect(idi.GetAggrEndpointPortID()).To(BeEquivalentTo(aggrEndpointPort))
			Expect(idi.GetEndpointPortID()).To(BeEquivalentTo(endpointPort))
		},
		Entry("Layer",
			IDInfo{
				Layer: "my-layer",
			},
			"layer/my-layer", "",
			"", "", "",
			"", "", "", "",
		),
		Entry("Namespace",
			IDInfo{
				Endpoint: FlowEndpoint{
					Namespace: "namespace1",
				},
			},
			"", "namespace/namespace1",
			"", "", "",
			"", "", "", "",
		),
		Entry("ServiceGroup",
			IDInfo{
				ServiceGroup: &ServiceGroup{
					Namespace: "*",
					ID: GetServiceGroupID([]v1.NamespacedName{
						{Namespace: "service-namespace", Name: "service-name"},
						{Namespace: "service-namespace2", Name: "service-name2"},
					}),
				},
			},
			"", "namespace/*",
			"svcgp;svc/service-namespace/service-name;svc/service-namespace2/service-name2", "", "",
			"", "", "", "",
		),
		Entry("Workload endpoint",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeWorkload,
					Namespace: "namespace1",
					Name:      "wepname",
					NameAggr:  "wepname*",
				},
			},
			"", "namespace/namespace1",
			"", "", "",
			"rep/namespace1/wepname*", "wep/namespace1/wepname/wepname*", "", "",
		),
		Entry("Replica set",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeReplicaSet,
					Namespace: "ns",
					NameAggr:  "repname",
				},
			},
			"", "namespace/ns",
			"", "", "",
			"rep/ns/repname", "", "", "",
		),
		Entry("Host endpoint",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeHost,
					Name:     "hepname",
					NameAggr: "*",
				},
			},
			"", "",
			"", "", "",
			"hosts/*", "host/hepname/*", "", "",
		),
		Entry("Global network set",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetworkSet,
					NameAggr: "global-ns",
				},
			},
			"", "",
			"", "", "",
			"ns/global-ns", "", "", "",
		),
		Entry("Namespaced network set",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeNetworkSet,
					Namespace: "n1",
					NameAggr:  "n1-ns",
				},
			},
			"", "namespace/n1",
			"", "", "",
			"ns/n1/n1-ns", "", "", "",
		),
		Entry("Namespaced network set with service group",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeNetworkSet,
					Namespace: "n1",
					NameAggr:  "n1-ns",
				},
				ServiceGroup: &ServiceGroup{
					Namespace: "sns",
					ID: GetServiceGroupID([]v1.NamespacedName{{
						Namespace: "sns", Name: "sn",
					}}),
				},
			},
			"", "namespace/sns",
			"svcgp;svc/sns/sn", "", "",
			"ns/n1/n1-ns;svcgp;svc/sns/sn", "", "", "",
		),
		Entry("Network",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetwork,
					NameAggr: "pub",
				},
			},
			"", "",
			"", "", "",
			"net/pub", "", "", "",
		),
		Entry("Network with a direction",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetwork,
					NameAggr: "pub",
				},
				Direction: DirectionEgress,
			},
			"", "",
			"", "", "",
			"net/pub;dir/egress", "", "", "",
		),
		Entry("Workload endpoint with service and endpoint ports",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeWorkload,
					Namespace: "namespace1",
					Name:      "wepname",
					NameAggr:  "wepname*",
					PortNum:   20000,
					Protocol:  "udp",
				},
				Service: v1.ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "service-namespace",
						Name:      "service-name",
					},
					PortName: "http",
					Port:     8080,
					Protocol: "udp",
				},
				ServiceGroup: &ServiceGroup{
					Namespace: "service-namespace",
					ID: GetServiceGroupID([]v1.NamespacedName{
						{Namespace: "service-namespace", Name: "service-name"},
						{Namespace: "service-namespace", Name: "service-name2"},
					}),
				},
			},
			"", "namespace/service-namespace",
			"svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2",
			"svc/service-namespace/service-name", "svcport/udp/http/8080;svc/service-namespace/service-name",
			"rep/namespace1/wepname*", "wep/namespace1/wepname/wepname*",
			"port/udp/20000;rep/namespace1/wepname*", "port/udp/20000;wep/namespace1/wepname/wepname*",
		),
		Entry("Replica set with service",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeReplicaSet,
					Namespace: "ns",
					NameAggr:  "repname",
					Protocol:  "tcp",
				},
				Service: v1.ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "service-namespace",
						Name:      "service-name",
					},
					Protocol: "tcp",
					Port:     1111,
				},
				ServiceGroup: &ServiceGroup{
					Namespace: "service-namespace",
					ID: GetServiceGroupID([]v1.NamespacedName{
						{Namespace: "service-namespace", Name: "service-name"},
						{Namespace: "service-namespace", Name: "service-name2"},
					}),
				},
			},
			"", "namespace/service-namespace",
			"svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2",
			"svc/service-namespace/service-name", "svcport/tcp//1111;svc/service-namespace/service-name",
			"rep/ns/repname", "", "", "",
		),
		Entry("Host endpoint with service",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeHost,
					Name:     "hepname",
					NameAggr: "*",
					Protocol: "sctp",
				},
				Service: v1.ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "service-namespace",
						Name:      "service-name",
					},
					Protocol: "sctp",
					Port:     1234,
				},
				ServiceGroup: &ServiceGroup{
					Namespace: "service-namespace",
					ID: GetServiceGroupID([]v1.NamespacedName{
						{Namespace: "service-namespace", Name: "service-name"},
						{Namespace: "service-namespace", Name: "service-name2"},
					}),
				},
			},
			"", "namespace/service-namespace",
			"svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2",
			"svc/service-namespace/service-name", "svcport/sctp//1234;svc/service-namespace/service-name",
			"hosts/*;svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2",
			"host/hepname/*;svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2", "", "",
		),
		Entry("Global network set with service",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetworkSet,
					NameAggr: "global-ns",
					Protocol: "udp",
				},
				Service: v1.ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "service-namespace",
						Name:      "service-name",
					},
					Protocol: "udp",
					Port:     1212,
				},
				ServiceGroup: &ServiceGroup{
					Namespace: "service-namespace",
					ID: GetServiceGroupID([]v1.NamespacedName{
						{Namespace: "service-namespace", Name: "service-name"},
						{Namespace: "service-namespace", Name: "service-name2"},
					}),
				},
			},
			"", "namespace/service-namespace",
			"svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2",
			"svc/service-namespace/service-name", "svcport/udp//1212;svc/service-namespace/service-name",
			"ns/global-ns;svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2", "", "", "",
		),
		Entry("Namespaced network set with service",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeNetworkSet,
					Namespace: "n1",
					NameAggr:  "n1-ns",
					Protocol:  "udp",
				},
				Service: v1.ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "service-namespace",
						Name:      "service-name",
					},
					Protocol: "udp",
					Port:     1313,
				},
				ServiceGroup: &ServiceGroup{
					Namespace: "service-namespace",
					ID: GetServiceGroupID([]v1.NamespacedName{
						{Namespace: "service-namespace", Name: "service-name"},
						{Namespace: "service-namespace", Name: "service-name2"},
					}),
				},
			},
			"", "namespace/service-namespace",
			"svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2",
			"svc/service-namespace/service-name", "svcport/udp//1313;svc/service-namespace/service-name",
			"ns/n1/n1-ns;svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2", "", "", "",
		),
		Entry("Network with service",
			IDInfo{
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetwork,
					NameAggr: "pub",
					Protocol: "udp",
				},
				Service: v1.ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "service-namespace",
						Name:      "service-name",
					},
					PortName: "http",
					Protocol: "udp",
					Port:     88,
				},
				ServiceGroup: &ServiceGroup{
					Namespace: "service-namespace",
					ID: GetServiceGroupID([]v1.NamespacedName{
						{Namespace: "service-namespace", Name: "service-name"},
						{Namespace: "service-namespace", Name: "service-name2"},
					}),
				},
			},
			"", "namespace/service-namespace",
			"svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2",
			"svc/service-namespace/service-name", "svcport/udp/http/88;svc/service-namespace/service-name",
			"net/pub;svcgp;svc/service-namespace/service-name;svc/service-namespace/service-name2", "", "", "",
		),
	)

	DescribeTable("Test node id parsing",
		func(id string, node IDInfo) {
			n, e := ParseGraphNodeID(v1.GraphNodeID(id), dummyServiceGroups)
			Expect(e).NotTo(HaveOccurred())
			Expect(*n).To(Equal(node))
		},
		Entry("layer",
			"layer/my-layer", IDInfo{
				ParsedIDType: v1.GraphNodeTypeLayer,
				Layer:        "my-layer",
			},
		),
		Entry("Namespace",
			"namespace/my-namespace", IDInfo{
				ParsedIDType: v1.GraphNodeTypeNamespace,
				Endpoint: FlowEndpoint{
					Namespace: "my-namespace",
				},
			},
		),
		Entry("service group",
			"svcgp;svc/my-service-namespace/my-service-name", IDInfo{
				ParsedIDType: v1.GraphNodeTypeServiceGroup,
				ServiceGroup: dummySg,
			},
		),
		Entry("service port with no port name",
			"svcport/udp//1233;svc/svc-namespace/svc-name", IDInfo{
				ParsedIDType: v1.GraphNodeTypeServicePort,
				Service: v1.ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "svc-namespace",
						Name:      "svc-name",
					},
					Protocol: "udp",
					Port:     1233,
				},
			},
		),
		Entry("service port with port name",
			"svcport/sctp/po.rt-name/1234;svc/svc-namespace/svc-name", IDInfo{
				ParsedIDType: v1.GraphNodeTypeServicePort,
				Service: v1.ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "svc-namespace",
						Name:      "svc-name",
					},
					Protocol: "sctp",
					PortName: "po.rt-name",
					Port:     1234,
				},
			},
		),
		Entry("workload endpoint",
			"wep/ns1/n1/na1", IDInfo{
				ParsedIDType: v1.GraphNodeTypeWorkload,
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeWorkload,
					Namespace: "ns1",
					Name:      "n1",
					NameAggr:  "na1",
				},
			},
		),
		Entry("replica set",
			"rep/ns1/na1", IDInfo{
				ParsedIDType: v1.GraphNodeTypeReplicaSet,
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeReplicaSet,
					Namespace: "ns1",
					NameAggr:  "na1",
				},
			},
		),
		Entry("host endpoint",
			"host/na1/*", IDInfo{
				ParsedIDType: v1.GraphNodeTypeHost,
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeHost,
					Name:     "na1",
					NameAggr: "*",
				},
			},
		),
		Entry("host endpoint with service group",
			"host/na1/*;svcgp;svc/my-service-namespace/my-service-name", IDInfo{
				ParsedIDType: v1.GraphNodeTypeHost,
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeHost,
					Name:     "na1",
					NameAggr: "*",
				},
				ServiceGroup: dummySg,
			},
		),
		Entry("network",
			"net/na1", IDInfo{
				ParsedIDType: v1.GraphNodeTypeNetwork,
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetwork,
					NameAggr: "na1",
				},
			},
		),
		Entry("global network set",
			"ns/na1", IDInfo{
				ParsedIDType: v1.GraphNodeTypeNetworkSet,
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetworkSet,
					NameAggr: "na1",
				},
			},
		),
		Entry("global network set with a direction",
			"ns/na1;dir/egress", IDInfo{
				ParsedIDType: v1.GraphNodeTypeNetworkSet,
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetworkSet,
					NameAggr: "na1",
				},
				Direction: DirectionEgress,
			},
		),
		Entry("namespaced network set",
			"ns/ns1/na1", IDInfo{
				ParsedIDType: v1.GraphNodeTypeNetworkSet,
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeNetworkSet,
					Namespace: "ns1",
					NameAggr:  "na1",
				},
			},
		),
		Entry("network with service group",
			"net/na1;svcgp;svc/my-service-namespace/my-service-name", IDInfo{
				ParsedIDType: v1.GraphNodeTypeNetwork,
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetwork,
					NameAggr: "na1",
				},
				ServiceGroup: dummySg,
			},
		),
		Entry("global network set with service group",
			"ns/na1;svcgp;svc/my-service-namespace/my-service-name", IDInfo{
				ParsedIDType: v1.GraphNodeTypeNetworkSet,
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeNetworkSet,
					NameAggr: "na1",
				},
				ServiceGroup: dummySg,
			},
		),
		Entry("namespaced network set with service",
			"ns/ns1/na1;svcgp;svc/my-service-namespace/my-service-name", IDInfo{
				ParsedIDType: v1.GraphNodeTypeNetworkSet,
				Endpoint: FlowEndpoint{
					Type:      v1.GraphNodeTypeNetworkSet,
					Namespace: "ns1",
					NameAggr:  "na1",
				},
				ServiceGroup: dummySg,
			},
		),
		Entry("wildcarded hosts",
			"hosts/*", IDInfo{
				ParsedIDType: v1.GraphNodeTypeHosts,
				Endpoint: FlowEndpoint{
					Type:     v1.GraphNodeTypeHosts,
					NameAggr: "*",
				},
			},
		),
	)

	DescribeTable("Test invalid node id parsing",
		func(id string) {
			_, e := ParseGraphNodeID(v1.GraphNodeID(id), dummyServiceGroups)
			Expect(e).To(HaveOccurred())
		},
		Entry("layer - extra /", "layer/my/layer"),
		Entry("layer - with %", "layer/my%layer"),
		Entry("layer - with endpoint", "layer/my-layer;ns/na1"),
		Entry("layer - with service", "layer/my-layer;svc/ns1/n1"),
		Entry("network set - with too many segments", "ns/a/b/c"),
	)
})

type mockServiceGroups struct {
	ServiceGroups
	sg *ServiceGroup
}

func (m mockServiceGroups) GetByService(svc v1.NamespacedName) *ServiceGroup {
	return m.sg
}

func (m mockServiceGroups) GetByEndpoint(ep FlowEndpoint) *ServiceGroup {
	return nil
}
