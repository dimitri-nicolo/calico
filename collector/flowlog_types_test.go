// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

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

		ruleID:       ingressRule1Allow,
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

		ruleID:       ingressRule1Allow,
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

		ruleID:       ingressRule1Allow,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}
)

var _ = Describe("Flow log types tests", func() {
	var flowMeta, expectedFlowMeta FlowMeta
	var flowStats FlowStats
	var flowLog, expectedFlowLog string
	var err error
	Context("FlowMeta construction from MetricUpdate", func() {
		It("generates the correct FlowMeta", func() {
			By("aggregating on duration")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, Default)
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
					Type:      "wep",
					Namespace: "kube-system",
					Name:      "iperf-4235-5623461",
					Labels:    "{\"test-app\":\"true\"}",
				},
				DstMeta: EndpointMetadata{
					Type:      "wep",
					Namespace: "default",
					Name:      "nginx-412354-5123451",
					Labels:    "{\"k8s-app\":\"true\"}",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			flowMeta, err = NewFlowMeta(muWithoutSrcEndpointMeta, Default)
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
					Type:      "net",
					Namespace: "-",
					Name:      "pvt",
					Labels:    "-",
				},
				DstMeta: EndpointMetadata{
					Type:      "wep",
					Namespace: "default",
					Name:      "nginx-412354-5123451",
					Labels:    "{\"k8s-app\":\"true\"}",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			flowMeta, err = NewFlowMeta(muWithoutDstEndpointMeta, Default)
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
					Type:      "wep",
					Namespace: "kube-system",
					Name:      "iperf-4235-5623461",
					Labels:    "{\"test-app\":\"true\"}",
				},
				DstMeta: EndpointMetadata{
					Type:      "net",
					Namespace: "-",
					Name:      "pub",
					Labels:    "-",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			By("aggregating on source port")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, SourcePort)
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
					Type:      "wep",
					Namespace: "kube-system",
					Name:      "iperf-4235-5623461",
					Labels:    "{\"test-app\":\"true\"}",
				},
				DstMeta: EndpointMetadata{
					Type:      "wep",
					Namespace: "default",
					Name:      "nginx-412354-5123451",
					Labels:    "{\"k8s-app\":\"true\"}",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			By("aggregating on prefix name")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, PrefixName)
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
					Type:      "wep",
					Namespace: "kube-system",
					Name:      "iperf-4235-*", // Keeping just the Generate Name
					Labels:    "-",            // Disregarding the labels
				},
				DstMeta: EndpointMetadata{
					Type:      "wep",
					Namespace: "default",
					Name:      "nginx-412354-*",
					Labels:    "-",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			flowMeta, err = NewFlowMeta(muWithoutSrcEndpointMeta, PrefixName)
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
					Type:      "net", // No EndpointMeta associated but Src IP Private
					Namespace: "-",
					Name:      "pvt",
					Labels:    "-",
				},
				DstMeta: EndpointMetadata{
					Type:      "wep",
					Namespace: "default",
					Name:      "nginx-412354-*",
					Labels:    "-",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			muWithoutPublicDstEndpointMeta := muWithoutDstEndpointMeta
			muWithoutPublicDstEndpointMeta.tuple.dst = ipStrTo16Byte("198.17.8.43")
			flowMeta, err = NewFlowMeta(muWithoutPublicDstEndpointMeta, PrefixName)
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
					Type:      "wep",
					Namespace: "kube-system",
					Name:      "iperf-4235-*", // Keeping just the Generate Name
					Labels:    "-",            // Disregarding the labels
				},
				DstMeta: EndpointMetadata{
					Type:      "net", // No EndpointMeta associated but Dst IP Public
					Namespace: "-",
					Name:      "pub",
					Labels:    "-",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))

			muWithoutAWSMetaDstEndpointMeta := muWithoutDstEndpointMeta
			muWithoutAWSMetaDstEndpointMeta.tuple.dst = ipStrTo16Byte("169.254.169.254")
			flowMeta, err = NewFlowMeta(muWithoutAWSMetaDstEndpointMeta, PrefixName)
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
					Type:      "wep",
					Namespace: "kube-system",
					Name:      "iperf-4235-*", // Keeping just the Generate Name
					Labels:    "-",            // Disregarding the labels
				},
				DstMeta: EndpointMetadata{
					Type:      "net", // No EndpointMeta associated but Dst IP AWS Metadata Server
					Namespace: "-",
					Name:      "aws", // <-- AWS MetaServer Endpoint
					Labels:    "-",
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
			flowMeta, err = NewFlowMeta(muWithEndpointMetaWithoutGenerateName, PrefixName)
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
					Type:      "wep",
					Namespace: "kube-system",
					Name:      "iperf-4235-*", // Keeping just the Generate Name
					Labels:    "-",
				},
				DstMeta: EndpointMetadata{
					Type:      "wep",
					Namespace: "default",
					Name:      "manually-created-pod", // Keeping the Name. No Generatename.
					Labels:    "-",
				},
				Action:   "allow",
				Reporter: "dst",
			}
			Expect(flowMeta).Should(Equal(expectedFlowMeta))
		})
	})

	Context("FlowLog Serialization", func() {
		It("generates the correct FlowLog string", func() {
			flowStats = FlowStats{}
			startTime := time.Date(2017, 11, 17, 20, 1, 0, 0, time.UTC)
			endTime := time.Date(2017, 11, 17, 20, 2, 0, 0, time.UTC)

			By("skipping aggergation")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, Default)
			Expect(err).To(BeNil())
			flowLog = FlowLog{flowMeta, flowStats}.Serialize(startTime, endTime, false)
			expectedFlowLog = "1510948860 1510948920 wep kube-system iperf-4235-5623461 - wep default nginx-412354-5123451 - 10.0.0.1 20.0.0.1 6 54123 80 0 0 0 dst 0 0 0 0 allow"
			Expect(flowLog).Should(Equal(expectedFlowLog))

			flowLog = FlowLog{flowMeta, flowStats}.Serialize(startTime, endTime, true)
			expectedFlowLog = "1510948860 1510948920 wep kube-system iperf-4235-5623461 {\"test-app\":\"true\"} wep default nginx-412354-5123451 {\"k8s-app\":\"true\"} 10.0.0.1 20.0.0.1 6 54123 80 0 0 0 dst 0 0 0 0 allow"
			Expect(flowLog).Should(Equal(expectedFlowLog))

			By("aggregating on source port")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, SourcePort)
			Expect(err).To(BeNil())
			flowLog = FlowLog{flowMeta, flowStats}.Serialize(startTime, endTime, false)
			expectedFlowLog = "1510948860 1510948920 wep kube-system iperf-4235-5623461 - wep default nginx-412354-5123451 - 10.0.0.1 20.0.0.1 6 - 80 0 0 0 dst 0 0 0 0 allow"
			Expect(flowLog).Should(Equal(expectedFlowLog))

			flowLog = FlowLog{flowMeta, flowStats}.Serialize(startTime, endTime, true)
			expectedFlowLog = "1510948860 1510948920 wep kube-system iperf-4235-5623461 {\"test-app\":\"true\"} wep default nginx-412354-5123451 {\"k8s-app\":\"true\"} 10.0.0.1 20.0.0.1 6 - 80 0 0 0 dst 0 0 0 0 allow"
			Expect(flowLog).Should(Equal(expectedFlowLog))

			By("aggregating on prefix name")
			flowMeta, err = NewFlowMeta(muWithEndpointMeta, PrefixName)
			Expect(err).To(BeNil())
			flowLog = FlowLog{flowMeta, flowStats}.Serialize(startTime, endTime, false)
			expectedFlowLog = "1510948860 1510948920 wep kube-system iperf-4235-* - wep default nginx-412354-* - - - 6 - 80 0 0 0 dst 0 0 0 0 allow"
			Expect(flowLog).Should(Equal(expectedFlowLog))

			flowMeta, err = NewFlowMeta(muWithoutSrcEndpointMeta, PrefixName)
			Expect(err).To(BeNil())
			flowLog = FlowLog{flowMeta, flowStats}.Serialize(startTime, endTime, false)
			expectedFlowLog = "1510948860 1510948920 net - pvt - wep default nginx-412354-* - - - 6 - 80 0 0 0 dst 0 0 0 0 allow"
			Expect(flowLog).Should(Equal(expectedFlowLog))

			muWithoutPublicDstEndpointMeta := muWithoutDstEndpointMeta
			muWithoutPublicDstEndpointMeta.tuple.dst = ipStrTo16Byte("198.17.8.43")
			flowMeta, err = NewFlowMeta(muWithoutPublicDstEndpointMeta, PrefixName)
			Expect(err).To(BeNil())
			flowLog = FlowLog{flowMeta, flowStats}.Serialize(startTime, endTime, false)
			expectedFlowLog = "1510948860 1510948920 wep kube-system iperf-4235-* - net - pub - - - 6 - 80 0 0 0 dst 0 0 0 0 allow"
			Expect(flowLog).Should(Equal(expectedFlowLog))

			muWithoutAWSMetaDstEndpointMeta := muWithoutDstEndpointMeta
			muWithoutAWSMetaDstEndpointMeta.tuple.dst = ipStrTo16Byte("169.254.169.254")
			flowMeta, err = NewFlowMeta(muWithoutAWSMetaDstEndpointMeta, PrefixName)
			Expect(err).To(BeNil())
			flowLog = FlowLog{flowMeta, flowStats}.Serialize(startTime, endTime, false)
			expectedFlowLog = "1510948860 1510948920 wep kube-system iperf-4235-* - net - aws - - - 6 - 80 0 0 0 dst 0 0 0 0 allow"
			Expect(flowLog).Should(Equal(expectedFlowLog))

		})
	})

})
