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

package utils

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"regexp"
	"strconv"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/ipsets"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/selector"
)

type EnvConfig struct {
	FelixImage   string `default:"tigera/felix:latest"`
	EtcdImage    string `default:"quay.io/coreos/etcd"`
	K8sImage     string `default:"gcr.io/google_containers/hyperkube-amd64:v1.10.4"`
	TyphaImage   string `default:"tigera/typha:latest"` // Note: this is overridden in the Makefile!
	BusyboxImage string `default:"busybox:latest"`
}

var Config EnvConfig

func init() {
	err := envconfig.Process("fv", &Config)
	if err != nil {
		panic(err)
	}
	log.WithField("config", Config).Info("Loaded config")
}

var Ctx = context.Background()

var NoOptions = options.SetOptions{}

func Run(command string, args ...string) {
	_ = run(true, command, args...)
}

func RunMayFail(command string, args ...string) error {
	return run(false, command, args...)
}

var currentTestOutput = []string{}

var LastRunOutput string

func run(checkNoError bool, command string, args ...string) error {
	outputBytes, err := Command(command, args...).CombinedOutput()
	currentTestOutput = append(currentTestOutput, fmt.Sprintf("Command: %v %v\n", command, args))
	currentTestOutput = append(currentTestOutput, string(outputBytes))
	LastRunOutput = string(outputBytes)
	if err != nil {
		log.WithFields(log.Fields{
			"command": command,
			"args":    args,
			"output":  string(outputBytes)}).WithError(err).Warning("Command failed")
	}
	if checkNoError {
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Command failed\nCommand: %v args: %v\nOutput:\n\n%v",
			command, args, string(outputBytes)))
	}
	return err
}

func AddToTestOutput(args ...string) {
	currentTestOutput = append(currentTestOutput, args...)
}

var _ = BeforeEach(func() {
	currentTestOutput = []string{}
})

var _ = AfterEach(func() {
	if CurrentGinkgoTestDescription().Failed {
		os.Stdout.WriteString("\n===== begin output from failed test =====\n")
		for _, output := range currentTestOutput {
			os.Stdout.WriteString(output)
		}
		os.Stdout.WriteString("===== end output from failed test =====\n\n")
	}
})

