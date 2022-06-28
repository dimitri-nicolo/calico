// Copyright (c) 2017-2022 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package infrastructure

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"

	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/fv/containers"
	"github.com/projectcalico/calico/felix/fv/tcpdump"
	"github.com/projectcalico/calico/felix/fv/utils"
)

var atomicCounter uint32

var cwLogDir = os.Getenv("FV_CWLOGDIR")

// FIXME: isolate individual Felix instances in their own cgroups.  Unfortunately, this doesn't work on systems that are using cgroupv1
// see https://elixir.bootlin.com/linux/v5.3.11/source/include/linux/cgroup-defs.h#L788 for explanation.
const CreateCgroupV2 = false

type Felix struct {
	*containers.Container

	// ExpectedIPIPTunnelAddr contains the IP that the infrastructure expects to
	// get assigned to the IPIP tunnel.  Filled in by AddNode().
	ExpectedIPIPTunnelAddr string
	// ExpectedVXLANTunnelAddr contains the IP that the infrastructure expects to
	// get assigned to the IPv4 VXLAN tunnel.  Filled in by AddNode().
	ExpectedVXLANTunnelAddr string
	// ExpectedVXLANV6TunnelAddr contains the IP that the infrastructure expects to
	// get assigned to the IPv6 VXLAN tunnel.  Filled in by AddNode().
	ExpectedVXLANV6TunnelAddr string
	// ExpectedWireguardTunnelAddr contains the IP that the infrastructure expects to
	// get assigned to the Wireguard tunnel.  Filled in by AddNode().
	ExpectedWireguardTunnelAddr string

	// IP of the Typha that this Felix is using (if any).
	TyphaIP string

	// If sets, acts like an external IP of a node. Filled in by AddNode().
	// XXX setup routes
	ExternalIP string

	startupDelayed   bool
	cwlCallsExpected bool
	cwlFile          string
	cwlGroupName     string
	cwlStreamName    string
	cwlRetentionDays int64
	uniqueName       string
}

func (f *Felix) GetFelixPID() int {
	if f.startupDelayed {
		log.Panic("GetFelixPID() called but startup is delayed")
	}
	return f.GetSinglePID("calico-felix")
}

func (f *Felix) GetFelixPIDs() []int {
	if f.startupDelayed {
		log.Panic("GetFelixPIDs() called but startup is delayed")
	}
	return f.GetPIDs("calico-felix")
}

func (f *Felix) TriggerDelayedStart() {
	if !f.startupDelayed {
		log.Panic("TriggerDelayedStart() called but startup wasn't delayed")
	}
	f.Exec("touch", "/start-trigger")
	f.startupDelayed = false
}

func (f *Felix) RunDebugConsoleCommand(commandAndArgs ...string) (string, error) {
	f.EnsureBinary("run-debug-console-command")
	return f.ExecCombinedOutput(append([]string{"/run-debug-console-command"}, commandAndArgs...)...)

}

