// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/felix/collector/testutil"
	"github.com/projectcalico/libcalico-go/lib/health"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	logGroupName  = "test-group"
	logStreamName = "test-stream"
	flushInterval = 500 * time.Millisecond
	includeLabels = false
	noService     = FlowService{Namespace: "-", Name: "-", Port: "-"}
)

var (
	pvtMeta = EndpointMetadata{Type: FlowLogEndpointTypeNet, Namespace: "-", Name: "-", AggregatedName: "pvt"}
	pubMeta = EndpointMetadata{Type: FlowLogEndpointTypeNet, Namespace: "-", Name: "-", AggregatedName: "pub"}
)

const CWDispatcher = "cloudwatch"

var _ = Describe("FlowLog Reporter verification", func() {
	var (
		cr *FlowLogsReporter
		cd LogDispatcher
		ca FlowLogAggregator
		cl cloudwatchlogsiface.CloudWatchLogsAPI
	)

	mt := &mockTime{}
	getEventsFromLogStream := func() []*cloudwatchlogs.OutputLogEvent {
		logEventsInput := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroupName,
			LogStreamName: &logStreamName,
		}
		logEventsOutput, _ := cl.GetLogEvents(logEventsInput)
		return logEventsOutput.Events
	}
	expectFlowLogsInEventStream := func(fls ...FlowLog) {
		events := getEventsFromLogStream()
		count := 0
		for _, fl := range fls {
			for _, ev := range events {
				flMsg, _ := getFlowLog(*ev.Message)
				if reflect.DeepEqual(flMsg, fl) {
					count++
					if count == len(fls) {
						break
					}
				}
			}
		}
		Expect(count).Should(Equal(len(fls)))
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

	extractFlowExtras := func(mus ...MetricUpdate) FlowExtras {
		var ipBs *boundedSet
		for _, mu := range mus {
			if mu.origSourceIPs == nil {
				continue
			}
			if ipBs == nil {
				ipBs = mu.origSourceIPs.Copy()
			} else {
				ipBs.Combine(mu.origSourceIPs)
			}
		}
		if ipBs != nil {
			return FlowExtras{
				OriginalSourceIPs:    ipBs.ToIPSlice(),
				NumOriginalSourceIPs: ipBs.TotalCount(),
			}
		} else {
			return FlowExtras{}
		}
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
	Context("No Aggregation kind specified", func() {
		BeforeEach(func() {
			cl = testutil.NewMockedCloudWatchLogsClient(logGroupName)
			cd = NewCloudWatchDispatcher(logGroupName, logStreamName, 7, cl)
			ca = NewFlowLogAggregator()
			ca.IncludePolicies(true)
			ds := map[string]LogDispatcher{CWDispatcher: cd}
			cr = NewFlowLogsReporter(ds, flushInterval, nil, false, &NoOpLogOffset{})
			cr.AddAggregator(ca, []string{CWDispatcher})
			cr.timeNowFn = mt.getMockTime
			cr.Start()

		})
		AfterEach(func() {
			Expect(cl.(testutil.CloudWatchLogsExpectation).RetentionPeriod()).To(BeNumerically("==", 7))
		})
		It("reports the given metric update in form of a flow to cloudwatchlogs", func() {
			By("reporting the first MetricUpdate")
			cr.Report(muNoConn1Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muNoConn1Rule1AllowUpdate)
			expectedFP := extractFlowPolicies(muNoConn1Rule1AllowUpdate)
			expectedFlowExtras := extractFlowExtras(muNoConn1Rule1AllowUpdate)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))

			By("reporting the same MetricUpdate with metrics in both directions")
			cr.Report(muConn1Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlowsStarted = 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muConn1Rule1AllowUpdate)
			expectedFP = extractFlowPolicies(muConn1Rule1AllowUpdate)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))

			By("reporting a expired MetricUpdate for the same tuple")
			cr.Report(muConn1Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muConn1Rule1AllowExpire)
			expectedFP = extractFlowPolicies(muConn1Rule1AllowExpire)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))

			By("reporting a MetricUpdate for denied packets")
			cr.Report(muNoConn3Rule2DenyUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlows = 1
			expectedNumFlowsStarted = 1
			expectedNumFlowsCompleted = 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muNoConn1Rule2DenyUpdate)
			expectedFP = extractFlowPolicies(muNoConn1Rule2DenyUpdate)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple3, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogReporterSrc,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))

			By("reporting a expired denied packet MetricUpdate for the same tuple")
			cr.Report(muNoConn3Rule2DenyExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muNoConn1Rule2DenyExpire)
			expectedFP = extractFlowPolicies(muNoConn1Rule2DenyExpire)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple3, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogReporterSrc,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))

		})
		It("aggregates metric updates for the duration of aggregation when reporting to cloudwatch logs", func() {

			By("reporting the same MetricUpdate twice and expiring it immediately")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn1Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muConn1Rule1AllowUpdate, muConn1Rule1AllowUpdate, muConn1Rule1AllowExpire)
			expectedFP := extractFlowPolicies(muConn1Rule1AllowUpdate, muConn1Rule1AllowUpdate, muConn1Rule1AllowExpire)
			expectedFlowExtras := extractFlowExtras(muConn1Rule1AllowUpdate, muConn1Rule1AllowUpdate, muConn1Rule1AllowExpire)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))

			By("reporting the same tuple different policies should be reported as separate flow logs")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muNoConn1Rule2DenyUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)

			expectedNumFlows = 1
			expectedNumFlowsStarted = 1
			expectedNumFlowsCompleted = 0
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 := calculatePacketStats(muConn1Rule1AllowUpdate)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 := calculatePacketStats(muNoConn1Rule2DenyUpdate)
			expectedFP1 := extractFlowPolicies(muConn1Rule1AllowUpdate)
			expectedFP2 := extractFlowPolicies(muNoConn1Rule2DenyUpdate)
			expectedFlowExtras1 := extractFlowExtras(muConn1Rule1AllowUpdate)
			expectedFlowExtras2 := extractFlowExtras(muNoConn1Rule2DenyUpdate)
			// We only care about the flow log entry to exist and don't care about the actual order.
			expectFlowLogsInEventStream(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1, pvtMeta, pubMeta, noService, nil, nil, expectedFP1, expectedFlowExtras1, noProcessInfo),
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogReporterSrc,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2, pvtMeta, pubMeta, noService, nil, nil, expectedFP2, expectedFlowExtras2, noProcessInfo))
		})

		It("aggregates metric updates from multiple tuples", func() {

			By("report different connections")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn2Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)

			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 := calculatePacketStats(muConn1Rule1AllowUpdate)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 := calculatePacketStats(muConn2Rule1AllowUpdate)
			expectedFP1 := extractFlowPolicies(muConn1Rule1AllowUpdate)
			expectedFP2 := extractFlowPolicies(muConn2Rule1AllowUpdate)
			expectedFlowExtras1 := extractFlowExtras(muConn1Rule1AllowUpdate)
			expectedFlowExtras2 := extractFlowExtras(muConn2Rule1AllowUpdate)
			// We only care about the flow log entry to exist and don't care about the actual order.
			expectFlowLogsInEventStream(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1, pvtMeta, pubMeta, noService, nil, nil, expectedFP1, expectedFlowExtras1, noProcessInfo),
				newExpectedFlowLog(tuple2, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2, pvtMeta, pubMeta, noService, nil, nil, expectedFP2, expectedFlowExtras2, noProcessInfo))

			By("report expirations of the same connections")
			cr.Report(muConn1Rule1AllowExpire)
			cr.Report(muConn2Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)

			expectedNumFlows = 1
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 = calculatePacketStats(muConn1Rule1AllowExpire)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 = calculatePacketStats(muConn2Rule1AllowExpire)
			expectedFP1 = extractFlowPolicies(muConn1Rule1AllowExpire)
			expectedFP2 = extractFlowPolicies(muConn2Rule1AllowExpire)
			expectedFlowExtras1 = extractFlowExtras(muConn1Rule1AllowExpire)
			expectedFlowExtras2 = extractFlowExtras(muConn2Rule1AllowExpire)
			// We only care about the flow log entry to exist and don't care about the actual order.
			expectFlowLogsInEventStream(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1, pvtMeta, pubMeta, noService, nil, nil, expectedFP1, expectedFlowExtras1, noProcessInfo),
				newExpectedFlowLog(tuple2, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2, pvtMeta, pubMeta, noService, nil, nil, expectedFP2, expectedFlowExtras2, noProcessInfo))

		})
		It("Doesn't process flows from Hostendoint to Hostendpoint", func() {
			By("Reporting a update with host endpoint to host endpoint")
			muConn1Rule1AllowUpdateCopy := muConn1Rule1AllowUpdate
			muConn1Rule1AllowUpdateCopy.srcEp = localHostEd1
			muConn1Rule1AllowUpdateCopy.dstEp = remoteHostEd1
			cr.Report(muConn1Rule1AllowUpdateCopy)
			time.Sleep(1 * time.Second)

			By("Verifying that flow logs are logged with pvt and pub metadata")
			time.Sleep(1 * time.Second)
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muConn1Rule1AllowUpdateCopy)
			expectedFP := extractFlowPolicies(muConn1Rule1AllowUpdateCopy)
			expectedFlowExtras := extractFlowExtras(muConn1Rule1AllowUpdateCopy)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))
		})
		It("reports the given metric update with original source IPs in a flow to cloudwatchlogs", func() {
			By("reporting the first MetricUpdate")
			cr.Report(muWithOrigSourceIPs)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muWithOrigSourceIPs)
			expectedFP := extractFlowPolicies(muWithOrigSourceIPs)
			expectedFlowExtras := extractFlowExtras(muWithOrigSourceIPs)
			meta := EndpointMetadata{Type: FlowLogEndpointTypeWep, Namespace: "default", Name: "nginx-412354-5123451", AggregatedName: "nginx-412354-*"}
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, meta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))
		})
	})
	Context("Enable Flowlogs for HEPs", func() {
		BeforeEach(func() {
			cl = testutil.NewMockedCloudWatchLogsClient(logGroupName)
			cd = NewCloudWatchDispatcher(logGroupName, logStreamName, 7, cl)
			ca = NewFlowLogAggregator()
			ca.IncludePolicies(true)
			ds := map[string]LogDispatcher{CWDispatcher: cd}
			cr = NewFlowLogsReporter(ds, flushInterval, nil, true, &NoOpLogOffset{})
			cr.AddAggregator(ca, []string{CWDispatcher})
			cr.timeNowFn = mt.getMockTime
			cr.Start()
		})
		It("processes flows from Hostendoint to Hostendpoint", func() {
			By("Reporting a update with host endpoint to host endpoint")
			muConn1Rule1AllowUpdateCopy := muConn1Rule1AllowUpdate
			muConn1Rule1AllowUpdateCopy.srcEp = localHostEd1
			muConn1Rule1AllowUpdateCopy.dstEp = remoteHostEd1
			cr.Report(muConn1Rule1AllowUpdateCopy)

			By("Verifying that flow logs are logged with HEP metadata")
			time.Sleep(1 * time.Second)
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muConn1Rule1AllowUpdateCopy)
			expectedSrcMeta := EndpointMetadata{Type: FlowLogEndpointTypeHep, Namespace: "-", Name: "eth1", AggregatedName: "localhost"}
			expectedDstMeta := EndpointMetadata{Type: FlowLogEndpointTypeHep, Namespace: "-", Name: "eth1", AggregatedName: "remotehost"}
			expectedFP := extractFlowPolicies(muConn1Rule1AllowUpdateCopy)
			expectedFlowExtras := extractFlowExtras(muConn1Rule1AllowUpdateCopy)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogReporterDst,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, expectedSrcMeta, expectedDstMeta, noService, nil, nil, expectedFP, expectedFlowExtras, noProcessInfo))
		})
	})
})

