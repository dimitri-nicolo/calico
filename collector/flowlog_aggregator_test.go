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
	muNoConn1Rule1AllowUpdateWithEndpointMeta = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple1,

		srcEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-01",
				OrchestratorID: "k8s",
				WorkloadID:     "iperf-4235-5623461/kube-system",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235", Labels: map[string]string{"test-app": "true"}},
		},

		dstEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-02",
				OrchestratorID: "k8s",
				WorkloadID:     "nginx-412354-5123451/default",
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
)
var _ = Describe("Flow log aggregator verification", func() {
	It("aggregates the fed metric updates", func() {
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

		By("prefix name")
		ca = NewCloudWatchAggregator().AggregateOver(PrefixName)
		ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMeta)
		// Construct a similar update; same tuple but diff src ports.
		muNoConn1Rule1AllowUpdateWithEndpointMetaCopy := muNoConn1Rule1AllowUpdateWithEndpointMeta
		// TODO: Handle and organize these test constants better. Right now they are all over the places
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
				WorkloadID:     "iperf-4235-5434134/kube-system",
				EndpointID:     "23456",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235", Labels: map[string]string{"test-app": "true"}},
		}

		muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.dstEp = &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-02",
				OrchestratorID: "k8s",
				WorkloadID:     "nginx-412354-6543645/default",
				EndpointID:     "256267",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354", Labels: map[string]string{"k8s-app": "true"}},
		}

		ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
		messages = ca.Get()
		// Two updates should still result in 1 flow
		Expect(len(messages)).Should(Equal(1))
	})
})
