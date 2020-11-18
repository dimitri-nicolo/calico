// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

type testL7Dispatcher struct {
	mutex sync.Mutex
	logs  []*L7Log
}

func (d *testL7Dispatcher) Initialize() error {
	return nil
}

func (d *testL7Dispatcher) Dispatch(logSlice interface{}) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	log.Info("In dispatch")
	fl := logSlice.([]*L7Log)
	d.logs = append(d.logs, fl...)
	return nil
}

func (d *testL7Dispatcher) getLogs() []*L7Log {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.logs
}

var _ = Describe("L7 Log Reporter", func() {

	var (
		ed1, ed2, ed3 *calc.EndpointData
		dispatcher    *testL7Dispatcher
		flushTrigger  chan time.Time
		reporter      *L7LogReporter
	)

	JustBeforeEach(func() {
		dispatcherMap := map[string]LogDispatcher{}
		dispatcher = &testL7Dispatcher{}
		dispatcherMap["testL7"] = dispatcher
		flushTrigger = make(chan time.Time)
		reporter = NewL7LogReporterWithShims(dispatcherMap, flushTrigger, nil)
		reporter.AddAggregator(NewL7LogAggregator(), []string{"testL7"})
		reporter.Start()
		remoteWlEpKey1 := model.WorkloadEndpointKey{
			OrchestratorID: "orchestrator",
			WorkloadID:     "default/remoteworkloadid1",
			EndpointID:     "remoteepid1",
		}
		ed1 = &calc.EndpointData{
			Key:      remoteWlEpKey1,
			Endpoint: remoteWlEp1,
			IsLocal:  false,
		}
		remoteWlEpKey2 := model.WorkloadEndpointKey{
			OrchestratorID: "orchestrator",
			WorkloadID:     "default/remoteworkloadid2",
			EndpointID:     "remoteepid2",
		}
		ed2 = &calc.EndpointData{
			Key:      remoteWlEpKey2,
			Endpoint: remoteWlEp2,
			IsLocal:  false,
		}
		localWlEPKey1 := model.WorkloadEndpointKey{
			Hostname:       "localhost",
			OrchestratorID: "orchestrator",
			WorkloadID:     "default/localworkloadid1",
			EndpointID:     "localepid1",
		}
		ed3 = &calc.EndpointData{
			Key:      localWlEPKey1,
			Endpoint: localWlEp1,
			IsLocal:  true,
			Ingress: &calc.MatchData{
				PolicyMatches: map[calc.PolicyID]int{
					calc.PolicyID{Name: "policy1", Tier: "default"}: 0,
					calc.PolicyID{Name: "policy2", Tier: "default"}: 0,
				},
				TierData: map[string]*calc.TierData{
					"default": {
						ImplicitDropRuleID: calc.NewRuleID("default", "policy2", "", calc.RuleIDIndexImplicitDrop,
							rules.RuleDirIngress, rules.RuleActionDeny),
						EndOfTierMatchIndex: 0,
					},
				},
				ProfileMatchIndex: 0,
			},
			Egress: &calc.MatchData{
				PolicyMatches: map[calc.PolicyID]int{
					calc.PolicyID{Name: "policy1", Tier: "default"}: 0,
					calc.PolicyID{Name: "policy2", Tier: "default"}: 0,
				},
				TierData: map[string]*calc.TierData{
					"default": {
						ImplicitDropRuleID: calc.NewRuleID("default", "policy2", "", calc.RuleIDIndexImplicitDrop,
							rules.RuleDirIngress, rules.RuleActionDeny),
						EndOfTierMatchIndex: 0,
					},
				},
				ProfileMatchIndex: 0,
			},
		}
	})

	It("should generate correct logs", func() {
		err := reporter.Log(L7Update{
			Tuple:         *NewTuple(remoteIp1, remoteIp2, proto_tcp, srcPort, dstPort),
			SrcEp:         ed1,
			DstEp:         ed2,
			Duration:      10,
			DurationMax:   12,
			BytesReceived: 500,
			BytesSent:     30,
			ResponseCode:  200,
			Method:        "GET",
			Domain:        "www.test.com",
			Path:          "/test/path",
			UserAgent:     "firefox",
			Type:          "html/1.1",
			Count:         1,
		})
		Expect(err).NotTo(HaveOccurred())
		err = reporter.Log(L7Update{
			Tuple:         *NewTuple(remoteIp1, localIp1, proto_tcp, srcPort, dstPort),
			SrcEp:         ed1,
			DstEp:         ed3,
			Duration:      20,
			DurationMax:   22,
			BytesReceived: 30,
			BytesSent:     50,
			ResponseCode:  200,
			Method:        "GET",
			Domain:        "www.testanother.com",
			Path:          "/test/different",
			UserAgent:     "firefox",
			Type:          "html/1.1",
			Count:         1,
		})
		Expect(err).NotTo(HaveOccurred())
		flushTrigger <- time.Now()
		time.Sleep(1 * time.Second)

		commonChecks := func(l *L7Log) {
			Expect(l.SrcNameAggr).To(Equal("remoteworkloadid1"))
			Expect(l.SrcNamespace).To(Equal("default"))
			Expect(l.SrcType).To(Equal(FlowLogEndpointTypeWep))
			Expect(l.Method).To(Equal("GET"))
			Expect(l.UserAgent).To(Equal("firefox"))
			Expect(l.ResponseCode).To(Equal(200))
			Expect(l.Type).To(Equal("html/1.1"))
			Expect(l.Count).To(Equal(1))
		}

		Eventually(dispatcher.getLogs()).Should(HaveLen(2))
		logs := dispatcher.getLogs()

		for _, l := range logs {
			commonChecks(l)
			if l.DstNameAggr == "remoteworkloadid2" {
				// TODO: Add service name checks
				Expect(l.DurationMean).To(Equal(10 * time.Millisecond))
				Expect(l.DurationMax).To(Equal(12 * time.Millisecond))
				Expect(l.BytesIn).To(Equal(500))
				Expect(l.BytesOut).To(Equal(30))
				Expect(l.DstNameAggr).To(Equal("remoteworkloadid2"))
				Expect(l.DstNamespace).To(Equal("default"))
				Expect(l.DstType).To(Equal(FlowLogEndpointTypeWep))
				Expect(l.URL).To(Equal("www.test.com/test/path"))
			} else {
				Expect(l.DurationMean).To(Equal(20 * time.Millisecond))
				Expect(l.DurationMax).To(Equal(22 * time.Millisecond))
				Expect(l.BytesIn).To(Equal(30))
				Expect(l.BytesOut).To(Equal(50))
				Expect(l.DstNameAggr).To(Equal("localworkloadid1"))
				Expect(l.DstNamespace).To(Equal("default"))
				Expect(l.DstType).To(Equal(FlowLogEndpointTypeWep))
				Expect(l.URL).To(Equal("www.testanother.com/test/different"))
			}
		}
	})
})