var _ = Describe("CloudWatch Reporter health verification", func() {
	var (
		cr *FlowLogsReporter
		cd LogDispatcher
		cl cloudwatchlogsiface.CloudWatchLogsAPI
		hr *health.HealthAggregator
	)

	mt := &mockTime{}
	Context("Test with no errors", func() {
		BeforeEach(func() {
			cl = testutil.NewMockedCloudWatchLogsClient(logGroupName)
			cd = NewCloudWatchDispatcher(logGroupName, logStreamName, 7, cl)
			hr = health.NewHealthAggregator()
			ds := map[string]LogDispatcher{CWDispatcher: cd}
			cr = NewFlowLogsReporter(ds, flushInterval, hr, false, &NoOpLogOffset{})
			cr.timeNowFn = mt.getMockTime
			cr.Start()
		})
		It("verify health reporting.", func() {
			By("checking the Readiness flag in health aggregator")
			expectedReport := health.HealthReport{Live: true, Ready: true}
			Eventually(func() health.HealthReport { return *hr.Summary() }, 15, 1).Should(Equal(expectedReport))
		})
	})
	Context("Test with client that times out requests", func() {
		BeforeEach(func() {
			cl = &timingoutCWFLMockClient{timeout: time.Second}
			cd = NewCloudWatchDispatcher(logGroupName, logStreamName, 7, cl)
			hr = health.NewHealthAggregator()
			ds := map[string]LogDispatcher{CWDispatcher: cd}
			cr = NewFlowLogsReporter(ds, flushInterval, hr, false, &NoOpLogOffset{})
			cr.timeNowFn = mt.getMockTime
			cr.Start()
		})
		It("verify health reporting.", func() {
			By("checking the Readiness flag in health aggregator")
			expectedReport := health.HealthReport{Live: true, Ready: false}
			Eventually(func() health.HealthReport { return *hr.Summary() }, 15, 1).Should(Equal(expectedReport))
		})
	})
})

