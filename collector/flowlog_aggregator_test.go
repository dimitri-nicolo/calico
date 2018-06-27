// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var (
	tuple4 = *NewTuple(localIp2, localIp1DNAT, proto_tcp, srcPort1, dstPort)
)

// Common MetricUpdate definitions
var (
	// Metric update without a connection (ingress stats match those of muConn1Rule1AllowUpdate).
	muNoConn1Rule1AllowUpdateWithEndpointMeta = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple1,

		srcEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-01",
				OrchestratorID: "k8s",
				WorkloadID:     "kube-system/iperf-4235-5623461",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235", Labels: map[string]string{"test-app": "true"}},
		},

		dstEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-02",
				OrchestratorID: "k8s",
				WorkloadID:     "default/nginx-412354-5123451",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354", Labels: map[string]string{"k8s-app": "true"}},
		},

		ruleID:       ingressRule1Allow,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	muNoConn1Rule1AllowUpdateWithEndpointIPClassified = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple4,

		srcEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-01",
				OrchestratorID: "k8s",
				WorkloadID:     "kube-system/iperf-4235-5623461",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235", Labels: map[string]string{"test-app": "true"}},
		},

		ruleID:       ingressRule1Allow,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}
)
var _ = Describe("Flow log aggregator tests", func() {
	// TODO(SS): Pull out the convenience functions for re-use.
	expectFlowLog := func(msg string, t Tuple, nf, nfs, nfc int, a FlowLogAction, fd FlowLogDirection, pi, po, bi, bo int) {
		fl, err := getFlowLog(msg)
		Expect(err).To(BeNil())
		expectedFlow := newExpectedFlowLog(t, nf, nfs, nfc, a, fd, pi, po, bi, bo)
		Expect(fl).Should(Equal(expectedFlow))
	}
	calculatePacketStats := func(mus ...MetricUpdate) (epi, epo, ebi, ebo int) {
		for _, mu := range mus {
			epi += mu.inMetric.deltaPackets
			epo += mu.outMetric.deltaPackets
			ebi += mu.inMetric.deltaBytes
			ebo += mu.outMetric.deltaBytes
		}
		return
	}
	Context("Flow log aggregator aggregation verification", func() {
		It("aggregates the fed metric updates", func() {
			By("defalt duration")
			ca := NewCloudWatchAggregator()
			ca.FeedUpdate(muNoConn1Rule1AllowUpdate)
			messages := ca.Get()
			Expect(len(messages)).Should(Equal(1))
			message := *(messages[0])

			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0

			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muNoConn1Rule1AllowUpdate)
			expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut)

			By("source port")
			ca = NewCloudWatchAggregator().AggregateOver(SourcePort)
			ca.FeedUpdate(muNoConn1Rule1AllowUpdate)
			// Construct a similar update; same tuple but diff src ports.
			muNoConn1Rule1AllowUpdateCopy := muNoConn1Rule1AllowUpdate
			tuple1Copy := tuple1
			tuple1Copy.l4Src = 44123
			muNoConn1Rule1AllowUpdateCopy.tuple = tuple1Copy
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateCopy)
			messages = ca.Get()
			// Two updates should still result in 1 flow
			Expect(len(messages)).Should(Equal(1))

			By("endpoint prefix names")
			ca = NewCloudWatchAggregator().AggregateOver(PrefixName)
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMeta)
			// Construct a similar update; same tuple but diff src ports.
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy := muNoConn1Rule1AllowUpdateWithEndpointMeta
			// TODO(SS): Handle and organize these test constants better. Right now they are all over the places
			// like reporter_prometheus_test.go, collector_test.go , etc.
			tuple1Copy = tuple1
			// Everything can change in the 5-tuple except for the dst port.
			tuple1Copy.l4Src = 44123
			tuple1Copy.src = ipStrTo16Byte("10.0.0.3")
			tuple1Copy.dst = ipStrTo16Byte("10.0.0.9")
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.tuple = tuple1Copy

			// Updating the Workload IDs for src and dst.
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.srcEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-01",
					OrchestratorID: "k8s",
					WorkloadID:     "kube-system/iperf-4235-5434134",
					EndpointID:     "23456",
				},
				Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235", Labels: map[string]string{"test-app": "true"}},
			}

			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.dstEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-02",
					OrchestratorID: "k8s",
					WorkloadID:     "default/nginx-412354-6543645",
					EndpointID:     "256267",
				},
				Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354", Labels: map[string]string{"k8s-app": "true"}},
			}

			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			messages = ca.Get()
			// Two updates should still result in 1 flow
			Expect(len(messages)).Should(Equal(1))

			// Updating the Workload IDs and labels for src and dst.
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.srcEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-01",
					OrchestratorID: "k8s",
					WorkloadID:     "kube-system/iperf-4235-5434134",
					EndpointID:     "23456",
				},
				// this new MetricUpdates src endpointMeta has a different label than one currently being tracked.
				Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235", Labels: map[string]string{"prod-app": "true"}},
			}

			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.dstEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-02",
					OrchestratorID: "k8s",
					WorkloadID:     "default/nginx-412354-6543645",
					EndpointID:     "256267",
				},
				// different label on the destination workload than one being tracked.
				Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354", Labels: map[string]string{"k8s-app": "false"}},
			}

			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			messages = ca.Get()
			// Two updates should still result in 1 flow
			Expect(len(messages)).Should(Equal(1))

			By("by endpoint IP type")
			ca = NewCloudWatchAggregator().AggregateOver(PrefixName)
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointIPClassified)

			muNoConn1Rule1AllowUpdateWithEndpointIPClassifiedCopy := muNoConn1Rule1AllowUpdateWithEndpointIPClassified
			muNoConn1Rule1AllowUpdateWithEndpointIPClassifiedCopy.tuple.dst = localIp2DNAT
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointIPClassified)
			messages = ca.Get()
			// Two updates should still result in 1 flow
			Expect(len(messages)).Should(Equal(1))
		})
	})
	Context("Flow log aggregator filter verification", func() {
		It("Filters out MetricUpdate based on filter applied", func() {
			By("Creating 2 aggregators - one for denied packets, and one for allowed packets")
			caa := NewCloudWatchAggregator().ForAction(rules.RuleActionAllow)
			cad := NewCloudWatchAggregator().ForAction(rules.RuleActionDeny)

			By("Checking that the MetricUpdate with deny action is only processed by the aggregator with the deny filter")
			caa.FeedUpdate(muNoConn1Rule2DenyUpdate)
			messages := caa.Get()
			Expect(len(messages)).Should(Equal(0))
			cad.FeedUpdate(muNoConn1Rule2DenyUpdate)
			messages = cad.Get()
			Expect(len(messages)).Should(Equal(1))

			By("Checking that the MetricUpdate with allow action is only processed by the aggregator with the allow filter")
			caa.FeedUpdate(muConn1Rule1AllowUpdate)
			messages = caa.Get()
			Expect(len(messages)).Should(Equal(1))
			cad.FeedUpdate(muConn1Rule1AllowUpdate)
			messages = cad.Get()
			Expect(len(messages)).Should(Equal(0))
		})

	})
})
