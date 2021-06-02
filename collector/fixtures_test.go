// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package collector

import (
	"net"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/proxy"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

const testMaxBoundedSetSize = 5

var (
	sendCongestionWnd = 10
	smoothRtt         = 1
	minRtt            = 2
	mss               = 4
	flowMetaDefault   = FlowMeta{
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaDefaultWithService = FlowMeta{
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
		DstService: FlowService{
			Namespace: "foo-ns",
			Name:      "foo-svc",
			PortName:  "foo-port",
			PortNum:   8080,
		},
		Action:   "allow",
		Reporter: "dst",
	}

	flowMetaDefaultNoSourceMeta = FlowMeta{
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaDefaultNoDestMeta = FlowMeta{
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaSourcePorts = FlowMeta{
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaPrefix = FlowMeta{
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaPrefixNoSourceMeta = FlowMeta{
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaPrefixNoDestMeta = FlowMeta{
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaPrefixWithName = FlowMeta{
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaNoDestPorts = FlowMeta{
		Tuple: Tuple{
			src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			proto: 6,
			l4Src: -1, // Is the only attribute that gets disregarded.
			l4Dst: -1,
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaNoDestPortsWithService = FlowMeta{
		Tuple: Tuple{
			src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			proto: 6,
			l4Src: -1, // Is the only attribute that gets disregarded.
			l4Dst: -1,
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
		DstService: FlowService{
			Namespace: "foo-ns",
			Name:      "foo-svc",
			PortName:  "-",
			PortNum:   8080,
		},
		Action:   "allow",
		Reporter: "dst",
	}

	flowMetaNoDestPortNoDestMeta = FlowMeta{
		Tuple: Tuple{
			src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			proto: 6,
			l4Src: -1,
			l4Dst: -1,
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
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}

	flowMetaNoDestPortNoSourceMeta = FlowMeta{
		Tuple: Tuple{
			src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			proto: 6,
			l4Src: -1,
			l4Dst: -1,
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
			Name:           "-",
			AggregatedName: "nginx-412354-*",
		},
		DstService: noService,
		Action:     "allow",
		Reporter:   "dst",
	}
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
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithEndpointMetaExpire = MetricUpdate{
		updateType: UpdateTypeExpire,
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
	}

	muWithEndpointMetaWithService = MetricUpdate{
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

		dstService: MetricServiceInfo{
			proxy.ServicePortName{
				NamespacedName: types.NamespacedName{
					Namespace: "foo-ns",
					Name:      "foo-svc",
				},
				Port: "foo-port",
			},
			8080,
		},
		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithEndpointMetaAndDifferentLabels = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple1,

		srcEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-01",
				OrchestratorID: "k8s",
				WorkloadID:     "kube-system/iperf-4235-5623461",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "iperf-4235-", Labels: map[string]string{"test-app": "true", "new-label": "true"}},
		},

		dstEp: &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "node-02",
				OrchestratorID: "k8s",
				WorkloadID:     "default/nginx-412354-5123451",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "nginx-412354-", Labels: map[string]string{"k8s-app": "false"}},
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

	muWithOrigSourceIPs = MetricUpdate{
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

		origSourceIPs: NewBoundedSetFromSlice(testMaxBoundedSetSize, []net.IP{net.ParseIP(publicIP1Str)}),

		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	muWithOrigSourceIPsExpire = MetricUpdate{
		updateType: UpdateTypeExpire,
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

		origSourceIPs: NewBoundedSetFromSlice(testMaxBoundedSetSize, []net.IP{net.ParseIP(publicIP1Str)}),

		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 0,
			deltaBytes:   0,
		},
	}

	muWithOrigSourceIPsUnknownRuleID = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple1,

		srcEp: nil,
		dstEp: nil,

		origSourceIPs: NewBoundedSetFromSlice(testMaxBoundedSetSize, []net.IP{net.ParseIP(publicIP1Str)}),

		unknownRuleID: &calc.RuleID{
			PolicyID: calc.PolicyID{
				Tier:      "__UNKNOWN__",
				Name:      "__UNKNOWN__",
				Namespace: "__UNKNOWN__",
			},
			Index:     -2,
			IndexStr:  "-2",
			Action:    rules.RuleActionAllow,
			Direction: rules.RuleDirIngress,
		},
		isConnection: false,
	}

	muWithMultipleOrigSourceIPs = MetricUpdate{
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

		origSourceIPs: NewBoundedSetFromSlice(testMaxBoundedSetSize, []net.IP{net.ParseIP(publicIP1Str), net.ParseIP(publicIP2Str)}),

		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	muWithEndpointMetaWithoutGenerateName = MetricUpdate{
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
				WorkloadID:     "default/manually-created-pod",
				EndpointID:     "4352",
			},
			Endpoint: &model.WorkloadEndpoint{GenerateName: "", Labels: map[string]string{"k8s-app": "true"}},
		},

		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	muWithProcessName = MetricUpdate{
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

		processName:       "test-process",
		processID:         1234,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithProcessNameDifferentIDSameTuple = MetricUpdate{
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

		processName:       "test-process",
		processID:         4321,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithProcessNameExpire = MetricUpdate{
		updateType: UpdateTypeExpire,
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

		processName: "test-process",
		processID:   1234,
	}

	muWithSameProcessNameDifferentID = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple2,

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

		processName:       "test-process",
		processID:         4321,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithSameProcessNameDifferentIDExpire = MetricUpdate{
		updateType: UpdateTypeExpire,
		tuple:      tuple2,

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

		processName: "test-process",
		processID:   4321,
	}

	muWithDifferentProcessNameDifferentID = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple3,

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

		processName:       "test-process-2",
		processID:         23456,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithDifferentProcessNameDifferentIDExpire = MetricUpdate{
		updateType: UpdateTypeExpire,
		tuple:      tuple3,

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

		processName:       "test-process-2",
		processID:         23456,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithProcessName2 = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple3,

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

		processName:       "test-process-2",
		processID:         9876,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithProcessName3 = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple4,

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

		processName:       "test-process-3",
		processID:         5678,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithProcessName4 = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple5,

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

		processName:       "test-process-4",
		processID:         34567,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}

	muWithProcessName5 = MetricUpdate{
		updateType: UpdateTypeReport,
		tuple:      tuple6,

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

		processName:       "test-process-5",
		processID:         7654,
		sendCongestionWnd: &sendCongestionWnd,
		smoothRtt:         &smoothRtt,
		minRtt:            &minRtt,
		mss:               &mss,
		tcpMetric: TCPMetricValue{
			deltaTotalRetrans:   7,
			deltaLostOut:        6,
			deltaUnRecoveredRTO: 8,
		},
	}
)
