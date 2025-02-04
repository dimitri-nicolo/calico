// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package wafevents

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/projectcalico/calico/felix/collector/types"
	"github.com/projectcalico/calico/felix/proto"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type testWAFEventReporter struct {
	logs chan []*v1.WAFLog
}

func (r *testWAFEventReporter) Start() error {
	return nil
}

func (r *testWAFEventReporter) Report(logs interface{}) error {
	log.Info("In dispatch")
	r.logs <- logs.([]*v1.WAFLog)
	return nil
}

var _ = Describe("WAFEvent Log Reporter", func() {
	var (
		r0, r1, r2   *Report
		dispatcher   *testWAFEventReporter
		flushTrigger chan time.Time
		r            *WAFEventReporter
	)

	JustBeforeEach(func() {
		dispatcher = &testWAFEventReporter{logs: make(chan []*v1.WAFLog)}
		flushTrigger = make(chan time.Time)
		r = NewReporterWithShims([]types.Reporter{dispatcher}, flushTrigger, nil)
		Expect(r.Start()).NotTo(HaveOccurred())

		// r0, sql injection, pass
		r0 = &Report{
			Src: &v1.WAFEndpoint{
				IP:           "10.0.0.1",
				PortNum:      65500,
				PodName:      "pod-client-0",
				PodNameSpace: "default-ns",
			},
			Dst: &v1.WAFEndpoint{
				IP:           "10.0.0.100",
				PortNum:      8080,
				PodName:      "pod-server-0",
				PodNameSpace: "server-ns",
			},
			WAFEvent: &proto.WAFEvent{
				TxId:    "id000",
				Host:    "server-svc.server-ns",
				SrcIp:   "10.0.0.1",
				SrcPort: 65500,
				DstIp:   "10.0.0.100",
				DstPort: 8080,
				Rules: []*proto.WAFRuleHit{
					{
						Rule: &proto.WAFRule{
							Id:       "942100",
							Message:  "SQL Injection Attack Detected via libinjection",
							Severity: "critical",
							File:     "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
							Line:     "5342",
						},
						Disruptive: false,
					},
					{
						Rule: &proto.WAFRule{
							Id:       "949110",
							Message:  "Inbound Anomaly Score Exceeded (Total Score: 5)",
							Severity: "emergency",
							File:     "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-949-BLOCKING-EVALUATION.conf",
							Line:     "7134",
						},
						Disruptive: false,
					},
				},
				Action: "pass",
				Request: &proto.HTTPRequest{
					Method:  "GET",
					Path:    "/",
					Version: "1.1",
					Headers: map[string]string{
						"User-Agent": "Firefox",
					},
				},
				Timestamp: &timestamppb.Timestamp{Seconds: 58800},
			},
		}
		// r1, sql injection, block
		r1 = &Report{
			Src: &v1.WAFEndpoint{
				IP:           "10.0.1.1",
				PortNum:      65501,
				PodName:      "pod-client-1",
				PodNameSpace: "default-ns",
			},
			Dst: &v1.WAFEndpoint{
				IP:           "10.0.0.100",
				PortNum:      8080,
				PodName:      "pod-server-0",
				PodNameSpace: "server-ns",
			},
			WAFEvent: &proto.WAFEvent{
				TxId:    "id001",
				Host:    "server-svc.server-ns",
				SrcIp:   "10.0.1.1",
				SrcPort: 65501,
				DstIp:   "10.0.0.100",
				DstPort: 8080,
				Rules: []*proto.WAFRuleHit{
					{
						Rule: &proto.WAFRule{
							Id:       "942100",
							Message:  "SQL Injection Attack Detected via libinjection",
							Severity: "critical",
							File:     "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
							Line:     "5342",
						},
						Disruptive: true,
					},
					{
						Rule: &proto.WAFRule{
							Id:       "949110",
							Message:  "Inbound Anomaly Score Exceeded (Total Score: 5)",
							Severity: "emergency",
							File:     "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-949-BLOCKING-EVALUATION.conf",
							Line:     "7134",
						},
						Disruptive: true,
					},
				},
				Action: "block",
				Request: &proto.HTTPRequest{
					Method:  "GET",
					Path:    "/",
					Version: "1.1",
					Headers: map[string]string{
						"User-Agent": "Firefox",
					},
				},
				Timestamp: &timestamppb.Timestamp{Seconds: 58800},
			},
		}
		// r2, OS File Access Attempt, block
		r2 = &Report{
			Src: &v1.WAFEndpoint{
				IP:           "10.0.2.1",
				PortNum:      65502,
				PodName:      "pod-client-2",
				PodNameSpace: "default-ns",
			},
			Dst: &v1.WAFEndpoint{
				IP:           "10.0.0.100",
				PortNum:      8080,
				PodName:      "pod-server-0",
				PodNameSpace: "server-ns",
			},
			WAFEvent: &proto.WAFEvent{
				TxId:    "id002",
				Host:    "server-svc.server-ns",
				SrcIp:   "10.0.2.1",
				SrcPort: 65502,
				DstIp:   "10.0.0.100",
				DstPort: 8080,
				Rules: []*proto.WAFRuleHit{
					{
						Rule: &proto.WAFRule{
							Id:       "930120",
							Message:  "OS File Access Attempt",
							Severity: "critical",
							File:     "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-930-APPLICATION-ATTACK-LFI.conf",
							Line:     "3091",
						},
						Disruptive: true,
					},
					{
						Rule: &proto.WAFRule{
							Id:       "932160",
							Message:  "Remote Command Execution: Unix Shell Code Found",
							Severity: "critical",
							File:     "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-932-APPLICATION-ATTACK-RCE.conf",
							Line:     "3488",
						},
						Disruptive: true,
					},
					{
						Rule: &proto.WAFRule{
							Id:       "949110",
							Message:  "Inbound Anomaly Score Exceeded (Total Score: 10)",
							Severity: "emergency",
							File:     "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-949-BLOCKING-EVALUATION.conf",
							Line:     "7134",
						},
						Disruptive: true,
					},
				},
				Action: "block",
				Request: &proto.HTTPRequest{
					Method:  "GET",
					Path:    "/",
					Version: "1.1",
					Headers: map[string]string{
						"User-Agent": "Firefox",
					},
				},
				Timestamp: &timestamppb.Timestamp{Seconds: 58800},
			},
		}
	})

	It("should generate correct logs", func() {
		// report the events
		err := r.Report(r0)
		Expect(err).NotTo(HaveOccurred())
		err = r.Report(r1)
		Expect(err).NotTo(HaveOccurred())
		err = r.Report(r2)
		Expect(err).NotTo(HaveOccurred())

		// flush and verify logs
		flushTrigger <- time.Now()
		logs := <-dispatcher.logs
		Expect(logs).To(HaveLen(3))

		Expect(logs).To(ContainElements(
			BeComparableTo(&v1.WAFLog{
				Timestamp: time.Unix(58800, 0),
				Destination: &v1.WAFEndpoint{
					IP:           "10.0.0.100",
					PortNum:      8080,
					PodName:      "pod-server-0",
					PodNameSpace: "server-ns",
				},
				Method:    "GET",
				Msg:       "WAF detected 2 violations [ pass ]",
				Path:      "/",
				Protocol:  "HTTP/1.1",
				RequestId: "id000",
				Rules: []v1.WAFRuleHit{
					{
						Id:         "942100",
						Message:    "SQL Injection Attack Detected via libinjection",
						Severity:   "critical",
						File:       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
						Line:       "5342",
						Disruptive: false,
					},
					{
						Id:         "949110",
						Message:    "Inbound Anomaly Score Exceeded (Total Score: 5)",
						Severity:   "emergency",
						File:       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-949-BLOCKING-EVALUATION.conf",
						Line:       "7134",
						Disruptive: false,
					},
				},
				Source: &v1.WAFEndpoint{
					IP:           "10.0.0.1",
					PortNum:      65500,
					PodName:      "pod-client-0",
					PodNameSpace: "default-ns",
				},
				Host: "server-svc.server-ns",
			}),
			BeComparableTo(&v1.WAFLog{
				Timestamp: time.Unix(58800, 0),
				Destination: &v1.WAFEndpoint{
					IP:           "10.0.0.100",
					PortNum:      8080,
					PodName:      "pod-server-0",
					PodNameSpace: "server-ns",
				},
				Method:    "GET",
				Msg:       "WAF detected 2 violations [ block ]",
				Path:      "/",
				Protocol:  "HTTP/1.1",
				RequestId: "id001",
				Rules: []v1.WAFRuleHit{
					{
						Id:         "942100",
						Message:    "SQL Injection Attack Detected via libinjection",
						Severity:   "critical",
						File:       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
						Line:       "5342",
						Disruptive: true,
					},
					{
						Id:         "949110",
						Message:    "Inbound Anomaly Score Exceeded (Total Score: 5)",
						Severity:   "emergency",
						File:       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-949-BLOCKING-EVALUATION.conf",
						Line:       "7134",
						Disruptive: true,
					},
				},
				Source: &v1.WAFEndpoint{
					IP:           "10.0.1.1",
					PortNum:      65501,
					PodName:      "pod-client-1",
					PodNameSpace: "default-ns",
				},
				Host: "server-svc.server-ns",
			}),
			BeComparableTo(&v1.WAFLog{
				Timestamp: time.Unix(58800, 0),
				Destination: &v1.WAFEndpoint{
					IP:           "10.0.0.100",
					PortNum:      8080,
					PodName:      "pod-server-0",
					PodNameSpace: "server-ns",
				},
				Method:    "GET",
				Msg:       "WAF detected 3 violations [ block ]",
				Path:      "/",
				Protocol:  "HTTP/1.1",
				RequestId: "id002",
				Rules: []v1.WAFRuleHit{
					{
						Id:         "930120",
						Message:    "OS File Access Attempt",
						Severity:   "critical",
						File:       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-930-APPLICATION-ATTACK-LFI.conf",
						Line:       "3091",
						Disruptive: true,
					},
					{
						Id:         "932160",
						Message:    "Remote Command Execution: Unix Shell Code Found",
						Severity:   "critical",
						File:       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-932-APPLICATION-ATTACK-RCE.conf",
						Line:       "3488",
						Disruptive: true,
					},
					{
						Id:         "949110",
						Message:    "Inbound Anomaly Score Exceeded (Total Score: 10)",
						Severity:   "emergency",
						File:       "/etc/modsecurity-ruleset/@owasp_crs/REQUEST-949-BLOCKING-EVALUATION.conf",
						Line:       "7134",
						Disruptive: true,
					},
				},
				Source: &v1.WAFEndpoint{
					IP:           "10.0.2.1",
					PortNum:      65502,
					PodName:      "pod-client-2",
					PodNameSpace: "default-ns",
				},
				Host: "server-svc.server-ns",
			}),
		))
	})

	It("should correct aggregate logs", func() {
		// report the events
		err := r.Report(r0)
		Expect(err).NotTo(HaveOccurred())
		err = r.Report(r0)
		Expect(err).NotTo(HaveOccurred())
		err = r.Report(r1)
		Expect(err).NotTo(HaveOccurred())
		err = r.Report(r2)
		Expect(err).NotTo(HaveOccurred())
		err = r.Report(r2)
		Expect(err).NotTo(HaveOccurred())
		err = r.Report(r2)
		Expect(err).NotTo(HaveOccurred())

		// flush and verify logs
		flushTrigger <- time.Now()
		logs := <-dispatcher.logs
		Expect(logs).To(HaveLen(3))
	})

	It("should perform on huge loads", func() {
		// get start time
		start := time.Now()

		// report the 100k events
		for i := 0; i < 25000; i++ {
			err := r.Report(r0)
			Expect(err).NotTo(HaveOccurred())
		}
		for i := 0; i < 75000; i++ {
			err := r.Report(r1)
			Expect(err).NotTo(HaveOccurred())
		}

		// flush and verify logs
		flushTrigger <- time.Now()
		logs := <-dispatcher.logs
		Expect(logs).To(HaveLen(2))

		// test if it takes less than 7 secs
		Expect(time.Since(start)).To(BeNumerically("<", 10*time.Second))
	})
})
