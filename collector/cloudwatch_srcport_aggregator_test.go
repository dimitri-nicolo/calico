// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	sIp = [16]byte{10, 0, 0, 1}
	dIp = [16]byte{20, 0, 0, 1}
)

var (
	sPort1 = 54123
	sPort2 = 54124
	sPort3 = 54003
	dPort  = 80
)

// Common Tuple definitions
var (
	tup1 = *NewTuple(sIp, dIp, proto_tcp, sPort1, dPort)
	tup2 = *NewTuple(sIp, dIp, proto_tcp, sPort2, dPort)
	tup3 = *NewTuple(sIp, dIp, proto_tcp, sPort3, dPort)
)

// Common MetricUpdate definitions
var (
	// Metric update without a connection (ingress stats match those of muConn1Rule1AllowUpdate).
	mu1 = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tup1,
		ruleID:       ingressRule1Allow,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	mu2 = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tup2,
		ruleID:       ingressRule1Allow,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	mu3 = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tup3,
		ruleID:       ingressRule1Allow,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}
)

var _ = Describe("CloudWatch SrcPort Aggregator verification", func() {
	var (
		ca FlowLogAggregator
	)
	BeforeEach(func() {
		log.SetOutput(os.Stdout)
		ca = NewCloudWatchAggregator()
	})
	It("reports the given metric updates aggregated based on src port", func() {
		ca.FeedUpdate(mu1)
		ca.FeedUpdate(mu2)
		ca.FeedUpdate(mu3)

		aggregatedResps := ca.Get()
		Expect(len(aggregatedResps)).Should(Equal(1))
		message := *aggregatedResps[0]
		// Expect only one response based on the aggregation.
		Expect(message).Should(Equal("10.0.0.1 20.0.0.1 80 6"))
	})
})
