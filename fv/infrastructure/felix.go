// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.
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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"path"

	"encoding/json"

	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/utils"
)

var cwLogDir = os.Getenv("FV_CWLOGDIR")

type Felix struct {
	*containers.Container

	// ExpectedIPIPTunnelAddr contains the IP that the infrastructure expects to
	// get assigned to the IPIP tunnel.  Filled in by AddNode().
	ExpectedIPIPTunnelAddr string
	// ExpectedVXLANTunnelAddr contains the IP that the infrastructure expects to
	// get assigned to the VXLAN tunnel.  Filled in by AddNode().
	ExpectedVXLANTunnelAddr string

	// IP of the Typha that this Felix is using (if any).
	TyphaIP string

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

var (
	ErrNoCloudwatchLogs = errors.New("No logs yet")
)

func (f *Felix) ReadFlowLogs(output string) ([]collector.FlowLog, error) {
	switch output {
	case "cloudwatch":
		return f.ReadCloudWatchLogs()
	case "file":
		return f.ReadFlowLogsFile()
	default:
		panic("unrecognized flow log output")
	}
}

func (f *Felix) ReadFlowLogsFile() ([]collector.FlowLog, error) {
	var flowLogs []collector.FlowLog
	logDir := path.Join(cwLogDir, f.uniqueName)
	log.WithField("dir", logDir).Info("Reading Flow Logs from file")
	logFile, err := os.Open(path.Join(logDir, collector.FlowLogFilename))
	if err != nil {
		return flowLogs, err
	}
	defer logFile.Close()

	s := bufio.NewScanner(logFile)
	for s.Scan() {
		var fljo collector.FlowLogJSONOutput
		err = json.Unmarshal(s.Bytes(), &fljo)
		if err != nil {
			return flowLogs, err
		}
		fl, err := fljo.ToFlowLog()
		if err != nil {
			return flowLogs, err
		}
		flowLogs = append(flowLogs, fl)
	}
	return flowLogs, nil
}

func (f *Felix) ReadCloudWatchLogs() ([]collector.FlowLog, error) {
	log.Infof("Read CloudWatchLogs file %v", cwLogDir+"/"+f.cwlFile)

	file, err := os.Open(cwLogDir + "/" + f.cwlFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	retentionDays := make(map[string]int64)
	logs := make(map[string][]collector.FlowLog)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "PutRetentionPolicy") {
			// Next line is LogGroupName: "<name>".
			scanner.Scan()
			lgName := strings.Split(scanner.Text(), "\"")[1]
			// Next line is RetentionInDays: <int>.
			scanner.Scan()
			days, err := strconv.ParseInt(strings.Split(strings.TrimSpace(scanner.Text()), " ")[1], 10, 64)
			Expect(err).NotTo(HaveOccurred())
			// Store this policy.
			retentionDays[lgName] = days
		} else if strings.Contains(line, "PutLogEvents") {
			var events []collector.FlowLog
			message := ""
			groupName := ""
			streamName := ""
			// Read until we see a line that is just "}", and we've seen the
			// group and stream names.
			for scanner.Scan() {
				line = strings.TrimSpace(scanner.Text())
				if strings.Contains(line, "Message: \"") {
					// Replace escaped double quotes, in the flow log
					// message string, with single quotes.  (So as not
					// to confuse the following line, which relies on
					// double quotes to identify the complete
					// message.)
					line = strings.Replace(line, "\\\"", "'", -1)
					message = strings.Split(line, "\"")[1]
				} else if strings.Contains(line, "Timestamp: ") {
					_, err := strconv.ParseInt(strings.Split(line, " ")[1], 10, 64)
					Expect(err).NotTo(HaveOccurred())
					// Parse the log message.
					fl := collector.FlowLog{}
					fl.Deserialize(message)
					events = append(events, fl)
				} else if strings.Contains(line, "LogGroupName: \"") {
					groupName = strings.Split(line, "\"")[1]
				} else if strings.Contains(line, "LogStreamName: \"") {
					streamName = strings.Split(line, "\"")[1]
				} else if line == "}" && groupName != "" && streamName != "" {
					// Store these logs.
					key := groupName + "/" + streamName
					previousEvents, ok := logs[key]
					if ok {
						logs[key] = append(previousEvents, events...)
					} else {
						logs[key] = events
					}
					break
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	log.WithFields(log.Fields{"retentionDays": retentionDays, "logs": logs}).Info("Data read")

	if len(logs) == 0 {
		return nil, ErrNoCloudwatchLogs
	}

	Expect(retentionDays).To(HaveLen(1))
	for group, days := range retentionDays {
		Expect(group).To(Equal(f.cwlGroupName))
		Expect(days).To(Equal(f.cwlRetentionDays))
	}

	Expect(logs).To(HaveLen(1))
	for groupSlashStream, events := range logs {
		Expect(groupSlashStream).To(Equal(f.cwlGroupName + "/" + f.cwlStreamName))
		return events, nil
	}

	return nil, errors.New("Should never get here")
}

func RunFelix(infra DatastoreInfra, options TopologyOptions) *Felix {
	log.Info("Starting felix")
	ipv6Enabled := fmt.Sprint(options.EnableIPv6)

	args := infra.GetDockerArgs()
	args = append(args,
		"--privileged",
		"-e", "FELIX_LOGSEVERITYSCREEN="+options.FelixLogSeverity,
		"-e", "FELIX_PROMETHEUSMETRICSENABLED=true",
		"-e", "FELIX_PROMETHEUSREPORTERENABLED=true",
		"-e", "FELIX_USAGEREPORTINGENABLED=false",
		"-e", "FELIX_IPV6SUPPORT="+ipv6Enabled,
		// Disable log dropping, because it can cause flakes in tests that look
		// for particular logs.
		"-e", "FELIX_DEBUGDISABLELOGDROPPING=true",
		"-v", "/lib/modules:/lib/modules",
	)

	// For FV, tell Felix to write CloudWatch logs to a file instead of to the real
	// AWS API.  Whether logs are actually generated, at all, still depends on
	// FELIX_CLOUDWATCHLOGSREPORTERENABLED; tests that want that should call
	// EnableCloudWatchLogs().
	uniqueName := fmt.Sprintf("%d-%d", os.Getpid(), containers.NextContainerIndex())
	cwlFile := "cwl-" + uniqueName + "-felixfv.txt"
	args = append(args,
		"-e", "FELIX_DEBUGCLOUDWATCHLOGSFILE=/cwlogs/"+cwlFile,
		"-v", cwLogDir+":/cwlogs",
	)
	cwlCallsExpected := false
	cwlGroupName := "tigera-flowlogs-<cluster-guid>"
	cwlStreamName := "<felix-hostname>_Flowlogs"
	cwlRetentionDays := int64(7)
	if setting, ok := options.ExtraEnvVars["FELIX_CLOUDWATCHLOGSREPORTERENABLED"]; ok {
		switch setting {
		case "true", "1", "yes", "y", "t":
			cwlCallsExpected = true
		}
	}
	if setting, ok := options.ExtraEnvVars["FELIX_CLOUDWATCHLOGSLOGGROUPNAME"]; ok {
		cwlGroupName = setting
	}
	if setting, ok := options.ExtraEnvVars["FELIX_CLOUDWATCHLOGSLOGSTREAMNAME"]; ok {
		cwlStreamName = setting
	}
	if setting, ok := options.ExtraEnvVars["FELIX_CLOUDWATCHLOGSRETENTIONDAYS"]; ok {
		var err error
		cwlRetentionDays, err = strconv.ParseInt(setting, 10, 64)
		Expect(err).NotTo(HaveOccurred())
	}

	// It's fine to always create the directory for felix flow logs, if they
	// aren't enabled the directory will just stay empty.
	logDir := path.Join(cwLogDir, uniqueName)
	os.MkdirAll(logDir, 0777)
	args = append(args, "-v", logDir+":/var/log/calico/flowlogs")

	if options.WithPrometheusPortTLS {
		EnsureTLSCredentials()
		options.ExtraVolumes[CertDir] = CertDir
		options.ExtraEnvVars["FELIX_PROMETHEUSREPORTERCAFILE"] = filepath.Join(CertDir, "ca.crt")
		options.ExtraEnvVars["FELIX_PROMETHEUSREPORTERKEYFILE"] = filepath.Join(CertDir, "server.key")
		options.ExtraEnvVars["FELIX_PROMETHEUSREPORTERCERTFILE"] = filepath.Join(CertDir, "server.crt")
		options.ExtraEnvVars["FELIX_PROMETHEUSMETRICSCAFILE"] = filepath.Join(CertDir, "ca.crt")
		options.ExtraEnvVars["FELIX_PROMETHEUSMETRICSKEYFILE"] = filepath.Join(CertDir, "server.key")
		options.ExtraEnvVars["FELIX_PROMETHEUSMETRICSCERTFILE"] = filepath.Join(CertDir, "server.crt")
	}

	if options.DelayFelixStart {
		args = append(args, "-e", "DELAY_FELIX_START=true")
	}

	for k, v := range options.ExtraEnvVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range options.ExtraVolumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", k, v))
	}

	args = append(args,
		utils.Config.FelixImage,
	)

	c := containers.Run("felix",
		containers.RunOpts{AutoRemove: true},
		args...,
	)

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
	f.Container.Stop()
	if f.cwlCallsExpected {
		Expect(cwLogDir + "/" + f.cwlFile).To(BeAnExistingFile())
	} else {
		Expect(cwLogDir + "/" + f.cwlFile).NotTo(BeAnExistingFile())
	}
}

func (f *Felix) Restart() {
	oldPID := f.GetFelixPID()
	f.Signal(syscall.SIGHUP)
	Eventually(f.GetFelixPID, "10s", "100ms").ShouldNot(Equal(oldPID))
}