var _ = Describe("FlowLog per minute verification", func() {
	var (
		cr *FlowLogsReporter
		cd LogDispatcher
		ca FlowLogAggregator
		cl cloudwatchlogsiface.CloudWatchLogsAPI
	)

	mt := &mockTime{}

	Context("Flow logs per minute verification", func() {
		It("Usage report is triggered before flushIntervalDuration", func() {
			By("Triggering report right away before flushIntervalDuration")
			cd = NewCloudWatchDispatcher(logGroupName, logStreamName, 7, cl)
			ca = NewFlowLogAggregator()
			ds := map[string]LogDispatcher{CWDispatcher: cd}
			mockFlushInterval := 600 * time.Second
			cr = NewFlowLogsReporter(ds, mockFlushInterval, nil, false, &NoOpLogOffset{})
			cr.AddAggregator(ca, []string{CWDispatcher})
			cr.timeNowFn = mt.getMockTime
			cr.Start()

			Expect(GetAndResetFlowsPerMinute()).Should(Equal(0.0))
		})
		It("Usage report is triggered post flushIntervalDuration", func() {
			By("Triggering report post flushIntervalDuration by mocking flushInterval")
			cl = testutil.NewMockedCloudWatchLogsClient(logGroupName)
			cd = NewCloudWatchDispatcher(logGroupName, logStreamName, 7, cl)
			ca = NewFlowLogAggregator()
			ca.IncludePolicies(true)
			ds := map[string]LogDispatcher{CWDispatcher: cd}
			cr = NewFlowLogsReporter(ds, flushInterval, nil, false, &NoOpLogOffset{})
			cr.AddAggregator(ca, []string{CWDispatcher})
			cr.timeNowFn = mt.getMockTime
			cr.Start()

			cr.Report(muNoConn1Rule1AllowUpdate)
			time.Sleep(1 * time.Second)

			Expect(GetAndResetFlowsPerMinute()).Should(BeNumerically(">", 0))
		})
	})
})

