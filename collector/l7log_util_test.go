// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var _ = Describe("L7 log utility functions", func() {
	Describe("getAddressAndPort tests", func() {
		Context("With an IP and port", func() {
			It("Should properly split the IP and port", func() {
				addr, port := getAddressAndPort("10.10.10.10:80")
				Expect(addr).To(Equal("10.10.10.10"))
				Expect(port).To(Equal(80))
			})
		})

		Context("With an IP without a port", func() {
			It("Should properly return the IP", func() {
				addr, port := getAddressAndPort("10.10.10.10")
				Expect(addr).To(Equal("10.10.10.10"))
				Expect(port).To(Equal(0))
			})
		})

		Context("With a service name and port", func() {
			It("Should properly split the service name and port", func() {
				addr, port := getAddressAndPort("my-svc:80")
				Expect(addr).To(Equal("my-svc"))
				Expect(port).To(Equal(80))
			})
		})

		Context("With a service name and no port", func() {
			It("Should properly return the service name", func() {
				addr, port := getAddressAndPort("my-svc")
				Expect(addr).To(Equal("my-svc"))
				Expect(port).To(Equal(0))
			})
		})

		Context("With a malformed address", func() {
			It("Should not return anything", func() {
				addr, port := getAddressAndPort("asdf:qewr:asdf:jkl")
				Expect(addr).To(Equal(""))
				Expect(port).To(Equal(0))
			})
		})
	})

	Describe("extractK8sServiceNameAndNamespace tests", func() {
		Context("With a Kubernetes service DNS name", func() {
			It("Should properly extract the service name and namespace", func() {
				name, ns := extractK8sServiceNameAndNamespace("my-svc.svc-namespace.svc.cluster.local")
				Expect(name).To(Equal("my-svc"))
				Expect(ns).To(Equal("svc-namespace"))
			})
		})

		Context("With a Kubernetes service DNS name without a namespace", func() {
			It("Should properly extract the service name and namespace", func() {
				name, ns := extractK8sServiceNameAndNamespace("my-svc.svc.cluster.local")
				Expect(name).To(Equal("my-svc"))
				Expect(ns).To(Equal(""))
			})
		})

		Context("With a Kubernetes service DNS name with a subdomain", func() {
			It("Should properly extrac the service name, subdomain, and namespace", func() {
				name, ns := extractK8sServiceNameAndNamespace("my-svc.place.svc-namespace.svc.cluster.local")
				Expect(name).To(Equal("my-svc.place"))
				Expect(ns).To(Equal("svc-namespace"))
			})
		})

		Context("With an invalid Kubernetes service DNS name", func() {
			It("Should return nothing", func() {
				// Pod DNS
				name, ns := extractK8sServiceNameAndNamespace("my-pod.pod-namespace.pod.cluster.local")
				Expect(name).To(Equal(""))
				Expect(ns).To(Equal(""))

				// Non Kubernetes DNS
				name, ns = extractK8sServiceNameAndNamespace("my-external-svc.com")
				Expect(name).To(Equal(""))
				Expect(ns).To(Equal(""))
			})
		})
	})
})