func RunFelix(infra DatastoreInfra, id int, options TopologyOptions) *Felix {
	log.Info("Starting felix")
	ipv6Enabled := fmt.Sprint(options.EnableIPv6)
	bpfEnableIPv6 := fmt.Sprint(options.BPFEnableIPv6)

	args := infra.GetDockerArgs()
	args = append(args, "--privileged")

	// Collect the environment variables for starting this particular container.  Note: we
	// are called concurrently with other instances of RunFelix so it's important to only
	// read from options.*.
	envVars := map[string]string{
		// Enable core dumps.
		"GOTRACEBACK": "crash",
		"GORACE":      "history_size=2",
		// Tell the wrapper to set the core file name pattern so we can find the dump.
		"SET_CORE_PATTERN": "true",

		"FELIX_LOGSEVERITYSCREEN":         options.FelixLogSeverity,
		"FELIX_PROMETHEUSMETRICSENABLED":  "true",
		"FELIX_PROMETHEUSREPORTERENABLED": "true",
		"FELIX_BPFLOGLEVEL":               "debug",
		"FELIX_USAGEREPORTINGENABLED":     "false",
		"FELIX_IPV6SUPPORT":               ipv6Enabled,
		"FELIX_BPFIPV6SUPPORT":            bpfEnableIPv6,
		// Disable log dropping, because it can cause flakes in tests that look for particular logs.
		"FELIX_DEBUGDISABLELOGDROPPING": "true",
	}
	// Collect the volumes for this container.
	wd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred(), "failed to get working directory")
	fvBin := os.Getenv("FV_BINARY")
	if fvBin == "" {
		fvBin = "bin/calico-felix-amd64"
	}
	volumes := map[string]string{
		path.Join(wd, "..", "bin"):        "/usr/local/bin",
		path.Join(wd, "..", fvBin):        "/usr/local/bin/calico-felix",
		path.Join(wd, "..", "bin", "bpf"): "/usr/lib/calico/bpf/",
		"/lib/modules":                    "/lib/modules",
		"/tmp":                            "/tmp",
	}

	containerName := containers.UniqueName(fmt.Sprintf("felix-%d", id))

	if os.Getenv("FELIX_FV_ENABLE_BPF") == "true" {
		if !options.TestManagesBPF {
			log.Info("FELIX_FV_ENABLE_BPF=true, enabling BPF with env var")
			envVars["FELIX_BPFENABLED"] = "true"
		} else {
			log.Info("FELIX_FV_ENABLE_BPF=true but test manages BPF state itself, not using env var")
		}

		if CreateCgroupV2 {
			envVars["FELIX_DEBUGBPFCGROUPV2"] = containerName
		}
	}

	// For FV, tell Felix to write CloudWatch logs to a file instead of to the real
	// AWS API.  Whether logs are actually generated, at all, still depends on
	// FELIX_CLOUDWATCHLOGSREPORTERENABLED; tests that want that should call
	// EnableCloudWatchLogs().
	uniqueName := fmt.Sprintf("%d-%d-%d", id, os.Getpid(), int(atomic.AddUint32(&atomicCounter, 1)))
	cwlFile := "cwl-" + uniqueName + "-felixfv.txt"
	envVars["FELIX_DEBUGCLOUDWATCHLOGSFILE"] = "/cwlogs/" + cwlFile
	volumes[cwLogDir] = "/cwlogs"

	cwlCallsExpected := false
	cwlGroupName := "tigera-flowlogs-<cluster-guid>"
	cwlStreamName := "<felix-hostname>_Flowlogs"
	cwlRetentionDays := int64(7)

	// It's fine to always create the directory for felix flow logs, if they
	// aren't enabled the directory will just stay empty.
	logDir := path.Join(cwLogDir, uniqueName)
	os.MkdirAll(logDir, 0777)
	args = append(args, "-v", logDir+":/var/log/calico/flowlogs")

	for k, v := range options.ExtraEnvVars {
		envVars[k] = v
	}

	if options.WithPrometheusPortTLS {
		EnsureTLSCredentials()
		envVars[CertDir] = CertDir
		envVars["FELIX_PROMETHEUSREPORTERCAFILE"] = filepath.Join(CertDir, "ca.crt")
		envVars["FELIX_PROMETHEUSREPORTERKEYFILE"] = filepath.Join(CertDir, "server.key")
		envVars["FELIX_PROMETHEUSREPORTERCERTFILE"] = filepath.Join(CertDir, "server.crt")
		envVars["FELIX_PROMETHEUSMETRICSCAFILE"] = filepath.Join(CertDir, "ca.crt")
		envVars["FELIX_PROMETHEUSMETRICSKEYFILE"] = filepath.Join(CertDir, "server.key")
		envVars["FELIX_PROMETHEUSMETRICSCERTFILE"] = filepath.Join(CertDir, "server.crt")
	}

	if options.DelayFelixStart {
		envVars["DELAY_FELIX_START"] = "true"
	}

	for k, v := range envVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add in the volumes.
	for k, v := range options.ExtraVolumes {
		volumes[k] = v
	}
	if id < len(options.PerNodeOptions) {
		for k, v := range options.PerNodeOptions[id].ExtraVolumes {
			volumes[k] = v
		}
	}
	for k, v := range volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", k, v))
	}

	args = append(args,
		utils.Config.FelixImage,
	)

	felixOpts := containers.RunOpts{
		AutoRemove: true,
	}
	if options.FelixStopGraceful {
		// Leave StopSignal defaulting to SIGTERM, and allow 10 seconds for Felix
		// to handle that gracefully.
		felixOpts.StopTimeoutSecs = 10
	} else {
		// Use SIGKILL to stop Felix immediately.
		felixOpts.StopSignal = "SIGKILL"
	}
	c := containers.RunWithFixedName(containerName, felixOpts, args...)

	if options.EnableIPv6 {
		c.Exec("sysctl", "-w", "net.ipv6.conf.all.disable_ipv6=0")
		c.Exec("sysctl", "-w", "net.ipv6.conf.default.disable_ipv6=0")
		c.Exec("sysctl", "-w", "net.ipv6.conf.lo.disable_ipv6=0")
		c.Exec("sysctl", "-w", "net.ipv6.conf.all.forwarding=1")
	} else {
		c.Exec("sysctl", "-w", "net.ipv6.conf.all.disable_ipv6=1")
		c.Exec("sysctl", "-w", "net.ipv6.conf.default.disable_ipv6=1")
		c.Exec("sysctl", "-w", "net.ipv6.conf.lo.disable_ipv6=1")
		c.Exec("sysctl", "-w", "net.ipv6.conf.all.forwarding=0")
	}

	// Configure our model host to drop forwarded traffic by default.  Modern
	// Kubernetes/Docker hosts now have this setting, and the consequence is that
	// whenever Calico policy intends to allow a packet, it must explicitly ACCEPT
	// that packet, not just allow it to pass through cali-FORWARD and assume it will
	// be accepted by the rest of the chain.  Establishing that setting in this FV
	// allows us to test that.
	c.Exec("iptables",
		"-w", "10", // Retry this for 10 seconds, e.g. if something else is holding the lock
		"-W", "100000", // How often to probe the lock in microsecs.
		"-P", "FORWARD", "DROP")

	return &Felix{
		Container:        c,
		startupDelayed:   options.DelayFelixStart,
		cwlFile:          cwlFile,
		cwlCallsExpected: cwlCallsExpected,
		cwlGroupName: strings.Replace(
			cwlGroupName,
			"<cluster-guid>",
			infra.GetClusterGUID(),
			1,
		),
		cwlStreamName: strings.Replace(
			cwlStreamName,
			"<felix-hostname>",
			c.Name,
			1,
		),
		cwlRetentionDays: cwlRetentionDays,
		uniqueName:       uniqueName,
	}
}