type logOffsetMock struct {
	mock.Mock
}

func (m *logOffsetMock) Read() Offsets {
	args := m.Called()
	v, _ := args.Get(0).(Offsets)
	return v
}

func (m *logOffsetMock) IsBehind(offsets Offsets) bool {
	args := m.Called()
	v, _ := args.Get(0).(bool)
	return v
}

func (m *logOffsetMock) GetIncreaseFactor(offsets Offsets) int {
	args := m.Called()
	v, _ := args.Get(0).(int)
	return v
}

type dispatcherMock struct {
	mock.Mock
	iteration    int
	maxIteration int
	collector    chan []*FlowLog
}

func (m *dispatcherMock) Initialize() error {
	m.collector = make(chan []*FlowLog)
	return nil
}

func (m *dispatcherMock) Dispatch(logSlice interface{}) error {
	m.iteration++
	log.Infof("Mocked dispatcher was called %d times ", m.iteration)
	var logs = logSlice.([]*FlowLog)
	log.Infof("Dispatching num=%d of logs", len(logs))
	if m.iteration <= m.maxIteration {
		m.collector <- logs
	}
	return nil
}

func (m *dispatcherMock) Close() {
	close(m.collector)
}

type mockTicker struct {
	mock.Mock
	tick chan time.Time
	stop chan bool
}