var _ = Describe("Test L7 Aggregation options", func() {
	var update L7Update
	JustBeforeEach(func() {
		remoteWlEpKey1 := model.WorkloadEndpointKey{
			OrchestratorID: "orchestrator",
			WorkloadID:     "default/remoteworkloadid1",
			EndpointID:     "remoteepid1",
		}
		ed1 := &calc.EndpointData{
			Key:      remoteWlEpKey1,
			Endpoint: remoteWlEp1,
			IsLocal:  false,
		}
		remoteWlEpKey2 := model.WorkloadEndpointKey{
			OrchestratorID: "orchestrator",
			WorkloadID:     "default/remoteworkloadid2",
			EndpointID:     "remoteepid2",
		}
		ed2 := &calc.EndpointData{
			Key:      remoteWlEpKey2,
			Endpoint: remoteWlEp2,
			IsLocal:  false,
		}

		update = L7Update{
			Tuple:            *NewTuple(remoteIp1, remoteIp2, proto_tcp, srcPort, dstPort),
			SrcEp:            ed1,
			DstEp:            ed2,
			Duration:         10,
			DurationMax:      12,
			BytesReceived:    500,
			BytesSent:        30,
			ResponseCode:     "200",
			Method:           "GET",
			Domain:           "www.test.com",
			Path:             "/test/path?val=a",
			UserAgent:        "firefox",
			Type:             "html/1.1",
			Count:            1,
			ServiceName:      "test-service",
			ServiceNamespace: "test-namespace",
			ServicePort:      80,
		}
	})

	It("Should return all data when there is no aggregation on all fields", func() {
		agg := L7AggregationKind{
			HTTPHeader:      L7HTTPHeaderInfo,
			HTTPMethod:      L7HTTPMethod,
			Service:         L7ServiceInfo,
			Destination:     L7DestinationInfo,
			Source:          L7SourceInfo,
			TrimURL:         L7FullURL,
			ResponseCode:    L7ResponseCode,
			NumURLPathParts: -1,
		}

		meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
		Expect(err).To(BeNil())
		Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
		Expect(meta.Method).To(Equal(update.Method))
		Expect(meta.Domain).To(Equal(update.Domain))
		Expect(meta.Path).To(Equal(update.Path))
		Expect(meta.UserAgent).To(Equal(update.UserAgent))
		Expect(meta.Type).To(Equal(update.Type))
		Expect(meta.ServiceName).To(Equal(update.ServiceName))
		Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
		Expect(meta.ServicePort).To(Equal(update.ServicePort))
		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DstNamespace).To(Equal("default"))
		Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
	})

	It("Should aggregate out the correct HTTP header details appropriately", func() {
		agg := L7AggregationKind{
			HTTPHeader:      L7HTTPHeaderInfoNone,
			HTTPMethod:      L7HTTPMethod,
			Service:         L7ServiceInfo,
			Destination:     L7DestinationInfo,
			Source:          L7SourceInfo,
			TrimURL:         L7FullURL,
			ResponseCode:    L7ResponseCode,
			NumURLPathParts: -1,
		}

		meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
		Expect(err).To(BeNil())
		Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
		Expect(meta.Method).To(Equal(update.Method))
		Expect(meta.Domain).To(Equal(update.Domain))
		Expect(meta.Path).To(Equal(update.Path))
		Expect(meta.UserAgent).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.Type).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.ServiceName).To(Equal(update.ServiceName))
		Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
		Expect(meta.ServicePort).To(Equal(update.ServicePort))
		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DstNamespace).To(Equal("default"))
		Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
	})

	It("Should aggregate out the HTTP method", func() {
		agg := L7AggregationKind{
			HTTPHeader:      L7HTTPHeaderInfo,
			HTTPMethod:      L7HTTPMethodNone,
			Service:         L7ServiceInfo,
			Destination:     L7DestinationInfo,
			Source:          L7SourceInfo,
			TrimURL:         L7FullURL,
			ResponseCode:    L7ResponseCode,
			NumURLPathParts: -1,
		}

		meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
		Expect(err).To(BeNil())
		Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
		Expect(meta.Method).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.Domain).To(Equal(update.Domain))
		Expect(meta.Path).To(Equal(update.Path))
		Expect(meta.UserAgent).To(Equal(update.UserAgent))
		Expect(meta.Type).To(Equal(update.Type))
		Expect(meta.ServiceName).To(Equal(update.ServiceName))
		Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
		Expect(meta.ServicePort).To(Equal(update.ServicePort))
		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DstNamespace).To(Equal("default"))
		Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
	})

	It("Should aggregate over service information properly", func() {
		agg := L7AggregationKind{
			HTTPHeader:      L7HTTPHeaderInfo,
			HTTPMethod:      L7HTTPMethod,
			Service:         L7ServiceInfoNone,
			Destination:     L7DestinationInfo,
			Source:          L7SourceInfo,
			TrimURL:         L7FullURL,
			ResponseCode:    L7ResponseCode,
			NumURLPathParts: -1,
		}

		meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
		Expect(err).To(BeNil())
		Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
		Expect(meta.Method).To(Equal(update.Method))
		Expect(meta.Domain).To(Equal(update.Domain))
		Expect(meta.Path).To(Equal(update.Path))
		Expect(meta.UserAgent).To(Equal(update.UserAgent))
		Expect(meta.Type).To(Equal(update.Type))
		Expect(meta.ServiceName).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.ServiceNamespace).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.ServicePort).To(Equal(0))
		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DstNamespace).To(Equal("default"))
		Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
	})

	It("Should aggregate out the destination information properly", func() {
		agg := L7AggregationKind{
			HTTPHeader:      L7HTTPHeaderInfo,
			HTTPMethod:      L7HTTPMethod,
			Service:         L7ServiceInfo,
			Destination:     L7DestinationInfoNone,
			Source:          L7SourceInfo,
			TrimURL:         L7FullURL,
			ResponseCode:    L7ResponseCode,
			NumURLPathParts: -1,
		}

		meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
		Expect(err).To(BeNil())
		Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
		Expect(meta.Method).To(Equal(update.Method))
		Expect(meta.Domain).To(Equal(update.Domain))
		Expect(meta.Path).To(Equal(update.Path))
		Expect(meta.UserAgent).To(Equal(update.UserAgent))
		Expect(meta.Type).To(Equal(update.Type))
		Expect(meta.ServiceName).To(Equal(update.ServiceName))
		Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
		Expect(meta.ServicePort).To(Equal(update.ServicePort))
		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DstNameAggr).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.DstNamespace).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.DstType).To(Equal(FlowLogEndpointType(flowLogFieldNotIncluded)))
	})

	It("Should aggregate out the source information properly", func() {
		agg := L7AggregationKind{
			HTTPHeader:      L7HTTPHeaderInfo,
			HTTPMethod:      L7HTTPMethod,
			Service:         L7ServiceInfo,
			Destination:     L7DestinationInfo,
			Source:          L7SourceInfoNone,
			TrimURL:         L7FullURL,
			ResponseCode:    L7ResponseCode,
			NumURLPathParts: -1,
		}

		meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
		Expect(err).To(BeNil())
		Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
		Expect(meta.Method).To(Equal(update.Method))
		Expect(meta.Domain).To(Equal(update.Domain))
		Expect(meta.Path).To(Equal(update.Path))
		Expect(meta.UserAgent).To(Equal(update.UserAgent))
		Expect(meta.Type).To(Equal(update.Type))
		Expect(meta.ServiceName).To(Equal(update.ServiceName))
		Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
		Expect(meta.ServicePort).To(Equal(update.ServicePort))
		Expect(meta.SrcNameAggr).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.SrcNamespace).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointType(flowLogFieldNotIncluded)))
		Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DstNamespace).To(Equal("default"))
		Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
	})

	It("Should aggregate out the response code", func() {
		agg := L7AggregationKind{
			HTTPHeader:      L7HTTPHeaderInfo,
			HTTPMethod:      L7HTTPMethod,
			Service:         L7ServiceInfo,
			Destination:     L7DestinationInfo,
			Source:          L7SourceInfo,
			TrimURL:         L7FullURL,
			ResponseCode:    L7ResponseCodeNone,
			NumURLPathParts: -1,
		}

		meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
		Expect(err).To(BeNil())
		Expect(meta.ResponseCode).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.Method).To(Equal(update.Method))
		Expect(meta.Domain).To(Equal(update.Domain))
		Expect(meta.Path).To(Equal(update.Path))
		Expect(meta.UserAgent).To(Equal(update.UserAgent))
		Expect(meta.Type).To(Equal(update.Type))
		Expect(meta.ServiceName).To(Equal(update.ServiceName))
		Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
		Expect(meta.ServicePort).To(Equal(update.ServicePort))
		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DstNamespace).To(Equal("default"))
		Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
	})

	Context("With URL aggregating on", func() {
		It("Should remove the entire URL", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7URLNone,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: -1,
			}

			meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
			Expect(err).To(BeNil())
			Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
			Expect(meta.Method).To(Equal(update.Method))
			Expect(meta.Domain).To(Equal(flowLogFieldNotIncluded))
			Expect(meta.Path).To(Equal(flowLogFieldNotIncluded))
			Expect(meta.UserAgent).To(Equal(update.UserAgent))
			Expect(meta.Type).To(Equal(update.Type))
			Expect(meta.ServiceName).To(Equal(update.ServiceName))
			Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
			Expect(meta.ServicePort).To(Equal(update.ServicePort))
			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DstNamespace).To(Equal("default"))
			Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
		})

		It("Should remove only the query parameters", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7URLWithoutQuery,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: -1,
			}

			meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
			Expect(err).To(BeNil())
			Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
			Expect(meta.Method).To(Equal(update.Method))
			Expect(meta.Domain).To(Equal(update.Domain))
			Expect(meta.Path).To(Equal("/test/path"))
			Expect(meta.UserAgent).To(Equal(update.UserAgent))
			Expect(meta.Type).To(Equal(update.Type))
			Expect(meta.ServiceName).To(Equal(update.ServiceName))
			Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
			Expect(meta.ServicePort).To(Equal(update.ServicePort))
			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DstNamespace).To(Equal("default"))
			Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
		})

		It("Should remove the path", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7BaseURL,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: -1,
			}

			meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
			Expect(err).To(BeNil())
			Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
			Expect(meta.Method).To(Equal(update.Method))
			Expect(meta.Domain).To(Equal(update.Domain))
			Expect(meta.Path).To(Equal(flowLogFieldNotIncluded))
			Expect(meta.UserAgent).To(Equal(update.UserAgent))
			Expect(meta.Type).To(Equal(update.Type))
			Expect(meta.ServiceName).To(Equal(update.ServiceName))
			Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
			Expect(meta.ServicePort).To(Equal(update.ServicePort))
			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DstNamespace).To(Equal("default"))
			Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
		})

		It("Should properly truncate parts of the URL path", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7FullURL,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: 1,
			}

			meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
			Expect(err).To(BeNil())
			Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
			Expect(meta.Method).To(Equal(update.Method))
			Expect(meta.Domain).To(Equal(update.Domain))
			Expect(meta.Path).To(Equal("/test"))
			Expect(meta.UserAgent).To(Equal(update.UserAgent))
			Expect(meta.Type).To(Equal(update.Type))
			Expect(meta.ServiceName).To(Equal(update.ServiceName))
			Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
			Expect(meta.ServicePort).To(Equal(update.ServicePort))
			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DstNamespace).To(Equal("default"))
			Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
		})

		It("Should properly truncate all parts of the URL path", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7FullURL,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: 0,
			}

			meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
			Expect(err).To(BeNil())
			Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
			Expect(meta.Method).To(Equal(update.Method))
			Expect(meta.Domain).To(Equal(update.Domain))
			Expect(meta.Path).To(Equal(""))
			Expect(meta.UserAgent).To(Equal(update.UserAgent))
			Expect(meta.Type).To(Equal(update.Type))
			Expect(meta.ServiceName).To(Equal(update.ServiceName))
			Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
			Expect(meta.ServicePort).To(Equal(update.ServicePort))
			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DstNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DstNamespace).To(Equal("default"))
			Expect(meta.DstType).To(Equal(FlowLogEndpointTypeWep))
		})
	})
})
