//go:build fvtests
// +build fvtests

// Copyright (c) 2023-2025 Tigera, Inc. All rights reserved.

package fv_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/collector/flowlog"
	"github.com/projectcalico/calico/felix/fv/connectivity"
	"github.com/projectcalico/calico/felix/fv/flowlogs"
	"github.com/projectcalico/calico/felix/fv/infrastructure"
	"github.com/projectcalico/calico/felix/fv/workload"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
)

var _ = infrastructure.DatastoreDescribe("_BPF-SAFE_ flow log with TCP stats", []apiconfig.DatastoreType{apiconfig.Kubernetes}, func(getInfra infrastructure.InfraFactory) {
	var (
		infra           infrastructure.DatastoreInfra
		opts            infrastructure.TopologyOptions
		tc              infrastructure.TopologyContainers
		flowLogsReaders []flowlogs.FlowLogReader
		ep1_1           *workload.Workload
		ep1_2           *workload.Workload
	)

	bpfEnabled := os.Getenv("FELIX_FV_ENABLE_BPF") == "true"

	BeforeEach(func() {
		infra = getInfra()
		opts = infrastructure.DefaultTopologyOptions()
		opts.FlowLogSource = infrastructure.FlowLogSourceFile

		opts.IPIPEnabled = false
		opts.ExtraEnvVars["FELIX_FLOWLOGSFILEENABLED"] = "true"
		opts.ExtraEnvVars["FELIX_FLOWLOGSFLUSHINTERVAL"] = "5"
		opts.ExtraEnvVars["FELIX_FLOWLOGSENABLEHOSTENDPOINT"] = "true"
		opts.ExtraEnvVars["FELIX_FLOWLOGSENABLENETWORKSETS"] = "true"
		opts.ExtraEnvVars["FELIX_FLOWLOGSFILEINCLUDELABELS"] = "true"
		opts.ExtraEnvVars["FELIX_FLOWLOGSFILEINCLUDEPOLICIES"] = "true"
		opts.ExtraEnvVars["FELIX_FLOWLOGSFILEAGGREGATIONKINDFORALLOWED"] = strconv.Itoa(int(AggrNone))
		opts.ExtraEnvVars["FELIX_FLOWLOGSFILEAGGREGATIONKINDFORDENIED"] = strconv.Itoa(int(AggrNone))
		opts.ExtraEnvVars["FELIX_FLOWLOGSCOLLECTORDEBUGTRACE"] = "true"
		opts.ExtraEnvVars["FELIX_FLOWLOGSCOLLECTTCPSTATS"] = "true"
		opts.ExtraEnvVars["FELIX_FLOWLOGSCOLLECTPROCESSPATH"] = "true"
		opts.ExtraEnvVars["FELIX_FLOWLOGSCOLLECTPROCESSINFO"] = "true"
		opts.ExtraEnvVars["FELIX_BPFCONNTRACKTIMEOUTS"] = "TCPFinsSeen=30s"

		// Start felix instances.
		tc, _ = infrastructure.StartNNodeTopology(2, opts, infra)

		// Install a default profile that allows all ingress and egress, in the absence of any Policy.
		infra.AddDefaultAllow()

		// Create workload on host 1.
		ep1_1 = workload.Run(tc.Felixes[0], "ep1-1", "default", "10.65.0.0", "8055", "tcp")
		ep1_1.ConfigureInInfra(infra)

		// Create workload on host 2.
		ep1_2 = workload.Run(tc.Felixes[1], "ep1-2", "default", "10.65.1.0", "8055", "tcp")
		ep1_2.ConfigureInInfra(infra)

		flowLogsReaders = []flowlogs.FlowLogReader{}
		for _, f := range tc.Felixes {
			flowLogsReaders = append(flowLogsReaders, f)
		}
	})

	It("should have the correct process info and tcp stats.", func() {
		cc := &connectivity.Checker{}
		cc.ExpectSome(ep1_1, ep1_2)
		cc.CheckConnectivity()
		Eventually(func() error {
			flowTester := flowlogs.NewFlowTesterDeprecated(flowLogsReaders, true, true, 0)
			flogs := flowTester.GetFlows()
			if len(flogs) == 0 {
				return fmt.Errorf("Error reading flowlogs")
			}

			tcpStatsSrc := false
			tcpStatsDst := false
			for _, flowLog := range flogs {
				if flowLog.SrcMeta.Type != "wep" || flowLog.SrcMeta.Namespace != "default" || flowLog.SrcMeta.Name != ep1_1.Name {
					return fmt.Errorf("Unexpected source meta in flow: %#v", flowLog.SrcMeta)
				}
				if flowLog.Reporter == "src" && !tcpStatsSrc {
					if strings.Contains(flowLog.ProcessName, "test-connection") &&
						flowLog.ProcessID != "" && checkIfFlowLogHasTCPStats(flowLog) {
						tcpStatsSrc = true
					}
				} else if !tcpStatsDst {
					if strings.Contains(flowLog.ProcessName, "test-workload") &&
						flowLog.ProcessID != "" &&
						checkIfFlowLogHasTCPStats(flowLog) {
						tcpStatsDst = true
					}
				}
				if tcpStatsSrc && tcpStatsDst {
					return nil
				}
			}
			return fmt.Errorf("Process Info and tcp stats not proper in flows")
		}, "30s", "3s").ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			for _, felix := range tc.Felixes {
				logNFTDiags(felix)
				felix.Exec("iptables-save", "-c")
				felix.Exec("ipset", "list")
				felix.Exec("ip", "r")
				felix.Exec("ip", "a")
			}
		}

		ep1_1.Stop()
		ep1_2.Stop()
		for _, felix := range tc.Felixes {
			if bpfEnabled {
				felix.Exec("calico-bpf", "connect-time", "clean")
			}
			felix.Stop()
		}

		if CurrentGinkgoTestDescription().Failed {
			infra.DumpErrorData()
		}
		infra.Stop()
	})
})

func checkIfFlowLogHasTCPStats(flowLog flowlog.FlowLog) bool {
	return flowLog.SendCongestionWnd.Mean != 0 && flowLog.SendCongestionWnd.Min != 0 &&
		flowLog.SmoothRtt.Mean != 0 && flowLog.SmoothRtt.Max != 0 &&
		flowLog.MinRtt.Mean != 0 && flowLog.MinRtt.Max != 0 &&
		flowLog.Mss.Mean != 0 && flowLog.Mss.Min != 0
}