func (m *mockTicker) invokeTick(x time.Time) {
	m.tick <- x
}

func (m *mockTicker) Channel() <-chan time.Time {
	return m.tick
}

func (m *mockTicker) Stop() {
	close(m.tick)
	m.stop <- true
	close(m.stop)
}

func (m *mockTicker) Done() chan bool {
	return m.stop
}

var _ = Describe("FlowLogsReporter should adjust aggregation levels", func() {
	Context("Simulate flow logs being stalled in the pipeline", func() {
		It("increments with 1 level 2 times and decrements to the initial level", func() {
			// mock log offset to mark that the log pipeline is stalled for two iterations and then rectifies
			var mockLogOffset = &logOffsetMock{}
			mockLogOffset.On("Read").Return(Offsets{})
			mockLogOffset.On("IsBehind").Return(true).Times(2)
			mockLogOffset.On("IsBehind").Return(false)
			mockLogOffset.On("GetIncreaseFactor").Return(1)

			// mock ticker
			var ticker = &mockTicker{}
			ticker.tick = make(chan time.Time)
			ticker.stop = make(chan bool)
			defer ticker.Stop()

			// mock log dispatcher
			var cd = &dispatcherMock{}
			cd.maxIteration = 4
			defer cd.Close()
			var ds = map[string]LogDispatcher{"mock": cd}

			// add a flow log aggregator  to a reporter with a mocked log offset
			var cr = newFlowLogsReporterTest(ds, nil, false, ticker, mockLogOffset)
			var ca = NewFlowLogAggregator()
			cr.AddAggregator(ca, []string{"mock"})

			By("Starting the log reporter")
			cr.Start()

			var expectedLevel = 0
			// Feed reporter with log with two iterations
			var i = 0
			for ; i < 2; i++ {
				By(fmt.Sprintf("Feeding metric updates to the reporter as batch %d", i+1))
				cr.Report(muNoConn1Rule1AllowUpdate)
				ticker.invokeTick(time.Now())
				var logs = <-cd.collector
				Expect(len(logs)).To(Equal(1))
				Expect(int(ca.GetCurrentAggregationLevel())).To(Equal(expectedLevel + 1))
				expectedLevel++
			}

			// Feed reporter with another log with two iterations
			for ; i < 4; i++ {
				By(fmt.Sprintf("Feeding metric updates to the reporter as batch %d", i+1))
				cr.Report(muNoConn1Rule1AllowUpdate)
				ticker.invokeTick(time.Now())
				var logs = <-cd.collector
				Expect(len(logs)).To(Equal(1))
				Expect(ca.GetCurrentAggregationLevel()).To(Equal(FlowDefault))
			}
		})

		It("keeps the same aggregation level if the pipeline is not stalled", func() {
			var mockLogOffset = &logOffsetMock{}
			mockLogOffset.On("Read").Return(Offsets{})
			mockLogOffset.On("IsBehind").Return(false)
			mockLogOffset.On("GetIncreaseFactor").Return(1)

			// mock ticker
			var ticker = &mockTicker{}
			ticker.tick = make(chan time.Time)
			ticker.stop = make(chan bool)
			defer ticker.Stop()

			// mock log dispatcher
			var cd = &dispatcherMock{}
			cd.maxIteration = 4
			defer cd.Close()
			var ds = map[string]LogDispatcher{"mock": cd}

			// add a flow log aggregator  to a reporter with a mocked log offset
			var cr = newFlowLogsReporterTest(ds, nil, false, ticker, mockLogOffset)
			var ca = NewFlowLogAggregator()
			ca.AggregateOver(FlowPrefixName)
			cr.AddAggregator(ca, []string{"mock"})

			By("Starting the log reporter")
			cr.Start()

			var expectedLevel = 0
			// Feed reporter with log with four iterations

			for i := 0; i < 4; i++ {
				By(fmt.Sprintf("Feeding metric updates to the reporter as batch %d", i+1))
				cr.Report(muNoConn1Rule1AllowUpdate)
				ticker.invokeTick(time.Now())
				var logs = <-cd.collector
				Expect(len(logs)).To(Equal(1))
				Expect(ca.GetCurrentAggregationLevel()).To(Equal(FlowPrefixName))
				expectedLevel++
			}
		})

		/* Temporary disable test- https://tigera.atlassian.net/browse/SAAS-647

		It("increases the same aggregation level across multiple dispatchers", func() {
			// mock log offset to mark that the log pipeline is stalled
			var mockLogOffset = &logOffsetMock{}
			mockLogOffset.On("Read").Return(Offsets{})
			mockLogOffset.On("IsBehind").Return(true)
			mockLogOffset.On("GetIncreaseFactor").Return(1)

			// mock ticker
			var ticker = &mockTicker{}
			ticker.tick = make(chan time.Time)
			ticker.stop = make(chan bool)
			defer ticker.Stop()

			// mock log dispatcher
			var cd = &dispatcherMock{}
			cd.maxIteration = 1
			defer cd.Close()
			var ds = map[string]LogDispatcher{"mock": cd}

			// add two flow log aggregators to a reporter with a mocked log offset
			var cr = newFlowLogsReporterTest(ds, nil, false, ticker, mockLogOffset)
			// first aggregator will have level FlowPrefixName
			var oneAggregator = NewFlowLogAggregator()
			oneAggregator.AggregateOver(FlowPrefixName)

			// second aggregator will have level FlowDefault
			var anotherAggregator = NewFlowLogAggregator()
			anotherAggregator.AggregateOver(FlowDefault)
			cr.AddAggregator(oneAggregator, []string{"mock"})
			cr.AddAggregator(anotherAggregator, []string{"mock"})

			By("Starting the log reporter")
			cr.Start()

			// Feed reporter with log with one iterations
			for i := 0; i < 1; i++ {
				By(fmt.Sprintf("Feeding metric updates to the reporter as batch %d", i+1))
				cr.Report(muNoConn1Rule1AllowUpdate)
				ticker.invokeTick(time.Now())
				var logs = <-cd.collector
				Expect(len(logs)).To(Equal(1))
			}

			Expect(oneAggregator.GetCurrentAggregationLevel()).To(Equal(FlowNoDestPorts))
			Expect(anotherAggregator.GetCurrentAggregationLevel()).To(Equal(FlowSourcePort))

		})*/

		It("increases only to the max level", func() {
			var mockLogOffset = &logOffsetMock{}
			mockLogOffset.On("Read").Return(Offsets{})
			mockLogOffset.On("IsBehind").Return(true)
			mockLogOffset.On("GetIncreaseFactor").Return(1)

			// mock ticker
			var ticker = &mockTicker{}
			ticker.tick = make(chan time.Time)
			ticker.stop = make(chan bool)
			defer ticker.Stop()

			// mock log dispatcher
			var cd = &dispatcherMock{}
			cd.maxIteration = 5
			defer cd.Close()
			var ds = map[string]LogDispatcher{"mock": cd}

			// add a flow log aggregator  to a reporter with a mocked log offset
			var cr = newFlowLogsReporterTest(ds, nil, false, ticker, mockLogOffset)
			var ca = NewFlowLogAggregator()
			ca.AggregateOver(FlowPrefixName)
			cr.AddAggregator(ca, []string{"mock"})

			By("Starting the log reporter")
			cr.Start()

			// Feed reporter with log with five iterations

			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("Feeding metric updates to the reporter as batch %d", i+1))
				cr.Report(muNoConn1Rule1AllowUpdate)
				ticker.invokeTick(time.Now())
				var logs = <-cd.collector
				Expect(len(logs)).To(Equal(1))
			}

			Expect(ca.GetCurrentAggregationLevel()).To(Equal(MaxAggregationLevel))
		})

	})
})

