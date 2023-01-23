// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package calc_test

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/proxy"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"

	kapiv1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/projectcalico/calico/felix/calc"
)

var _ = Describe("ServiceLookupsCache tests", func() {
	var sc *ServiceLookupsCache
	var updates []api.Update
	var key1 model.ResourceKey
	var spec1 kapiv1.ServiceSpec
	clusterIPStr := "10.0.0.0"
	clusterIP, _ := IPStringToArray(clusterIPStr)
	sv1Port := int32(123)
	sv1NodePort := int32(234)
	extIP1Str := "11.0.0.1"
	extIP1, _ := IPStringToArray(extIP1Str)
	extIP2Str := "11.0.0.2"
	extIP2, _ := IPStringToArray(extIP2Str)
	svc1 := proxy.ServicePortName{
		NamespacedName: types.NamespacedName{Name: "service1", Namespace: "ns1"},
		Port:           "namedport",
		Protocol:       kapiv1.ProtocolTCP,
	}
	svc1AggrPort := proxy.ServicePortName{
		NamespacedName: types.NamespacedName{Name: "service1", Namespace: "ns1"},
		Port:           "*",
		Protocol:       kapiv1.ProtocolTCP,
	}

	BeforeEach(func() {
		sc = NewServiceLookupsCache()

		By("adding a node and a service")
		key1 = model.ResourceKey{Kind: model.KindKubernetesService, Name: "service1", Namespace: "ns1"}
		spec1 = kapiv1.ServiceSpec{
			ClusterIP: clusterIPStr,
			ClusterIPs: []string{
				clusterIPStr,
			},
			ExternalIPs: []string{
				extIP1Str,
				extIP2Str,
			},
			Ports: []kapiv1.ServicePort{{
				Port:     sv1Port,
				NodePort: sv1NodePort,
				Protocol: kapiv1.ProtocolTCP,
				Name:     "namedport",
			}},
		}
		updates = []api.Update{{
			KVPair: model.KVPair{
				Key: key1,
				Value: &kapiv1.Service{
					Spec: spec1,
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		}}

		for _, u := range updates {
			sc.OnResourceUpdate(u)
		}
	})

	It("Should handle each type of lookup", func() {
		By("checking cluster IP and port of service 1")
		svc, ok := sc.GetServiceFromPreDNATDest(clusterIP, int(sv1Port), 6)
		Expect(ok).To(BeTrue())
		Expect(svc).To(Equal(svc1))

		By("checking external IP 1 and 2 of service 1")
		svc, ok = sc.GetServiceFromPreDNATDest(extIP1, int(sv1Port), 6)
		Expect(ok).To(BeTrue())
		Expect(svc).To(Equal(svc1))
		svc, ok = sc.GetServiceFromPreDNATDest(extIP2, int(sv1Port), 6)
		Expect(ok).To(BeTrue())
		Expect(svc).To(Equal(svc1))

		By("checking node port without DNAT")
		svc, ok = sc.GetNodePortService(int(sv1NodePort), 6)
		Expect(ok).To(BeTrue())
		Expect(svc).To(Equal(svc1))

		By("checking name and namespace (ResourceKey)")
		spec, ok := sc.GetServiceSpecFromResourceKey(key1)
		Expect(ok).To(BeTrue())
		Expect(spec).To(Equal(spec1))
	})

	It("Should handle multiple matching service ports", func() {
		By("updating a service to have multiple matching service ports")
		sc.OnResourceUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: model.KindKubernetesService, Name: "service1", Namespace: "ns1"},
				Value: &kapiv1.Service{
					Spec: kapiv1.ServiceSpec{
						ClusterIP: clusterIPStr,
						ClusterIPs: []string{
							clusterIPStr,
						},
						ExternalIPs: []string{
							extIP1Str,
							extIP2Str,
						},
						Ports: []kapiv1.ServicePort{{
							Port:     sv1Port,
							NodePort: sv1NodePort,
							Protocol: kapiv1.ProtocolTCP,
							Name:     "namedport",
						}, {
							Port:     sv1Port,
							NodePort: sv1NodePort + 1,
							Protocol: kapiv1.ProtocolTCP,
							Name:     "namedport2",
						}},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		By("checking the service port name is *")
		svc, ok := sc.GetServiceFromPreDNATDest(clusterIP, int(sv1Port), 6)
		Expect(ok).To(BeTrue())
		Expect(svc).To(Equal(svc1AggrPort))

		By("checking node port without DNAT is still a single port")
		svc, ok = sc.GetNodePortService(int(sv1NodePort), 6)
		Expect(ok).To(BeTrue())
		Expect(svc).To(Equal(svc1))
	})

	It("Should handle multiple matching service node ports", func() {
		By("updating a service to have multiple matching service node ports")
		sc.OnResourceUpdate(api.Update{
			KVPair: model.KVPair{
				Key: model.ResourceKey{Kind: model.KindKubernetesService, Name: "service1", Namespace: "ns1"},
				Value: &kapiv1.Service{
					Spec: kapiv1.ServiceSpec{
						ClusterIP: clusterIPStr,
						ClusterIPs: []string{
							clusterIPStr,
						},
						ExternalIPs: []string{
							extIP1Str,
							extIP2Str,
						},
						Ports: []kapiv1.ServicePort{{
							Port:     sv1Port,
							NodePort: sv1NodePort,
							Protocol: kapiv1.ProtocolTCP,
							Name:     "namedport",
						}, {
							Port:     sv1Port + 1,
							NodePort: sv1NodePort,
							Protocol: kapiv1.ProtocolTCP,
							Name:     "namedport2",
						}},
					},
				},
			},
			UpdateType: api.UpdateTypeKVNew,
		})

		By("checking the service port name is still a single port")
		svc, ok := sc.GetServiceFromPreDNATDest(clusterIP, int(sv1Port), 6)
		Expect(ok).To(BeTrue())
		Expect(svc).To(Equal(svc1))

		By("checking node port without DNAT is *")
		svc, ok = sc.GetNodePortService(int(sv1NodePort), 6)
		Expect(ok).To(BeTrue())
		Expect(svc).To(Equal(svc1AggrPort))
	})

	It("Should handle deletion of config", func() {
		By("deleting all resources")
		for _, u := range updates {
			sc.OnResourceUpdate(api.Update{
				KVPair:     model.KVPair{Key: u.Key},
				UpdateType: api.UpdateTypeKVDeleted,
			})
		}

		By("checking all IPs return no results")
		_, ok := sc.GetServiceFromPreDNATDest(clusterIP, int(sv1Port), 6)
		Expect(ok).To(BeFalse())

		_, ok = sc.GetServiceFromPreDNATDest(extIP1, int(sv1Port), 6)
		Expect(ok).To(BeFalse())

		_, ok = sc.GetServiceFromPreDNATDest(extIP2, int(sv1Port), 6)
		Expect(ok).To(BeFalse())

		By("checking node port return no results")
		_, ok = sc.GetNodePortService(int(sv1NodePort), 6)
		Expect(ok).To(BeFalse())
	})
})