func GetCommandOutput(command string, args ...string) (string, error) {
	cmd := Command(command, args...)
	log.Infof("Running '%s %s'", cmd.Path, strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func RunCommand(command string, args ...string) error {
	output, err := GetCommandOutput(command, args...)
	log.Infof("output: %v", output)
	return err
}

func Command(name string, args ...string) *exec.Cmd {
	log.WithFields(log.Fields{
		"command":     name,
		"commandArgs": args,
	}).Info("Creating Command.")

	return exec.Command(name, args...)
}

func LogOutput(cmd *exec.Cmd, name string) error {
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Getting StdoutPipe failed for %s: %v", name, err)
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Getting StderrPipe failed for %s: %v", name, err)
	}
	stdoutReader := bufio.NewReader(outPipe)
	stderrReader := bufio.NewReader(errPipe)
	go func() {
		for {
			line, err := stdoutReader.ReadString('\n')
			if err != nil {
				log.WithError(err).Infof("End of %s stdout", name)
				return
			}
			log.Infof("%s stdout: %s", name, strings.TrimSpace(string(line)))
		}
	}()
	go func() {
		for {
			line, err := stderrReader.ReadString('\n')
			if err != nil {
				log.WithError(err).Infof("End of %s stderr", name)
				return
			}
			log.Infof("%s stderr: %s", name, strings.TrimSpace(string(line)))
		}
	}()
	return nil
}

func GetEtcdClient(etcdIP string) client.Interface {
	client, err := client.New(apiconfig.CalicoAPIConfig{
		Spec: apiconfig.CalicoAPIConfigSpec{
			DatastoreType: apiconfig.EtcdV3,
			EtcdConfig: apiconfig.EtcdConfig{
				EtcdEndpoints: "http://" + etcdIP + ":2379",
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())
	return client
}

func IPSetIDForSelector(rawSelector string) string {
	sel, err := selector.Parse(rawSelector)
	Expect(err).ToNot(HaveOccurred())

	ipSetData := calc.IPSetData{
		Selector: sel,
	}
	setID := ipSetData.UniqueID()
	return setID
}

func IPSetNameForSelector(ipVersion int, rawSelector string) string {
	setID := IPSetIDForSelector(rawSelector)
	var ipFamily ipsets.IPFamily
	if ipVersion == 4 {
		ipFamily = ipsets.IPFamilyV4
	} else {
		ipFamily = ipsets.IPFamilyV6
	}
	ipVerConf := ipsets.NewIPVersionConfig(
		ipFamily,
		rules.IPSetNamePrefix,
		nil,
		nil,
	)

	return ipVerConf.NameForMainIPSet(setID)
}

// Run a connection test command.
// Report if connection test is successful and packet loss string for packet loss test.
func RunConnectionCmd(connectionCmd *exec.Cmd) (bool, string) {
	outPipe, err := connectionCmd.StdoutPipe()
	Expect(err).NotTo(HaveOccurred())
	errPipe, err := connectionCmd.StderrPipe()
	Expect(err).NotTo(HaveOccurred())
	err = connectionCmd.Start()
	Expect(err).NotTo(HaveOccurred())

	wOut, err := ioutil.ReadAll(outPipe)
	Expect(err).NotTo(HaveOccurred())
	wErr, err := ioutil.ReadAll(errPipe)
	Expect(err).NotTo(HaveOccurred())
	err = connectionCmd.Wait()

	log.WithFields(log.Fields{
		"stdout": string(wOut),
		"stderr": string(wErr)}).WithError(err).Info("Connection test")

	return (err == nil), extractPacketStatString(string(wErr))
}

const ConnectionTypeStream = "stream"
const ConnectionTypePing = "ping"

type ConnConfig struct {
	ConnType string
	ConnID   string
}

func (cc ConnConfig) getTestMessagePrefix() string {
	return cc.ConnType + ":" + cc.ConnID + "~"
}

// Assembly a test message.
func (cc ConnConfig) GetTestMessage(sequence int) string {
	return cc.getTestMessagePrefix() + fmt.Sprintf("%d", sequence)
}

// Extract sequence number from test message.
func (cc ConnConfig) GetTestMessageSequence(msg string) (int, error) {
	msg = strings.TrimSpace(msg)
	seqString := strings.TrimPrefix(msg, cc.getTestMessagePrefix())
	if seqString == msg {
		// TrimPrefix failed.
		return 0, errors.New("invalid message prefix format:" + msg)
	}

	seq, err := strconv.Atoi(seqString)
	if err != nil || seq < 0 {
		return 0, errors.New("invalid message sequence format:" + msg)
	}
	return seq, nil
}

func IsMessagePartOfStream(msg string) bool {
	return strings.HasPrefix(strings.TrimSpace(msg), ConnectionTypeStream)
}

const (
	PacketLossPrefix        = "PacketLoss"
	PacketLossPercentPrefix = PacketLossPrefix + "Percent"
	PacketLossNumberPrefix  = PacketLossPrefix + "Number"
	PacketTotalReqPrefix    = "TotalReq"
	PacketTotalReplyPrefix  = "TotalReply"
)

// extract packet stat string from an output.
func extractPacketStatString(s string) string {
	re := regexp.MustCompile(PacketTotalReqPrefix + `<\d+>` + "," + PacketTotalReplyPrefix + `<\d+>`)
	stat := re.FindString(s)

	return stat
}

func FormPacketStatString(totalReq, totalReply int) string {
	return fmt.Sprintf("%s<%d>,%s<%d>", PacketTotalReqPrefix, totalReq, PacketTotalReplyPrefix, totalReply)
}

// extract one packet loss measurement from string. Return -1 if measurement not found.
func extractPacketLoss(prefix string, s string) int {
	var number int
	re := regexp.MustCompile(prefix + `<\d+>`)
	lossString := re.FindString(s)

	re = regexp.MustCompile(`\d+`)
	substring := re.FindString(lossString)
	if substring != "" {
		var err error
		number, err = strconv.Atoi(substring)
		Expect(err).NotTo(HaveOccurred())
	} else {
		number = -1
	}

	return number
}

// Form a packet loss string from a maxPercent and maxNumber.
func FormPacketLossString(maxPercent, maxNumber int) string {
	var ps, ns string
	if maxPercent >= 0 {
		ps = fmt.Sprintf("%s<%d>", PacketLossPercentPrefix, maxPercent)
	}
	if maxNumber >= 0 {
		ns = fmt.Sprintf("%s<%d>", PacketLossNumberPrefix, maxNumber)
	}

	return fmt.Sprintf("%s%s", ps, ns)
}

func GetPacketLossDirect(s string) (int, int) {
	return extractPacketLoss(PacketLossPercentPrefix, s), extractPacketLoss(PacketLossNumberPrefix, s)
}

func extractPacketNumbers(s string) (int, int) {
	re := regexp.MustCompile(`\d+`)
	numbers := re.FindAllString(extractPacketStatString(s), -1)
	Expect(len(numbers)).To(Equal(2))

	totalReq, err := strconv.Atoi(numbers[0])
	Expect(err).NotTo(HaveOccurred())

	totalReply, err := strconv.Atoi(numbers[1])
	Expect(err).NotTo(HaveOccurred())

	return totalReq, totalReply
}

func GetPacketLossFromStat(s string) (int, int) {
	totalReq, totalReply := extractPacketNumbers(s)
	diff := totalReq - totalReply
	Expect(diff).To(BeNumerically(">=", 0))

	// Calculate packet loss and print out result.
	loss := float64(diff) / float64(totalReq) * 100
	if loss > 0 && uint(loss) == 0 {
		// Set minimal loss to 1 percent.
		return 1, diff
	} else {
		return int(loss), diff
	}
}