type timingoutCWFLMockClient struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	timeout time.Duration
}

func (c *timingoutCWFLMockClient) DescribeLogGroupsWithContext(ctx aws.Context, input *cloudwatchlogs.DescribeLogGroupsInput, req ...request.Option) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	time.Sleep(c.timeout)
	return nil, awserr.New(cloudwatchlogs.ErrCodeServiceUnavailableException, "cloudwatch logs service not available", nil)
}

func (c *timingoutCWFLMockClient) CreateLogGroupWithContext(ctx aws.Context, input *cloudwatchlogs.CreateLogGroupInput, req ...request.Option) (*cloudwatchlogs.CreateLogGroupOutput, error) {
	time.Sleep(c.timeout)
	return nil, awserr.New(cloudwatchlogs.ErrCodeServiceUnavailableException, "cloudwatch logs service not available", nil)
}

func getFlowLog(fl string) (FlowLog, error) {
	flowLog := &FlowLog{}
	err := flowLog.Deserialize(fl)
	return *flowLog, err
}

func newExpectedFlowLog(t Tuple, nf, nfs, nfc int, a FlowLogAction, fr FlowLogReporter, pi, po, bi, bo int, srcMeta, dstMeta EndpointMetadata, dstService FlowService, srcLabels, dstLabels map[string]string, fp FlowPolicies, fe FlowExtras, fpi testProcessInfo) FlowLog {
	return FlowLog{
		FlowMeta: FlowMeta{
			Tuple:      t,
			Action:     a,
			Reporter:   fr,
			SrcMeta:    srcMeta,
			DstMeta:    dstMeta,
			DstService: dstService,
		},
		FlowLabels: FlowLabels{
			SrcLabels: srcLabels,
			DstLabels: dstLabels,
		},
		FlowPolicies: fp,
		FlowExtras:   fe,
		FlowProcessReportedStats: FlowProcessReportedStats{
			ProcessName:     fpi.processName,
			NumProcessNames: fpi.numProcessNames,
			ProcessID:       fpi.processID,
			NumProcessIDs:   fpi.numProcessIDs,
			FlowReportedStats: FlowReportedStats{
				NumFlows:          nf,
				NumFlowsStarted:   nfs,
				NumFlowsCompleted: nfc,
				PacketsIn:         pi,
				PacketsOut:        po,
				BytesIn:           bi,
				BytesOut:          bo,
			},
		},
	}
}
