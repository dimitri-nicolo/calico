// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package winfv_test

import (
	"errors"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/windows-networking/pkg/testutils"

	"github.com/projectcalico/calico/felix/fv/metrics"
	. "github.com/projectcalico/calico/felix/fv/winfv"
	"github.com/projectcalico/calico/libcalico-go/lib/winutils"
)

// Generate traffic flows to test flow logs on Windows nodes.
// Common features which are not specific on Windows (e.g. cloudwatch)
// are tested with Linux FVs.
//
// Infra setup
//
//             Windows node                         Linx node
//
//              porter                          client (allowed to porter)
//                                              client-b (denied to porter)
//                                              nginx
//

type aggregation int

const (
	AggrNone         aggregation = 0
	AggrBySourcePort aggregation = 1
	AggrByPodPrefix  aggregation = 2
)

type expectation struct {
	labels                bool
	policies              bool
	aggregationForAllowed aggregation
	aggregationForDenied  aggregation
}

var _ = Describe("Windows flow logs test", func() {
	var (
		expectation                    expectation
		flowLogsReaders                []metrics.FlowLogReader
		porter, client, clientB, nginx string
		fv                             *WinFV
		err                            error
	)

	BeforeEach(func() {
		Skip("Temporarily skip failing flow log tests on HPC")
		fv, err = NewWinFV(winutils.GetHostPath("c:\\CalicoWindows"),
			winutils.GetHostPath("c:\\TigeraCalico\\flowlogs"),
			winutils.GetHostPath("c:\\TigeraCalico\\felix-dns-cache.txt"))
		Expect(err).NotTo(HaveOccurred())

		flowLogsReaders = []metrics.FlowLogReader{fv}

		// Get Pod IPs.
		client = testutils.InfraPodIP("client", "demo")
		clientB = testutils.InfraPodIP("client-b", "demo")
		porter = testutils.InfraPodIP("porter", "demo")
		nginx = testutils.InfraPodIP("nginx", "demo")
		log.Infof("Pod IP client %s, client-b %s, porter %s, nginx %s",
			client, clientB, porter, nginx)

		Expect(client).NotTo(BeEmpty())
		Expect(clientB).NotTo(BeEmpty())
		Expect(porter).NotTo(BeEmpty())
		Expect(nginx).NotTo(BeEmpty())
	})

	checkFlowLogs := func() {
		// Within 60s we should see the complete set of expected allow and deny
		// flow logs.
		Eventually(func() error {
			flowTester := metrics.NewFlowTesterDeprecated(flowLogsReaders, expectation.labels, expectation.policies, 80)
			if fv.GetBackendType() == CalicoBackendVXLAN {
				// Windows VXLAN can't complete a flow in time.
				flowTester.IgnoreStartCompleteCount = true
			}
			err := flowTester.PopulateFromFlowLogs()
			if err != nil {
				return err
			}

			// Only report errors at the end.
			var errs []string

			// Now we tick off each FlowMeta that we expect, and check that
			// the log(s) for each one are present and as expected.
			switch expectation.aggregationForAllowed {
			case AggrNone:
				err = flowTester.CheckFlow(
					"wep demo client client", client,
					"wep demo porter porter", porter,
					metrics.NoService, 1, 1,
					[]metrics.ExpectedPolicy{
						{
							Reporter: "dst",
							Action:   "allow",
							Policies: []string{"0|default|demo/knp.default.allow-client|allow|0"},
						},
					})
				if err != nil {
					errs = append(errs, fmt.Sprintf("Error agg for allowed; agg pod prefix; flow 1: %v", err))
				}
				err = flowTester.CheckFlow(
					"wep demo porter porter", porter,
					"wep demo nginx nginx", nginx,
					"demo nginx - 80", 1, 1,
					[]metrics.ExpectedPolicy{
						{
							Reporter: "src",
							Action:   "allow",
							Policies: []string{"0|default|demo/knp.default.allow-nginx|allow|0"},
						},
					})
				if err != nil {
					errs = append(errs, fmt.Sprintf("Error agg for allowed; agg pod prefix; flow 2: %v", err))
				}
			case AggrByPodPrefix:
				err = flowTester.CheckFlow(
					"wep demo - client", "",
					"wep demo - porter", "",
					metrics.NoService, 1, 1,
					[]metrics.ExpectedPolicy{
						{
							Reporter: "dst",
							Action:   "allow",
							Policies: []string{"0|default|demo/knp.default.allow-client|allow|0"},
						},
					})
				if err != nil {
					errs = append(errs, fmt.Sprintf("Error agg for allowed; agg pod prefix; flow 1: %v", err))
				}
				err = flowTester.CheckFlow(
					"wep demo - porter", "",
					"wep demo - nginx", "",
					"demo nginx - 80", 1, 1,
					[]metrics.ExpectedPolicy{
						{
							Reporter: "src",
							Action:   "allow",
							Policies: []string{"0|default|demo/knp.default.allow-nginx|allow|0"},
						},
					})
				if err != nil {
					errs = append(errs, fmt.Sprintf("Error agg for allowed; agg pod prefix; flow 2: %v", err))
				}
			}
			switch expectation.aggregationForDenied {
			case AggrNone:
				err = flowTester.CheckFlow(
					"wep demo client-b client-b", clientB,
					"wep demo porter porter", porter,
					metrics.NoService, 1, 1,
					[]metrics.ExpectedPolicy{
						{
							Reporter: "dst",
							Action:   "deny",
							Policies: []string{"0|__PROFILE__|__PROFILE__.__NO_MATCH__|deny|0"},
						},
					})
				if err != nil {
					errs = append(errs, fmt.Sprintf("Error agg for denied; agg pod prefix: %v", err))
				}
			case AggrBySourcePort:
				err = flowTester.CheckFlow(
					"wep demo client-b client-b", clientB,
					"wep demo porter porter", porter,
					metrics.NoService, 1, 1,
					[]metrics.ExpectedPolicy{
						{
							Reporter: "dst",
							Action:   "deny",
							Policies: []string{"0|__PROFILE__|__PROFILE__.__NO_MATCH__|deny|0"},
						},
					})
				if err != nil {
					errs = append(errs, fmt.Sprintf("Error agg for denied; agg pod prefix: %v", err))
				}
			}

			// Finally check that there are no remaining flow logs that we did not expect.
			err = flowTester.CheckAllFlowsAccountedFor()
			if err != nil {
				errs = append(errs, err.Error())
			}

			if len(errs) == 0 {
				return nil
			}

			return errors.New(strings.Join(errs, "\n==============\n"))

		}, "60s", "10s").ShouldNot(HaveOccurred())
	}

	Context("File flow logs only", func() {
		setupAndRunFelix := func(config map[string]interface{}) {
			err := fv.AddConfigItems(config)
			Expect(err).NotTo(HaveOccurred())

			fv.RestartFelix()

			// Initiate traffic.
			testutils.InfraInitiateTraffic()
		}

		It("should get expected flow logs with no aggregation", func() {
			expectation.labels = true
			expectation.policies = true
			expectation.aggregationForAllowed = AggrNone
			expectation.aggregationForDenied = AggrNone

			config := map[string]interface{}{
				"FlowLogsFileAggregationKindForAllowed": 0,
				"FlowLogsFileAggregationKindForDenied":  0,
				"FlowLogsFlushInterval":                 "10",
			}
			setupAndRunFelix(config)

			checkFlowLogs()
		})

		It("should get expected flow logs with default aggregation", func() {
			expectation.labels = true
			expectation.policies = true
			expectation.aggregationForAllowed = AggrByPodPrefix
			expectation.aggregationForDenied = AggrBySourcePort

			config := map[string]interface{}{
				"FlowLogsFileAggregationKindForAllowed": 2,
				"FlowLogsFileAggregationKindForDenied":  1,
				"FlowLogsFlushInterval":                 "10",
			}
			setupAndRunFelix(config)

			checkFlowLogs()
		})

		AfterEach(func() {
			err := fv.RestoreConfig()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
