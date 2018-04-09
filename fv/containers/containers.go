// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.
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

package containers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/utils"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type Container struct {
	Name     string
	IP       string
	Hostname string
	runCmd   *exec.Cmd

	mutex    sync.Mutex
	binaries set.Set

	logFinished sync.WaitGroup
}

var containerIdx = 0

func (c *Container) Stop() {
	if c == nil {
		log.Info("Stop no-op because nil container")
		return
	}

	logCxt := log.WithField("container", c.Name)
	c.mutex.Lock()
	if c.runCmd == nil {
		logCxt.Info("Stop no-op because container is not running")
		c.mutex.Unlock()
		return
	}
	c.mutex.Unlock()

	logCxt.Info("Stop")

	// Ask docker to stop the container.
	withTimeoutPanic(logCxt, 30*time.Second, c.execDockerStop)
	// Shut down the docker run process (if needed).
	withTimeoutPanic(logCxt, 5*time.Second, func() { c.signalDockerRun(os.Interrupt) })

	// Wait for the container to exit, then escalate to killing it.
	startTime := time.Now()
	for {
		if !c.ListedInDockerPS() {
			// Container has stopped.  Mkae sure the docker CLI command is dead (it should be already)
			// and wait for its log.
			logCxt.Info("Container stopped (no longer listed in 'docker ps')")
			withTimeoutPanic(logCxt, 5*time.Second, func() { c.signalDockerRun(os.Kill) })
			withTimeoutPanic(logCxt, 10*time.Second, func() { c.logFinished.Wait() })
			return
		}
		if time.Since(startTime) > 2*time.Second {
			logCxt.Info("Container didn't stop, asking docker to kill it")
			// `docker kill` asks the docker daemon to kill the container but, on a
			// resource constrained system, we've seen that fail because the CLI command
			// was blocked so we kill the CLI command too.
			err := exec.Command("docker", "kill", c.Name).Run()
			logCxt.WithError(err).Info("Ran 'docker kill'")
			withTimeoutPanic(logCxt, 5*time.Second, func() { c.signalDockerRun(os.Kill) })
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	c.WaitNotRunning(60 * time.Second)
	logCxt.Info("Container stopped")
	withTimeoutPanic(logCxt, 5*time.Second, func() { c.signalDockerRun(os.Kill) })
	withTimeoutPanic(logCxt, 10*time.Second, func() { c.logFinished.Wait() })
}

func withTimeoutPanic(logCxt *log.Entry, t time.Duration, f func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()

	select {
	case <-done:
		return
	case <-time.After(t):
		logCxt.Panic("Timeout!")
	}
}

func (c *Container) execDockerStop() {
	logCxt := log.WithField("container", c.Name)
	logCxt.Info("Executing 'docker stop'")
	cmd := exec.Command("docker", "stop", c.Name)
	err := cmd.Run()
	if err != nil {
		logCxt.WithError(err).WithField("cmd", cmd).Error("docker stop command failed")
		return
	}
	logCxt.Info("'docker stop' returned success")
}

func (c *Container) signalDockerRun(sig os.Signal) {
	logCxt := log.WithFields(log.Fields{
		"container": c.Name,
		"signal":    sig,
	})
	logCxt.Info("Sending signal to 'docker run' process")
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.runCmd == nil {
		return
	}
	c.runCmd.Process.Signal(sig)
	logCxt.Info("Signalled docker run")
}

type RunOpts struct {
	AutoRemove bool
}

func Run(namePrefix string, opts RunOpts, args ...string) (c *Container) {

	// Build unique container name and struct.
	containerIdx++
	c = &Container{Name: fmt.Sprintf("%v-%d-%d-felixfv", namePrefix, os.Getpid(), containerIdx)}

	// Prep command to run the container.
	log.WithField("container", c).Info("About to run container")
	runArgs := []string{"run", "--name", c.Name, "--hostname", c.Name}

	if opts.AutoRemove {
		runArgs = append(runArgs, "--rm")
	}

	// Add remaining args
	runArgs = append(runArgs, args...)

	c.runCmd = utils.Command("docker", runArgs...)

	// Get the command's output pipes, so we can merge those into the test's own logging.
	stdout, err := c.runCmd.StdoutPipe()
	Expect(err).NotTo(HaveOccurred())
	stderr, err := c.runCmd.StderrPipe()
	Expect(err).NotTo(HaveOccurred())

	// Start the container running.
	err = c.runCmd.Start()
	Expect(err).NotTo(HaveOccurred())

	// Merge container's output into our own logging.
	c.logFinished.Add(2)
	go copyOutputToLog(c.Name, "stdout", stdout, &c.logFinished)
	go copyOutputToLog(c.Name, "stderr", stderr, &c.logFinished)

	// Note: it might take a long time for the container to start running, e.g. if the image
	// needs to be downloaded.
	c.WaitUntilRunning()

	// Fill in rest of container struct.
	c.IP = c.GetIP()
	c.Hostname = c.GetHostname()
	c.binaries = set.New()
	log.WithField("container", c).Info("Container now running")
	return
}

// Start executes "docker start" on a container. Useful when used after Stop()
// to restart a container.
func (c *Container) Start() {
	c.runCmd = utils.Command("docker", "start", "--attach", c.Name)

	stdout, err := c.runCmd.StdoutPipe()
	Expect(err).NotTo(HaveOccurred())
	stderr, err := c.runCmd.StderrPipe()
	Expect(err).NotTo(HaveOccurred())

	// Start the container running.
	err = c.runCmd.Start()
	Expect(err).NotTo(HaveOccurred())

	// Merge container's output into our own logging.
	c.logFinished.Add(2)
	go copyOutputToLog(c.Name, "stdout", stdout, &c.logFinished)
	go copyOutputToLog(c.Name, "stderr", stderr, &c.logFinished)

	c.WaitUntilRunning()

	log.WithField("container", c).Info("Container now running")
}

// Remove deletes a container. Should be manually called after a non-auto-removed container
// is stopped.
func (c *Container) Remove() {
	c.runCmd = utils.Command("docker", "rm", "-f", c.Name)
	err := c.runCmd.Start()
	Expect(err).NotTo(HaveOccurred())

	log.WithField("container", c).Info("Removed container.")
}

func copyOutputToLog(name string, streamName string, stream io.Reader, done *sync.WaitGroup) {
	defer done.Done()
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(nil, 10*1024*1024) // Increase maximum buffer size (but don't pre-alloc).
	for scanner.Scan() {
		log.Info(name, "[", streamName, "] ", scanner.Text())
	}
	logCxt := log.WithFields(log.Fields{
		"name":   name,
		"stream": stream,
	})
	if scanner.Err() != nil {
		logCxt.WithError(scanner.Err()).Error("Non-EOF error reading container stream")
	}
	logCxt.Info("Stream finished")
}

func (c *Container) DockerInspect(format string) string {
	inspectCmd := utils.Command("docker", "inspect",
		"--format="+format,
		c.Name,
	)
	outputBytes, err := inspectCmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	return string(outputBytes)
}

func (c *Container) GetIP() string {
	output := c.DockerInspect("{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}")
	return strings.TrimSpace(output)
}

func (c *Container) GetHostname() string {
	output := c.DockerInspect("{{.Config.Hostname}}")
	return strings.TrimSpace(output)
}

func (c *Container) GetPIDs(processName string) []int {
	out, err := c.ExecOutput("pgrep", fmt.Sprintf("^%s$", processName))
	if err != nil {
		log.WithError(err).Warn("pgrep failed, assuming no PIDs")
		return nil
	}
	var pids []int
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		Expect(err).NotTo(HaveOccurred())
		pids = append(pids, pid)
	}
	return pids
}

func (c *Container) GetSinglePID(processName string) int {
	// Get the process's PID.  This retry loop ensures that we don't get tripped up if we see multiple
	// PIDs, which can happen transiently when a process restarts/forks off a subprocess.
	start := time.Now()
	for {
		pids := c.GetPIDs(processName)
		if len(pids) == 1 {
			return pids[0]
		}
		Expect(time.Since(start)).To(BeNumerically("<", time.Second),
			"Timed out waiting for there to be a single PID")
		time.Sleep(50 * time.Millisecond)
	}
}

func (c *Container) WaitUntilRunning() {
	log.Info("Wait for container to be listed in docker ps")

	// Set up so we detect if container startup fails.
	stoppedChan := make(chan struct{})
	go func() {
		defer close(stoppedChan)
		err := c.runCmd.Wait()
		log.WithError(err).WithField("name", c.Name).Info("Container stopped ('docker run' exited)")
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.runCmd = nil
	}()

	for {
		Expect(stoppedChan).NotTo(BeClosed(), "Container failed before being listed in 'docker ps'")

		cmd := utils.Command("docker", "ps")
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
		if strings.Contains(string(out), c.Name) {
			break
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func (c *Container) Stopped() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.runCmd == nil
}

func (c *Container) ListedInDockerPS() bool {
	cmd := utils.Command("docker", "ps")
	out, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	return strings.Contains(string(out), c.Name)
}

func (c *Container) WaitNotRunning(timeout time.Duration) {
	log.Info("Wait for container not to be listed in docker ps")
	start := time.Now()
	for {
		if !c.ListedInDockerPS() {
			break
		}
		if time.Since(start) > timeout {
			log.Panic("Timed out waiting for container not to be listed.")
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func (c *Container) EnsureBinary(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.binaries.Contains(name) {
		utils.Command("docker", "cp", "../bin/"+name, c.Name+":/"+name).Run()
		c.binaries.Add(name)
	}
}

func (c *Container) CopyFileIntoContainer(hostPath, containerPath string) error {
	cmd := utils.Command("docker", "cp", hostPath, c.Name+":"+containerPath)
	return cmd.Run()
}

func (c *Container) Exec(cmd ...string) {
	log.WithField("container", c.Name).WithField("command", cmd).Info("Running command")
	arg := []string{"exec", c.Name}
	arg = append(arg, cmd...)
	utils.Run("docker", arg...)
}

func (c *Container) ExecMayFail(cmd ...string) error {
	arg := []string{"exec", c.Name}
	arg = append(arg, cmd...)
	return utils.RunMayFail("docker", arg...)
}

func (c *Container) ExecOutput(args ...string) (string, error) {
	arg := []string{"exec", c.Name}
	arg = append(arg, args...)
	cmd := exec.Command("docker", arg...)
	out, err := cmd.Output()
	if err != nil {
		if out == nil {
			return "", err
		}
		return string(out), err
	}
	return string(out), nil
}

func (c *Container) SourceName() string {
	return c.Name
}

func (c *Container) CanConnectTo(ip, port, protocol string) bool {

	// Ensure that the container has the 'test-connection' binary.
	c.EnsureBinary("test-connection")

	// Run 'test-connection' to the target.
	connectionCmd := utils.Command("docker", "exec", c.Name,
		"/test-connection", "--protocol="+protocol, "-", ip, port)
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

	return err == nil
}

func RunEtcd() *Container {
	log.Info("Starting etcd")
	return Run("etcd",
		RunOpts{AutoRemove: true},
		"--privileged", // So that we can add routes inside the etcd container,
		// when using the etcd container to model an external client connecting
		// into the cluster.
		utils.Config.EtcdImage,
		"etcd",
		"--advertise-client-urls", "http://127.0.0.1:2379",
		"--listen-client-urls", "http://0.0.0.0:2379")
}

type Felix struct {
	*Container
}

func (f *Felix) GetFelixPID() int {
	return f.GetSinglePID("calico-felix")
}

func (f *Felix) GetFelixPIDs() []int {
	return f.GetPIDs("calico-felix")
}

func RunFelix(etcdIP string, options TopologyOptions) *Felix {
	log.Info("Starting felix")
	ipv6Enabled := fmt.Sprint(options.EnableIPv6)

	args := []string{
		"--privileged",
		"-e", "CALICO_DATASTORE_TYPE=etcdv3",
		"-e", "CALICO_ETCD_ENDPOINTS=http://" + etcdIP + ":2379",
		"-e", "FELIX_LOGSEVERITYSCREEN=" + options.FelixLogSeverity,
		"-e", "FELIX_DATASTORETYPE=etcdv3",
		"-e", "FELIX_PROMETHEUSMETRICSENABLED=true",
		"-e", "FELIX_PROMETHEUSREPORTERENABLED=true",
		"-e", "FELIX_USAGEREPORTINGENABLED=false",
		"-e", "FELIX_IPV6SUPPORT=" + ipv6Enabled,
		"-v", "/lib/modules:/lib/modules",
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

	c := Run("felix",
		RunOpts{AutoRemove: true},
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

	return &Felix{
		Container: c,
	}
}

type TopologyOptions struct {
	FelixLogSeverity      string
	EnableIPv6            bool
	ExtraEnvVars          map[string]string
	ExtraVolumes          map[string]string
	AlphaFeaturesToEnable string
}

func DefaultTopologyOptions() TopologyOptions {
	return TopologyOptions{
		FelixLogSeverity: "info",
		EnableIPv6:       true,
		ExtraEnvVars:     map[string]string{},
		ExtraVolumes:     map[string]string{},
	}
}

// StartSingleNodeEtcdTopology starts an etcd container and a single Felix container; it initialises
// the datastore and installs a Node resource for the Felix node.
func StartSingleNodeEtcdTopology(options TopologyOptions) (felix *Felix, etcd *Container, calicoClient client.Interface) {
	felixes, etcd, calicoClient := StartNNodeEtcdTopology(1, options)
	felix = felixes[0]
	return
}

// StartNNodeEtcdTopology starts an etcd container and a set of Felix hosts.  If n > 1, sets
// up IPIP, otherwise this is skipped.
//
// - Configures an IPAM pool for 10.65.0.0/16 (so that Felix programs the all-IPAM blocks IP set)
//   but (for simplicity) we don't actually use IPAM to assign IPs.
// - Configures routes between the hosts, giving each host 10.65.x.0/24, where x is the
//   index in the returned array.  When creating workloads, use IPs from the relevant block.
// - Configures the Tunnel IP for each host as 10.65.x.1.
func StartNNodeEtcdTopology(n int, opts TopologyOptions) (felixes []*Felix, etcd *Container, client client.Interface) {
	log.Infof("Starting a %d-node etcd topology.", n)
	success := false
	var err error
	defer func() {
		if !success {
			log.WithError(err).Error("Failed to start topology, tearing down containers")
			for _, felix := range felixes {
				felix.Stop()
			}
			etcd.Stop()
		}
	}()

	// First start etcd.
	etcd = RunEtcd()

	// Connect to etcd.
	client = utils.GetEtcdClient(etcd.IP, opts.AlphaFeaturesToEnable)
	mustInitDatastore(client)

	// Install a CNX license key for testing purposes.
	addCNXLicense(client)

	if n > 1 {
		Eventually(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			ipPool := api.NewIPPool()
			ipPool.Name = "test-pool"
			ipPool.Spec.CIDR = "10.65.0.0/16"
			ipPool.Spec.IPIPMode = api.IPIPModeAlways
			_, err = client.IPPools().Create(ctx, ipPool, options.SetOptions{})
			return err
		}).ShouldNot(HaveOccurred())
	}

	for i := 0; i < n; i++ {
		// Then start Felix and create a node for it.
		felix := RunFelix(etcd.IP, opts)

		felixNode := api.NewNode()
		felixNode.Name = felix.Hostname
		if n > 1 {
			felixNode.Spec.BGP = &api.NodeBGPSpec{
				IPv4Address:        felix.IP,
				IPv4IPIPTunnelAddr: fmt.Sprintf("10.65.%d.1", i),
			}
		}
		Eventually(func() error {
			_, err = client.Nodes().Create(utils.Ctx, felixNode, utils.NoOptions)
			if err != nil {
				log.WithError(err).Warn("Failed to create node")
			}
			return err
		}, "10s", "500ms").ShouldNot(HaveOccurred())

		felixes = append(felixes, felix)
	}

	// Set up routes between the hosts, note: we're not using IPAM here but we set up similar
	// CIDR-based routes.
	for i, iFelix := range felixes {
		for j, jFelix := range felixes {
			if i == j {
				continue
			}

			jBlock := fmt.Sprintf("10.65.%d.0/24", j)
			err := iFelix.ExecMayFail("ip", "route", "add", jBlock, "via", jFelix.IP, "dev", "tunl0", "onlink")
			Expect(err).ToNot(HaveOccurred())
		}
	}
	success = true
	return
}

func mustInitDatastore(client client.Interface) {
	Eventually(func() error {
		log.Info("Initializing the datastore...")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		err := client.EnsureInitialized(
			ctx,
			"v3.0.0-test",
			"v2.0.0-test",
			"felix-fv",
		)
		log.WithError(err).Info("EnsureInitialized result")
		return err
	}).ShouldNot(HaveOccurred())
}

func addCNXLicense(client client.Interface) {
	Eventually(func() error {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		licenseKey := api.NewLicenseKey()
		licenseKey.Name = "default"
		licenseKey.Spec.Token = `eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJlaWNWbHlTbGxFMlAtQ25tIiwidGFnIjoiTk1KSHlRV2M1UWZ6M1dydHNCamxhZyIsInR5cCI6IkpXVCJ9.afBv55v15cFsaHqcsyDkfA.yBMyDIRFBtWxyNxI.Q18a_G6i2kiN0NsqtGSQjc0o2CrkdivRJFkpAlkYIttBAultPADLZmfgf0nzVqZkKAkOGSbIxjY5BgW59FEyaiEs8sL11HZqPB8l2eOqK4BSj5wx3yEhsFzQkD1pZZz8qVgE0Ml3SaSiGVhe4ADTiSsUBbU9JD_aRaa4m1QvS4IQiT_QuWxUtOi-LRXsvHURnkTs3K_WGu7_QW5RRHDGD_CP2kfTUMeSvcWSiT8vgrgPj5q4Zpz4XTWNT-u0sJraWu79tOqCu9YwKeDVMKgJ04sunGc9xsimkhUmOnwuiIEeR24GyL7I5FDrCUC6Oiif62o_ECaQA6NjHAFdq-LNCIb902tKD0BQ-q6AzUrjs21GNr9_oJZJXKL6m74UJULMVgxXZKze2IH9EXtQ0b2jHbi9-qyMp6Rc34Z4HtYmQPB3CRHjDTmzUpEXOsF-reYffRHLJY5DUk7fDfTnhBmUksYonuuGLKep1_YYAiAhkomj7mupFNVN5JnZx8P-v4cfAr4PZxF6Lw5utN5R1hArroYA1Z-2Et0LC6BbE6Q1j7_zmaBs2BEnNfWNn2LFBBOCHzax51ISz_DIcGSidsRDNE9vQDYhcb9MGqOtaCDAA5zHCArVxu2PiwJj6JNbdNB9nvLWlAqxUU4zJwNPFd9xQIR53RFNB0LHID-ab_H7_NFX0auolwSz5Fm14ID4SKvD7_1aqUJG9_WiEtNz9yZJL5vkspdSxnR59L4alUYErxSEWGmOIBvJPemftZBilH1Vmxt0MFyu7sxK_uEJ55OtxNXCfaa_MPp0Yhn9mjTeCSMH8dV6ahZuL8B85BHjFkqY_nLV5UKEvPcyflo4JLDAOvhTZ0bbqvheEx48FQPisSJoK5zY61FqK1tFrID5rdJQ4RMpe4Bix0Dy213hN08U1iNklHUgR-MMw2f4sfGouBm-3B-7P9bqwQlEVyKLkyBzOgWd0PADc0i5bdxCxoqL8AAehPTEGIk-lb2TKe71dCW47oZQwigRgbLHRJnYF9iVlFoXXf-MLH_edh5Gi2OD397MtuBvpGWS8KVjiyUYX-NhvOqgzqrRCH-7kRkmYBsL446hNzGYMjbxut488a2amVrsIuR4oerJnkSdK3o.MnNW4M-g2iiXOi1GVe5zaQ`
		licenseKey.Spec.Certificate = `-----BEGIN CERTIFICATE-----
MIIFxjCCA66gAwIBAgIQVq3rz5D4nQF1fIgMEh71DzANBgkqhkiG9w0BAQsFADCB
tTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1
cml0eSA8c2lydEB0aWdlcmEuaW8+MT8wPQYDVQQDEzZUaWdlcmEgRW50aXRsZW1l
bnRzIEludGVybWVkaWF0ZSBDZXJ0aWZpY2F0ZSBBdXRob3JpdHkwHhcNMTgwNDA1
MjEzMDI5WhcNMjAxMDA2MjEzMDI5WjCBnjELMAkGA1UEBhMCVVMxEzARBgNVBAgT
CkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xFDASBgNVBAoTC1Rp
Z2VyYSwgSW5jMSIwIAYDVQQLDBlTZWN1cml0eSA8c2lydEB0aWdlcmEuaW8+MSgw
JgYDVQQDEx9UaWdlcmEgRW50aXRsZW1lbnRzIENlcnRpZmljYXRlMIIBojANBgkq
hkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAwg3LkeHTwMi651af/HEXi1tpM4K0LVqb
5oUxX5b5jjgi+LHMPzMI6oU+NoGPHNqirhAQqK/k7W7r0oaMe1APWzaCAZpHiMxE
MlsAXmLVUrKg/g+hgrqeije3JDQutnN9h5oZnsg1IneBArnE/AKIHH8XE79yMG49
LaKpPGhpF8NoG2yoWFp2ekihSohvqKxa3m6pxoBVdwNxN0AfWxb60p2SF0lOi6B3
hgK6+ILy08ZqXefiUs+GC1Af4qI1jRhPkjv3qv+H1aQVrq6BqKFXwWIlXCXF57CR
hvUaTOG3fGtlVyiPE4+wi7QDo0cU/+Gx4mNzvmc6lRjz1c5yKxdYvgwXajSBx2pw
kTP0iJxI64zv7u3BZEEII6ak9mgUU1CeGZ1KR2Xu80JiWHAYNOiUKCBYHNKDCUYl
RBErYcAWz2mBpkKyP6hbH16GjXHTTdq5xENmRDHabpHw5o+21LkWBY25EaxjwcZa
Y3qMIOllTZ2iRrXu7fSP6iDjtFCcE2bFAgMBAAGjZzBlMA4GA1UdDwEB/wQEAwIF
oDATBgNVHSUEDDAKBggrBgEFBQcDAjAdBgNVHQ4EFgQUIY7LzqNTzgyTBE5efHb5
kZ71BUEwHwYDVR0jBBgwFoAUxZA5kifzo4NniQfGKb+4wruTIFowDQYJKoZIhvcN
AQELBQADggIBAAK207LaqMrnphF6CFQnkMLbskSpDZsKfqqNB52poRvUrNVUOB1w
3dSEaBUjhFgUU6yzF+xnuH84XVbjD7qlM3YbdiKvJS9jrm71saCKMNc+b9HSeQAU
DGY7GPb7Y/LG0GKYawYJcPpvRCNnDLsSVn5N4J1foWAWnxuQ6k57ymWwcddibYHD
OPakOvO4beAnvax3+K5dqF0bh2Np79YolKdIgUVzf4KSBRN4ZE3AOKlBfiKUvWy6
nRGvu8O/8VaI0vGaOdXvWA5b61H0o5cm50A88tTm2LHxTXynE3AYriHxsWBbRpoM
oFnmDaQtGY67S6xGfQbwxrwCFd1l7rGsyBQ17cuusOvMNZEEWraLY/738yWKw3qX
U7KBxdPWPIPd6iDzVjcZrS8AehUEfNQ5yd26gDgW+rZYJoAFYv0vydMEyoI53xXs
cpY84qV37ZC8wYicugidg9cFtD+1E0nVgOLXPkHnmc7lIDHFiWQKfOieH+KoVCbb
zdFu3rhW31ygphRmgszkHwApllCTBBMOqMaBpS8eHCnetOITvyB4Kiu1/nKvVxhY
exit11KQv8F3kTIUQRm0qw00TSBjuQHKoG83yfimlQ8OazciT+aLpVaY8SOrrNnL
IJ8dHgTpF9WWHxx04DDzqrT7Xq99F9RzDzM7dSizGxIxonoWcBjiF6n5
-----END CERTIFICATE-----`
		_, err := client.LicenseKey().Create(ctx, licenseKey, options.SetOptions{})
		log.WithError(err).Info("Add license result")
		return err
	}).ShouldNot(HaveOccurred())
}
