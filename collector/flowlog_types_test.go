// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

// Common MetricUpdate definitions
var (
	// Metric update without a connection (ingress stats match those of muConn1Rule1AllowUpdate).
	muWithEndpointMeta = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple1,

		srcEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-01",
				OrchestratorID: "k8s",
				WorkloadID:     "kube-system/iperf-4235-5623461",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235-", Labels: map[string]string{"test-app": "true"}},
		},

		dstEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-02",
				OrchestratorID: "k8s",
				WorkloadID:     "default/nginx-412354-5123451",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354-", Labels: map[string]string{"k8s-app": "true"}},
		},

		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	muWithoutSrcEndpointMeta = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple1,

		srcEp: nil,

		dstEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-02",
				OrchestratorID: "k8s",
				WorkloadID:     "default/nginx-412354-5123451",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354-", Labels: map[string]string{"k8s-app": "true"}},
		},

		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	muWithoutDstEndpointMeta = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple1,

		srcEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-01",
				OrchestratorID: "k8s",
				WorkloadID:     "kube-system/iperf-4235-5623461",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235-", Labels: map[string]string{"test-app": "true"}},
		},

		dstEp: nil,

		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}
)

var _ = Describe("Flow log types tests", func() {
	var flowMeta, expectedFlowMeta FlowMeta
	var err error

	Context("FlowMeta construction from MetricUpdate", func() {
		It("generates the correct FlowMeta", func() {
			By("aggregating on duration")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, FlowDefault)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 0, 0, 1},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 20, 0, 0, 1},
					proto: 6,
					l4Src: 54123,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "iperf-4235-5623461",
					AggregatedName: "iperf-4235-*",
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "default",
					Name:           "nginx-412354-5123451",
					AggregatedName: "nginx-412354-*",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			flowMeta, err = NewFlowMeta(muWithoutSrcEndpointMeta, FlowDefault)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 0, 0, 1},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 20, 0, 0, 1},
					proto: 6,
					l4Src: 54123,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "net",
					Namespace:      "-",
					Name:           "-",
					AggregatedName: "pvt",
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "default",
					Name:           "nginx-412354-5123451",
					AggregatedName: "nginx-412354-*",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			flowMeta, err = NewFlowMeta(muWithoutDstEndpointMeta, FlowDefault)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 0, 0, 1},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 20, 0, 0, 1},
					proto: 6,
					l4Src: 54123,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "iperf-4235-5623461",
					AggregatedName: "iperf-4235-*",
				},
				DstMeta: EndpointMetadata{
					Type:           "net",
					Namespace:      "-",
					Name:           "-",
					AggregatedName: "pub",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			By("aggregating on source port")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, FlowSourcePort)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 0, 0, 1},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 20, 0, 0, 1},
					proto: 6,
					l4Src: -1, // Is the only attribute that gets disregarded.
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "iperf-4235-5623461",
					AggregatedName: "iperf-4235-*",
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "default",
					Name:           "nginx-412354-5123451",
					AggregatedName: "nginx-412354-*",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			By("aggregating on prefix name")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, FlowPrefixName)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					proto: 6,
					l4Src: -1, // Is the only attribute that gets disregarded.
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "-",
					AggregatedName: "iperf-4235-*", // Keeping just the Generate Name
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "default",
					Name:           "-",
					AggregatedName: "nginx-412354-*",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			flowMeta, err = NewFlowMeta(muWithoutSrcEndpointMeta, FlowPrefixName)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					proto: 6,
					l4Src: -1, // Is the only attribute that gets disregarded.
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "net", // No EndpointMeta associated but Src IP Private
					Namespace:      "-",
					Name:           "-",
					AggregatedName: "pvt",
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "default",
					Name:           "-",
					AggregatedName: "nginx-412354-*",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			muWithoutPublicDstEndpointMeta := muWithoutDstEndpointMeta
			muWithoutPublicDstEndpointMeta.tuple.dst = ipStrTo16Byte("198.17.8.43")
			flowMeta, err = NewFlowMeta(muWithoutPublicDstEndpointMeta, FlowPrefixName)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					proto: 6,
					l4Src: -1,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "-",
					AggregatedName: "iperf-4235-*", // Keeping just the Generate Name
				},
				DstMeta: EndpointMetadata{
					Type:           "net", // No EndpointMeta associated but Dst IP Public
					Namespace:      "-",
					Name:           "-",
					AggregatedName: "pub",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			muWithoutAWSMetaDstEndpointMeta := muWithoutDstEndpointMeta
			muWithoutAWSMetaDstEndpointMeta.tuple.dst = ipStrTo16Byte("169.254.169.254")
			flowMeta, err = NewFlowMeta(muWithoutAWSMetaDstEndpointMeta, FlowPrefixName)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					proto: 6,
					l4Src: -1, // Is the only attribute that gets disregarded.
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "-",
					AggregatedName: "iperf-4235-*", // Keeping just the Generate Name
				},
				DstMeta: EndpointMetadata{
					Type:           "net", // No EndpointMeta associated but Dst IP AWS Metadata Server
					Namespace:      "-",
					Name:           "-",
					AggregatedName: "aws",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			muWithEndpointMetaWithoutGenerateName := muWithEndpointMeta
			muWithEndpointMetaWithoutGenerateName.dstEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-02",
					OrchestratorID: "k8s",
					WorkloadID:     "default/manually-created-pod",
					EndpointID:     "4352",
				},
				Endpoint: &model.WorkloadEndpoint{GenerateName: "", Labels: map[string]string{"k8s-app": "true"}},
			}
			flowMeta, err = NewFlowMeta(muWithEndpointMetaWithoutGenerateName, FlowPrefixName)
			Expect(err).To(BeNil())
			expectedFlowMeta = FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					proto: 6,
					l4Src: -1,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "-",
					AggregatedName: "iperf-4235-*", // Keeping just the Generate Name
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "default",
					Name:           "-",
					AggregatedName: "manually-created-pod", // Keeping the Name. No Generatename.
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))
		})
	})

})
