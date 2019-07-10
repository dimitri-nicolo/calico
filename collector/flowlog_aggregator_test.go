// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
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
)

var _ = Describe("Flow log aggregator tests", func() {
	// TODO(SS): Pull out the convenience functions for re-use.
	expectFlowLog := func(fl FlowLog, t Tuple, nf, nfs, nfc int, a FlowLogAction, fr FlowLogReporter, pi, po, bi, bo int, sm, dm EndpointMetadata, sl, dl map[string]string, fp FlowPolicies) {
		expectedFlow := newExpectedFlowLog(t, nf, nfs, nfc, a, fr, pi, po, bi, bo, sm, dm, sl, dl, fp)

		// We don't include the start and end time in the comparison, so copy to a new log without these
		var flNoTime FlowLog
		flNoTime.FlowMeta = fl.FlowMeta
		flNoTime.FlowLabels = fl.FlowLabels
		flNoTime.FlowPolicies = fl.FlowPolicies
		flNoTime.FlowReportedStats = fl.FlowReportedStats
		Expect(flNoTime).Should(Equal(expectedFlow))
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
	calculateHTTPRequestStats := func(mus ...MetricUpdate) (allowed, denied int) {
		for _, mu := range mus {
			allowed += mu.inMetric.deltaAllowedHTTPRequests
			denied += mu.inMetric.deltaDeniedHTTPRequests
		}
		return
	}
	extractFlowPolicies := func(mus ...MetricUpdate) FlowPolicies {
		fp := make(FlowPolicies)
		for _, mu := range mus {
			for idx, r := range mu.ruleIDs {
				name := fmt.Sprintf("%d|%s|%s.%s|%s", idx,
					r.TierString(),
					r.TierString(),
					r.NameString(),
					r.ActionString())
				fp[name] = emptyValue
			}
		}
		return fp
	}
	Context("Flow log aggregator aggregation verification", func() {
		It("aggregates the fed metric updates", func() {
			By("default duration")
			ca := NewFlowLogAggregator()
			ca.IncludePolicies(true)
			ca.FeedUpdate(muNoConn1Rule1AllowUpdate)
			messages := ca.Get()
			Expect(len(messages)).Should(Equal(1))
			message := *(messages[0])

			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0

			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muNoConn1Rule1AllowUpdate)
			expectedFP := extractFlowPolicies(muNoConn1Rule1AllowUpdate)
			expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta, nil, nil, expectedFP)

			By("source port")
			ca = NewFlowLogAggregator().AggregateOver(FlowSourcePort)
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
			ca = NewFlowLogAggregator().AggregateOver(FlowPrefixName)
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
				Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235-", Labels: map[string]string{"test-app": "true"}},
			}

			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.dstEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-02",
					OrchestratorID: "k8s",
					WorkloadID:     "default/nginx-412354-6543645",
					EndpointID:     "256267",
				},
				Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354-", Labels: map[string]string{"k8s-app": "true"}},
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
				Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235-", Labels: map[string]string{"prod-app": "true"}},
			}

			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.dstEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-02",
					OrchestratorID: "k8s",
					WorkloadID:     "default/nginx-412354-6543645",
					EndpointID:     "256267",
				},
				// different label on the destination workload than one being tracked.
				Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354-", Labels: map[string]string{"k8s-app": "false"}},
			}

			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			messages = ca.Get()
			// Two updates should still result in 1 flow
			Expect(len(messages)).Should(Equal(1))

			By("by endpoint IP classification as the meta name when meta info is missing")
			ca = NewFlowLogAggregator().AggregateOver(FlowPrefixName)
			endpointMeta := calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-01",
					OrchestratorID: "k8s",
					WorkloadID:     "kube-system/iperf-4235-5623461",
					EndpointID:     "4352",
				},
				Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235-", Labels: map[string]string{"test-app": "true"}},
			}

			muWithoutDstEndpointMeta := MetricUpdate{
				updateType: UpdateTypeReport,
				tuple:      *NewTuple(ipStrTo16Byte("192.168.0.4"), ipStrTo16Byte("192.168.0.14"), proto_tcp, srcPort1, dstPort),

				// src endpoint meta info available
				srcEp: &endpointMeta,

				// dst endpoint meta info not available
				dstEp: nil,

				ruleIDs:      []*calc.RuleID{ingressRule1Allow},
				isConnection: false,
				inMetric: MetricValue{
					deltaPackets: 1,
					deltaBytes:   20,
				},
			}
			ca.FeedUpdate(muWithoutDstEndpointMeta)

			// Another metric update comes in. This time on a different dst private IP
			muWithoutDstEndpointMetaCopy := muWithoutDstEndpointMeta
			muWithoutDstEndpointMetaCopy.tuple.dst = ipStrTo16Byte("192.168.0.17")
			ca.FeedUpdate(muWithoutDstEndpointMetaCopy)
			messages = ca.Get()
			// One flow expected: srcMeta.GenerateName -> pvt
			// Two updates should still result in 1 flow
			Expect(len(messages)).Should(Equal(1))

			// Initial Update
			ca.FeedUpdate(muWithoutDstEndpointMeta)
			// + metric update comes in. This time on a non-private dst IP
			muWithoutDstEndpointMetaCopy.tuple.dst = ipStrTo16Byte("198.17.8.43")
			ca.FeedUpdate(muWithoutDstEndpointMetaCopy)
			messages = ca.Get()
			// 2nd flow expected: srcMeta.GenerateName -> pub
			// Three updates so far should result in 2 flows
			Expect(len(messages)).Should(Equal(2)) // Metric Update comes in with a non private as the dst IP

			// Initial Updates
			ca.FeedUpdate(muWithoutDstEndpointMeta)
			ca.FeedUpdate(muWithoutDstEndpointMetaCopy)
			// + metric update comes in. This time with missing src endpointMeta
			muWithoutDstEndpointMetaCopy.srcEp = nil
			muWithoutDstEndpointMetaCopy.dstEp = &endpointMeta
			ca.FeedUpdate(muWithoutDstEndpointMetaCopy)
			messages = ca.Get()

			// 3rd flow expected: pvt -> dst.GenerateName
			// Four updates so far should result in 3 flows
			Expect(len(messages)).Should(Equal(3)) // Metric Update comes in with a non private as the dst IP

			// Confirm the expected flow metas
			fm1 := FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					proto: 6,
					l4Src: unsetIntField,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "-",
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

			fm2 := FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					proto: 6,
					l4Src: unsetIntField,
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
					Namespace:      "kube-system",
					Name:           "-",
					AggregatedName: "iperf-4235-*",
				},
				Action:   "allow",
				Reporter: "dst",
			}

			fm3 := FlowMeta{
				Tuple: Tuple{
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					proto: 6,
					l4Src: unsetIntField,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "kube-system",
					Name:           "-",
					AggregatedName: "iperf-4235-*",
				},
				DstMeta: EndpointMetadata{
					Type:           "net",
					Namespace:      "-",
					Name:           "-",
					AggregatedName: "pvt",
				},
				Action:   "allow",
				Reporter: "dst",
			}

			flowLogMetas := []FlowMeta{}
			for _, fl := range messages {
				flowLogMetas = append(flowLogMetas, fl.FlowMeta)
			}

			Expect(flowLogMetas).Should(ConsistOf(fm1, fm2, fm3))
		})

		It("aggregates labels from metric updates", func() {
			By("intersecting labels in FlowSpec when IncludeLabels configured")
			ca := NewFlowLogAggregator().IncludeLabels(true)
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMeta)

			// Construct a similar update; but the endpoints have different labels
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy := muNoConn1Rule1AllowUpdateWithEndpointMeta
			// Updating the Workload IDs for src and dst.
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.srcEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-01",
					OrchestratorID: "k8s",
					WorkloadID:     "kube-system/iperf-4235-5623461",
					EndpointID:     "4352",
				},
				Endpoint: &model.WorkloadEndpoint{
					GenerateName: "iperf-4235-",
					Labels:       map[string]string{"test-app": "true", "new-label": "true"}, // "new-label" appended
				},
			}

			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.dstEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-02",
					OrchestratorID: "k8s",
					WorkloadID:     "default/nginx-412354-5123451",
					EndpointID:     "4352",
				},
				Endpoint: &model.WorkloadEndpoint{
					GenerateName: "nginx-412354-",
					Labels:       map[string]string{"k8s-app": "false"}, // conflicting labels; originally "k8s-app": "true"
				},
			}
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			messages := ca.Get()
			// Since the FlowMeta remains the same it should still equal 1.
			Expect(len(messages)).Should(Equal(1))
			message := *(messages[0])

			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			srcMeta := EndpointMetadata{
				Type:           "wep",
				Namespace:      "kube-system",
				Name:           "iperf-4235-5623461",
				AggregatedName: "iperf-4235-*",
			}
			dstMeta := EndpointMetadata{
				Type:           "wep",
				Namespace:      "default",
				Name:           "nginx-412354-5123451",
				AggregatedName: "nginx-412354-*",
			}
			// The labels should have been intersected correctly.
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn*2, expectedPacketsOut, expectedBytesIn*2, expectedBytesOut, srcMeta, dstMeta, map[string]string{"test-app": "true"}, map[string]string{}, nil)

			By("not affecting flow logs when IncludeLabels is disabled")
			ca = NewFlowLogAggregator().IncludeLabels(false)
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMeta)

			// Construct a similar update; but the endpoints have different labels
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy = muNoConn1Rule1AllowUpdateWithEndpointMeta
			// Updating the Workload IDs for src and dst.
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.srcEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-01",
					OrchestratorID: "k8s",
					WorkloadID:     "kube-system/iperf-4235-5623461",
					EndpointID:     "4352",
				},
				Endpoint: &model.WorkloadEndpoint{
					GenerateName: "iperf-4235-",
					Labels:       map[string]string{"test-app": "true", "new-label": "true"}, // "new-label" appended
				},
			}

			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.dstEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-02",
					OrchestratorID: "k8s",
					WorkloadID:     "default/nginx-412354-5123451",
					EndpointID:     "4352",
				},
				Endpoint: &model.WorkloadEndpoint{
					GenerateName: "nginx-412354-",
					Labels:       map[string]string{"k8s-app": "false"}, // conflicting labels; originally "k8s-app": "true"
				},
			}
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			messages = ca.Get()
			// Since the FlowMeta remains the same it should still equal 1.
			Expect(len(messages)).Should(Equal(1))
			message = *(messages[0])

			expectedNumFlows = 1
			expectedNumFlowsStarted = 1
			expectedNumFlowsCompleted = 0
			srcMeta = EndpointMetadata{
				Type:           "wep",
				Namespace:      "kube-system",
				Name:           "iperf-4235-5623461",
				AggregatedName: "iperf-4235-*",
			}
			dstMeta = EndpointMetadata{
				Type:           "wep",
				Namespace:      "default",
				Name:           "nginx-412354-5123451",
				AggregatedName: "nginx-412354-*",
			}
			// The labels should have been intersected right.
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn*2, expectedPacketsOut, expectedBytesIn*2, expectedBytesOut, srcMeta, dstMeta, nil, nil, nil) // nil & nil for Src and Dst Labels respectively.
		})
	})

	Context("Flow log aggregator filter verification", func() {
		It("Filters out MetricUpdate based on filter applied", func() {
			By("Creating 2 aggregators - one for denied packets, and one for allowed packets")
			var caa, cad FlowLogAggregator

			By("Checking that the MetricUpdate with deny action is only processed by the aggregator with the deny filter")
			caa = NewFlowLogAggregator().ForAction(rules.RuleActionAllow)
			cad = NewFlowLogAggregator().ForAction(rules.RuleActionDeny)

			caa.FeedUpdate(muNoConn1Rule2DenyUpdate)
			messages := caa.Get()
			Expect(len(messages)).Should(Equal(0))
			cad.FeedUpdate(muNoConn1Rule2DenyUpdate)
			messages = cad.Get()
			Expect(len(messages)).Should(Equal(1))

			By("Checking that the MetricUpdate with allow action is only processed by the aggregator with the allow filter")
			caa = NewFlowLogAggregator().ForAction(rules.RuleActionAllow)
			cad = NewFlowLogAggregator().ForAction(rules.RuleActionDeny)

			caa.FeedUpdate(muConn1Rule1AllowUpdate)
			messages = caa.Get()
			Expect(len(messages)).Should(Equal(1))
			cad.FeedUpdate(muConn1Rule1AllowUpdate)
			messages = cad.Get()
			Expect(len(messages)).Should(Equal(0))
		})

	})

	Context("Flow log aggregator http request countes", func() {
		It("Aggregates HTTP allowed and denied packets", func() {
			By("Feeding in two updates containing HTTP request counts")
			ca := NewFlowLogAggregator().ForAction(rules.RuleActionAllow).(*flowLogAggregator)
			ca.FeedUpdate(muConn1Rule1HTTPReqAllowUpdate)
			ca.FeedUpdate(muConn1Rule1HTTPReqAllowUpdate)
			messages := ca.Get()
			Expect(len(messages)).Should(Equal(1))
			// StartedFlowRefs count should be 1
			flowLog := messages[0]
			Expect(flowLog.NumFlowsStarted).Should(Equal(1))

			hra, hrd := calculateHTTPRequestStats(muConn1Rule1HTTPReqAllowUpdate, muConn1Rule1HTTPReqAllowUpdate)
			Expect(flowLog.HTTPRequestsAllowedIn).To(Equal(hra))
			Expect(flowLog.HTTPRequestsDeniedIn).To(Equal(hrd))
		})
	})

	Context("Flow log aggregator flowstore lifecycle", func() {
		It("Purges only the completed non-aggregated flowMetas", func() {
			By("Accounting for only the completed 5-tuple refs when making a purging decision")
			ca := NewFlowLogAggregator().ForAction(rules.RuleActionDeny).(*flowLogAggregator)
			ca.FeedUpdate(muNoConn1Rule2DenyUpdate)
			messages := ca.Get()
			Expect(len(messages)).Should(Equal(1))
			// StartedFlowRefs count should be 1
			flowLog := messages[0]
			Expect(flowLog.NumFlowsStarted).Should(Equal(1))

			// flowStore is not purged of the entry since the flowRef hasn't been expired
			Expect(len(ca.flowStore)).Should(Equal(1))

			// Feeding an update again. But StartedFlowRefs count should be 0
			ca.FeedUpdate(muNoConn1Rule2DenyUpdate)
			messages = ca.Get()
			Expect(len(messages)).Should(Equal(1))
			flowLog = messages[0]
			Expect(flowLog.NumFlowsStarted).Should(Equal(0))

			// Feeding an expiration of the conn.
			ca.FeedUpdate(muNoConn1Rule2DenyExpire)
			messages = ca.Get()
			Expect(len(messages)).Should(Equal(1))
			flowLog = messages[0]
			Expect(flowLog.NumFlowsCompleted).Should(Equal(1))
			Expect(flowLog.NumFlowsStarted).Should(Equal(0))
			Expect(flowLog.NumFlows).Should(Equal(1))

			// flowStore is now purged of the entry since the flowRef has been expired
			Expect(len(ca.flowStore)).Should(Equal(0))
		})

		It("Purges only the completed aggregated flowMetas", func() {
			By("Accounting for only the completed 5-tuple refs when making a purging decision")
			ca := NewFlowLogAggregator().AggregateOver(FlowPrefixName).(*flowLogAggregator)
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMeta)
			// Construct a similar update; same tuple but diff src ports.
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy := muNoConn1Rule1AllowUpdateWithEndpointMeta
			tuple1Copy := tuple1
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
				Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235-", Labels: map[string]string{"test-app": "true"}},
			}

			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.dstEp = &calc.EndpointData{
				Key: model.WorkloadEndpointKey{
					Hostname:       "node-02",
					OrchestratorID: "k8s",
					WorkloadID:     "default/nginx-412354-6543645",
					EndpointID:     "256267",
				},
				Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354-", Labels: map[string]string{"k8s-app": "true"}},
			}

			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			messages := ca.Get()
			// Two updates should still result in 1 flowMeta
			Expect(len(messages)).Should(Equal(1))
			// flowStore is not purged of the entry since the flowRefs havn't been expired
			Expect(len(ca.flowStore)).Should(Equal(1))
			// And the no. of Started Flows should be 2
			flowLog := messages[0]
			Expect(flowLog.NumFlowsStarted).Should(Equal(2))

			// Update one of the two flows and expire the other.
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMeta)
			muNoConn1Rule1AllowUpdateWithEndpointMetaCopy.updateType = UpdateTypeExpire
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMetaCopy)
			messages = ca.Get()
			Expect(len(messages)).Should(Equal(1))
			// flowStore still carries that 1 flowMeta
			Expect(len(ca.flowStore)).Should(Equal(1))
			flowLog = messages[0]
			Expect(flowLog.NumFlowsStarted).Should(Equal(0))
			Expect(flowLog.NumFlowsCompleted).Should(Equal(1))
			Expect(flowLog.NumFlows).Should(Equal(2))

			// Expire the sole flowRef
			muNoConn1Rule1AllowUpdateWithEndpointMeta.updateType = UpdateTypeExpire
			ca.FeedUpdate(muNoConn1Rule1AllowUpdateWithEndpointMeta)
			// Pre-purge/Dispatch the meta still lingers
			Expect(len(ca.flowStore)).Should(Equal(1))
			// On a dispatch the flowMeta is eventually purged
			messages = ca.Get()
			Expect(len(ca.flowStore)).Should(Equal(0))
			flowLog = messages[0]
			Expect(flowLog.NumFlowsStarted).Should(Equal(0))
			Expect(flowLog.NumFlowsCompleted).Should(Equal(1))
			Expect(flowLog.NumFlows).Should(Equal(1))
		})

		It("Updates the stats associated with the flows", func() {
			By("Accounting for only the packet/byte counts as seen during the interval")
			ca := NewFlowLogAggregator().ForAction(rules.RuleActionAllow).(*flowLogAggregator)
			ca.FeedUpdate(muConn1Rule1AllowUpdate)
			messages := ca.Get()
			Expect(len(messages)).Should(Equal(1))
			// After the initial update the counts as expected.
			flowLog := messages[0]
			Expect(flowLog.PacketsIn).Should(Equal(2))
			Expect(flowLog.BytesIn).Should(Equal(22))
			Expect(flowLog.PacketsOut).Should(Equal(3))
			Expect(flowLog.BytesOut).Should(Equal(33))

			// The flow doesn't expire. But the Get should reset the stats.
			// A new update on top, then, should result in the same counts
			ca.FeedUpdate(muConn1Rule1AllowUpdate)
			messages = ca.Get()
			Expect(len(messages)).Should(Equal(1))
			// After the initial update the counts as expected.
			flowLog = messages[0]
			Expect(flowLog.PacketsIn).Should(Equal(2))
			Expect(flowLog.BytesIn).Should(Equal(22))
			Expect(flowLog.PacketsOut).Should(Equal(3))
			Expect(flowLog.BytesOut).Should(Equal(33))
		})
	})
})
