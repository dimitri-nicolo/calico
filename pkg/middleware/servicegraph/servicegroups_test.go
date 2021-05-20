// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "github.com/tigera/es-proxy/pkg/apis/v1"

	. "github.com/tigera/es-proxy/pkg/middleware/servicegraph"
)

type serviceGroupInput struct {
	ServicePort
	FlowEndpoint
}

var _ = Describe("ServicePort relationships test", func() {
	DescribeTable("Basic service tests",
		func(sgis []serviceGroupInput, results []ServiceGroup) {
			srh := NewServiceGroups()

			for _, sgi := range sgis {
				srh.AddMapping(sgi.ServicePort, sgi.FlowEndpoint)
			}
			srh.FinishMappings()
			actualMap := make(map[v1.GraphNodeID]ServiceGroup)
			err := srh.Iter(func(sg *ServiceGroup) error {
				actualMap[sg.ID] = *sg
				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			for i := range results {
				Expect(actualMap).To(HaveKey(results[i].ID))
				actual := actualMap[results[i].ID]

				Expect(actual.Namespace).To(Equal(results[i].Namespace))
				Expect(actual.Name).To(Equal(results[i].Name))
				Expect(actual.Services).To(ConsistOf(results[i].Services))
				Expect(actual.ServicePorts).To(Equal(results[i].ServicePorts))
			}
		},
		Entry("Two services related by a third through Endpoints",
			[]serviceGroupInput{{
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service1",
					},
					Port:  "",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "rep",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "name1*",
					Port:      9443,
					Proto:     "tcp",
				},
			}, {
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service1",
					},
					Port:  "",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "rep",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "name2*",
					Port:      9443,
					Proto:     "tcp",
				},
			}, {
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service2",
					},
					Port:  "",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "rep",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "name3*",
					Port:      9443,
					Proto:     "tcp",
				},
			}, {
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service2",
					},
					Port:  "",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "rep",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "name4*",
					Port:      9443,
					Proto:     "tcp",
				},
			}, {
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service3",
					},
					Port:  "",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "rep",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "name1*",
					Port:      9443,
					Proto:     "tcp",
				},
			}, {
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service3",
					},
					Port:  "",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "rep",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "name4*",
					Port:      9443,
					Proto:     "tcp",
				},
			}},
			[]ServiceGroup{{
				ID:        "svcgp;svc/namespace1/service1;svc/namespace1/service2;svc/namespace1/service3",
				Name:      "service1/service2/service3",
				Namespace: "namespace1",
				Services: []v1.NamespacedName{{
					Namespace: "namespace1", Name: "service1",
				}, {
					Namespace: "namespace1", Name: "service2",
				}, {
					Namespace: "namespace1", Name: "service3",
				}},
				ServicePorts: map[ServicePort]map[FlowEndpoint]struct{}{
					ServicePort{
						NamespacedName: v1.NamespacedName{
							Name: "service1", Namespace: "namespace1",
						},
						Proto: "tcp",
					}: {
						FlowEndpoint{
							Type: "rep", Namespace: "namespace1", NameAggr: "name1*", Name: "", Port: 9443, Proto: "tcp",
						}: struct{}{},
						FlowEndpoint{
							Type: "rep", Namespace: "namespace1", NameAggr: "name2*", Name: "", Port: 9443, Proto: "tcp",
						}: struct{}{},
					},
					ServicePort{
						NamespacedName: v1.NamespacedName{
							Name: "service2", Namespace: "namespace1",
						},
						Proto: "tcp",
					}: {
						FlowEndpoint{
							Type: "rep", Namespace: "namespace1", NameAggr: "name3*", Name: "", Port: 9443, Proto: "tcp",
						}: struct{}{},
						FlowEndpoint{
							Type: "rep", Namespace: "namespace1", NameAggr: "name4*", Name: "", Port: 9443, Proto: "tcp",
						}: struct{}{},
					},
					ServicePort{
						NamespacedName: v1.NamespacedName{
							Name: "service3", Namespace: "namespace1",
						},
						Proto: "tcp",
					}: {
						FlowEndpoint{
							Type: "rep", Namespace: "namespace1", NameAggr: "name1*", Name: "", Port: 9443, Proto: "tcp",
						}: struct{}{},
						FlowEndpoint{
							Type: "rep", Namespace: "namespace1", NameAggr: "name4*", Name: "", Port: 9443, Proto: "tcp",
						}: struct{}{},
					},
				},
			}},
		),

		Entry("Two services different ports related by service",
			[]serviceGroupInput{{
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service1",
					},
					Port:  "port1",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "rep",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "name1*",
					Port:      9443,
					Proto:     "tcp",
				},
			}, {
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service1",
					},
					Port:  "port2",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "rep",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "name2*",
					Port:      9444,
					Proto:     "tcp",
				},
			}},
			[]ServiceGroup{{
				ID:        "svcgp;svc/namespace1/service1",
				Name:      "service1",
				Namespace: "namespace1",
				Services: []v1.NamespacedName{{
					Namespace: "namespace1", Name: "service1",
				}},
				ServicePorts: map[ServicePort]map[FlowEndpoint]struct{}{
					ServicePort{
						NamespacedName: v1.NamespacedName{
							Name: "service1", Namespace: "namespace1",
						},
						Port: "port1", Proto: "tcp",
					}: {
						FlowEndpoint{
							Type: "rep", Namespace: "namespace1", NameAggr: "name1*", Name: "", Port: 9443, Proto: "tcp",
						}: struct{}{},
					},
					ServicePort{
						NamespacedName: v1.NamespacedName{
							Name: "service1", Namespace: "namespace1",
						},
						Port: "port2", Proto: "tcp",
					}: {
						FlowEndpoint{
							Type: "rep", Namespace: "namespace1", NameAggr: "name2*", Name: "", Port: 9444, Proto: "tcp",
						}: struct{}{},
					},
				},
			}},
		),

		Entry("Two services using different ports in the same network set",
			[]serviceGroupInput{{
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service1",
					},
					Port:  "port1",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "ns",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "net1",
					Port:      9443,
					Proto:     "tcp",
				},
			}, {
				ServicePort{
					NamespacedName: v1.NamespacedName{
						Namespace: "namespace1",
						Name:      "service2",
					},
					Port:  "port2",
					Proto: "tcp",
				},
				FlowEndpoint{
					Type:      "ns",
					Namespace: "namespace1",
					Name:      "",
					NameAggr:  "net1",
					Port:      9444,
					Proto:     "tcp",
				},
			}},
			// Different ports in the same network set are not treated as identical Endpoints so these services will
			// not be grouped together.
			[]ServiceGroup{{
				ID:        "svcgp;svc/namespace1/service1",
				Name:      "service1",
				Namespace: "namespace1",
				Services: []v1.NamespacedName{{
					Namespace: "namespace1", Name: "service1",
				}},
				ServicePorts: map[ServicePort]map[FlowEndpoint]struct{}{
					ServicePort{
						NamespacedName: v1.NamespacedName{
							Name: "service1", Namespace: "namespace1",
						},
						Port: "port1", Proto: "tcp",
					}: {
						FlowEndpoint{
							Type: "ns", Namespace: "namespace1", NameAggr: "net1", Name: "", Port: 9443, Proto: "tcp",
						}: struct{}{},
					},
				},
			}, {
				ID:        "svcgp;svc/namespace1/service2",
				Name:      "service2",
				Namespace: "namespace1",
				Services: []v1.NamespacedName{{
					Namespace: "namespace1", Name: "service2",
				}},
				ServicePorts: map[ServicePort]map[FlowEndpoint]struct{}{
					ServicePort{
						NamespacedName: v1.NamespacedName{
							Name: "service2", Namespace: "namespace1",
						},
						Port: "port2", Proto: "tcp",
					}: {
						FlowEndpoint{
							Type: "ns", Namespace: "namespace1", NameAggr: "net1", Name: "", Port: 9444, Proto: "tcp",
						}: struct{}{},
					},
				},
			}},
		),
	)
})