func (f *Felix) Stop() {
	if CreateCgroupV2 {
		_ = f.ExecMayFail("rmdir", path.Join("/run/calico/cgroup/", f.Name))
	}
	f.Container.Stop()
	if f.cwlCallsExpected {
		Expect(cwLogDir + "/" + f.cwlFile).To(BeAnExistingFile())
	} else {
		Expect(cwLogDir + "/" + f.cwlFile).NotTo(BeAnExistingFile())
	}
}

func (f *Felix) Restart() {
	oldPID := f.GetFelixPID()
	f.Exec("kill", "-HUP", fmt.Sprint(oldPID))
	Eventually(f.GetFelixPID, "10s", "100ms").ShouldNot(Equal(oldPID))
}

// AttachTCPDump returns tcpdump attached to the container
func (f *Felix) AttachTCPDump(iface string) *tcpdump.TCPDump {
	return tcpdump.Attach(f.Container.Name, "", iface)
}

func (f *Felix) ProgramIptablesDNAT(serviceIP, targetIP, chain string) {
	f.Exec(
		"iptables",
		"-w", "10", // Retry this for 10 seconds, e.g. if something else is holding the lock
		"-W", "100000", // How often to probe the lock in microsecs.
		"-t", "nat", "-A", chain,
		"--destination", serviceIP,
		"-j", "DNAT", "--to-destination", targetIP,
	)
}

func (f *Felix) FlowLogDir() string {
	return path.Join(cwLogDir, f.uniqueName)
}
