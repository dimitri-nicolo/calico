// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

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
			ServicePortName:  "test-port",
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
			URLCharLimit:    28,
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
		Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.SourcePortNum).To(Equal(srcPort))

		Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DestNamespace).To(Equal("default"))
		Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DestPortNum).To(Equal(dstPort))
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
			URLCharLimit:    28,
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
		Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.SourcePortNum).To(Equal(srcPort))

		Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DestNamespace).To(Equal("default"))
		Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DestPortNum).To(Equal(dstPort))
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
			URLCharLimit:    28,
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
		Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

		Expect(meta.ServicePortName).To(Equal(update.ServicePortName))
		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.SourcePortNum).To(Equal(srcPort))

		Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DestNamespace).To(Equal("default"))
		Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DestPortNum).To(Equal(dstPort))
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
			URLCharLimit:    28,
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

		Expect(meta.ServicePortName).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.SourcePortNum).To(Equal(srcPort))

		Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DestNamespace).To(Equal("default"))
		Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DestPortNum).To(Equal(dstPort))
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
			URLCharLimit:    28,
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
		Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.SourcePortNum).To(Equal(srcPort))

		Expect(meta.DestNameAggr).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.DestNamespace).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.DestType).To(Equal(FlowLogEndpointType(flowLogFieldNotIncluded)))
		Expect(meta.DestPortNum).To(Equal(0))
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
			URLCharLimit:    28,
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
		Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

		Expect(meta.SrcNameAggr).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.SrcNamespace).To(Equal(flowLogFieldNotIncluded))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointType(flowLogFieldNotIncluded)))
		Expect(meta.SourcePortNum).To(Equal(0))

		Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DestNamespace).To(Equal("default"))
		Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DestPortNum).To(Equal(dstPort))
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
			URLCharLimit:    28,
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
		Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

		Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
		Expect(meta.SrcNamespace).To(Equal("default"))
		Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.SourcePortNum).To(Equal(srcPort))

		Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
		Expect(meta.DestNamespace).To(Equal("default"))
		Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
		Expect(meta.DestPortNum).To(Equal(dstPort))
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
				URLCharLimit:    28,
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
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
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
				URLCharLimit:    28,
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
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
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
				URLCharLimit:    28,
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
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
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
				URLCharLimit:    28,
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
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
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
				URLCharLimit:    28,
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
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
		})

		It("Should output full url with query params when URLCharLimit is more than length of the URL", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7FullURL,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: -1,
				URLCharLimit:    40,
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
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
		})

		It("Should output truncated domain, empty path when URLCharLimit is less than length of domain", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7FullURL,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: -1,
				URLCharLimit:    10,
			}

			meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
			Expect(err).To(BeNil())
			Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
			Expect(meta.Method).To(Equal(update.Method))
			Expect(meta.Domain).To(Equal("www.test.c"))
			Expect(meta.Path).To(Equal(flowLogFieldNotIncluded))
			Expect(meta.UserAgent).To(Equal(update.UserAgent))
			Expect(meta.Type).To(Equal(update.Type))
			Expect(meta.ServiceName).To(Equal(update.ServiceName))
			Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
		})

		It("Should output full domain, parts of path when URLCharLimit is more than domain length but less than full path url", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7FullURL,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: 5,
				URLCharLimit:    15,
			}

			meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
			Expect(err).To(BeNil())
			Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
			Expect(meta.Method).To(Equal(update.Method))
			Expect(meta.Domain).To(Equal(update.Domain))
			Expect(meta.Path).To(Equal("/te"))
			Expect(meta.UserAgent).To(Equal(update.UserAgent))
			Expect(meta.Type).To(Equal(update.Type))
			Expect(meta.ServiceName).To(Equal(update.ServiceName))
			Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
		})

		It("Should output empty domain and path for L7URLNone case no matter what URLCharLimit is passed", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7URLNone,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: 5,
				URLCharLimit:    15,
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
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
		})

		It("Should output full domain and empty path for L7BaseURL case when URLCharLimit is more than domain length", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7BaseURL,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: 5,
				URLCharLimit:    20,
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
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
		})

		It("Should output full domain and max path for L7URLWithoutQuery case when URLCharLimit is between domain length and full url length", func() {
			agg := L7AggregationKind{
				HTTPHeader:      L7HTTPHeaderInfo,
				HTTPMethod:      L7HTTPMethod,
				Service:         L7ServiceInfo,
				Destination:     L7DestinationInfo,
				Source:          L7SourceInfo,
				TrimURL:         L7URLWithoutQuery,
				ResponseCode:    L7ResponseCode,
				NumURLPathParts: 5,
				URLCharLimit:    20,
			}

			meta, _, err := NewL7MetaSpecFromUpdate(update, agg)
			Expect(err).To(BeNil())
			Expect(meta.ResponseCode).To(Equal(update.ResponseCode))
			Expect(meta.Method).To(Equal(update.Method))
			Expect(meta.Domain).To(Equal(update.Domain))
			Expect(meta.Path).To(Equal("/test/pa"))
			Expect(meta.UserAgent).To(Equal(update.UserAgent))
			Expect(meta.Type).To(Equal(update.Type))
			Expect(meta.ServiceName).To(Equal(update.ServiceName))
			Expect(meta.ServiceNamespace).To(Equal(update.ServiceNamespace))
			Expect(meta.ServicePortName).To(Equal(update.ServicePortName))

			Expect(meta.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(meta.SrcNamespace).To(Equal("default"))
			Expect(meta.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.SourcePortNum).To(Equal(srcPort))

			Expect(meta.DestNameAggr).To(Equal("remoteworkloadid2"))
			Expect(meta.DestNamespace).To(Equal("default"))
			Expect(meta.DestType).To(Equal(FlowLogEndpointTypeWep))
			Expect(meta.DestPortNum).To(Equal(dstPort))
		})
	})
})
