// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"sync"
	"time"

	"github.com/google/gopacket/layers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/testutils"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

type testDispatcher struct {
	mutex sync.Mutex
	logs  []*DNSLog
}

func (d *testDispatcher) Initialize() error {
	return nil
}

func (d *testDispatcher) Dispatch(logSlice interface{}) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	log.Info("In dispatch")
	fl := logSlice.([]*DNSLog)
	d.logs = append(d.logs, fl...)
	return nil
}

func (d *testDispatcher) getLogs() []*DNSLog {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.logs
}

type testSink struct {
	name          string
	includeLabels bool
	aggregation   DNSAggregationKind
	aggregator    DNSLogAggregator
	dispatcher    *testDispatcher
}

var _ = Describe("DNS Log Reporter", func() {

	var (
		sinks        []*testSink
		flushTrigger chan time.Time
		reporter     *DNSLogReporter
	)

	JustBeforeEach(func() {
		sinks = nil
		sinks = append(sinks, &testSink{name: "noLabelsOrAgg", includeLabels: false, aggregation: DNSDefault})
		sinks = append(sinks, &testSink{name: "LabelsAndAgg", includeLabels: true, aggregation: DNSPrefixNameAndIP})
		sinks = append(sinks, &testSink{name: "LabelsNoAgg", includeLabels: true, aggregation: DNSDefault})
		dispatcherMap := map[string]LogDispatcher{}
		for _, sink := range sinks {
			sink.aggregator = NewDNSLogAggregator().IncludeLabels(sink.includeLabels).AggregateOver(sink.aggregation)
			sink.dispatcher = &testDispatcher{}
			dispatcherMap[sink.name] = sink.dispatcher
		}
		flushTrigger = make(chan time.Time)
		reporter = NewDNSLogReporterWithShims(dispatcherMap, flushTrigger, nil)
		for _, sink := range sinks {
			reporter.AddAggregator(sink.aggregator, []string{sink.name})
		}
		reporter.Start()
	})

	It("should generate correct logs", func() {
		dns := &layers.DNS{}
		dns.Questions = append(dns.Questions, testutils.MakeQ("google.com"))
		dns.Answers = append(dns.Answers, testutils.MakeA("google.com", "1.1.1.1"))

		client1 := &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "host1",
				OrchestratorID: "k8s",
				WorkloadID:     "alice/test1-a345cf",
				EndpointID:     "ep1",
			},
			Endpoint: &model.WorkloadEndpoint{
				Name:         "test1-a345cf",
				GenerateName: "test1",
				Labels: map[string]string{
					"group":    "test1",
					"name":     "test1-a345cf",
					"common":   "red",
					"specific": "socks",
				},
			},
		}
		client2 := &calc.EndpointData{
			Key: model.WorkloadEndpointKey{
				Hostname:       "host1",
				OrchestratorID: "k8s",
				WorkloadID:     "alice/test1-56dca3",
				EndpointID:     "ep2",
			},
			Endpoint: &model.WorkloadEndpoint{
				Name:         "test1-56dca3",
				GenerateName: "test1",
				Labels: map[string]string{
					"group":    "test1",
					"name":     "test1-56dca3",
					"common":   "red",
					"specific": "shoes",
				},
			},
		}
		err := reporter.Log(DNSUpdate{
			ClientEP: client1,
			ClientIP: net.ParseIP("1.2.3.4"),
			ServerIP: net.ParseIP("8.8.8.8"),
			DNS:      dns,
		})
		Expect(err).NotTo(HaveOccurred())
		err = reporter.Log(DNSUpdate{
			ClientEP: client2,
			ClientIP: net.ParseIP("1.2.3.5"),
			ServerIP: net.ParseIP("8.8.8.8"),
			DNS:      dns,
		})
		Expect(err).NotTo(HaveOccurred())
		flushTrigger <- time.Now()

		commonChecks := func(l *DNSLog) {
			Expect(l.ClientNameAggr).To(Equal("test1*"))
			Expect(l.ClientNamespace).To(Equal("alice"))
		}

		// Logs with no aggregation and no labels.
		Eventually(sinks[0].dispatcher.getLogs).Should(HaveLen(2))
		for _, l := range sinks[0].dispatcher.getLogs() {
			commonChecks(l)
			Expect(l.Count).To(BeNumerically("==", 1))
			Expect(l.ClientName).To(ContainSubstring("test1-"))
			Expect(l.ClientIP).NotTo(BeNil())
			Expect(*l.ClientIP).To(ContainSubstring("1.2.3."))
			Expect(l.ClientLabels).To(BeNil())
		}

		// Logs with aggregation and labels.
		Eventually(sinks[1].dispatcher.getLogs).Should(HaveLen(1))
		for _, l := range sinks[1].dispatcher.getLogs() {
			commonChecks(l)
			Expect(l.Count).To(BeNumerically("==", 2))
			Expect(l.ClientName).To(Equal(flowLogFieldNotIncluded))
			Expect(l.ClientIP).To(BeNil())
			Expect(l.ClientLabels).To(Equal(map[string]string{
				"group":  "test1",
				"common": "red",
			}))
		}

		// Logs with labels but no aggregation.
		Eventually(sinks[2].dispatcher.getLogs).Should(HaveLen(2))
		for _, l := range sinks[2].dispatcher.getLogs() {
			commonChecks(l)
			Expect(l.Count).To(BeNumerically("==", 1))
			Expect(l.ClientName).To(ContainSubstring("test1-"))
			Expect(l.ClientIP).NotTo(BeNil())
			Expect(*l.ClientIP).To(ContainSubstring("1.2.3."))
			Expect(l.ClientLabels).To(Or(
				Equal(map[string]string{
					"group":    "test1",
					"name":     "test1-a345cf",
					"common":   "red",
					"specific": "socks",
				}),
				Equal(map[string]string{
					"group":    "test1",
					"name":     "test1-56dca3",
					"common":   "red",
					"specific": "shoes",
				})))
		}
	})
})
