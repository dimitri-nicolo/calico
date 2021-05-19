// Copyright (c) 2021 Tigera, Inc. All rights reserved.
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

package tproxy

import (
	"bufio"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/utils"
)

var connRegexp = regexp.MustCompile(`Proxying from (\d+\.\d+\.\d+\.\d+):\d+ to (\d+\.\d+\.\d+\.\d+:\d+) orig dest (\d+\.\d+\.\d+\.\d+:\d+)`)

type TProxy struct {
	cmd              *exec.Cmd
	out              io.ReadCloser
	err              io.ReadCloser
	listeningStarted chan struct{}

	cname string
	port  uint16

	connections map[ConnKey]int
}

type ConnKey struct {
	ClientIP      string
	ServiceIPPort string
	PodIPPort     string
}

func New(f *infrastructure.Felix, port uint16) *TProxy {
	f.EnsureBinary("tproxy")
	return &TProxy{
		cname: f.Name,
		port:  port,

		listeningStarted: make(chan struct{}),

		connections: make(map[ConnKey]int),
	}
}

func (t *TProxy) Start() {
	t.cmd = utils.Command("docker", "exec", t.cname, "/tproxy", strconv.Itoa(int(t.port)))

	var err error
	t.out, err = t.cmd.StdoutPipe()
	Expect(err).NotTo(HaveOccurred())

	t.err, err = t.cmd.StderrPipe()
	Expect(err).NotTo(HaveOccurred())

	go t.readStdout()
	go t.readStderr()

	err = t.cmd.Start()

	select {
	case <-t.listeningStarted:
	case <-time.After(60 * time.Second):
		ginkgo.Fail("Failed to start tproxy: it never reported that it was listening")
	}

	Expect(err).NotTo(HaveOccurred())
}

func (t *TProxy) Stop() {
	err := t.cmd.Process.Kill()
	if err != nil {
		log.WithError(err).Error("Failed to kill tproxy; maybe it failed to start?")
	}
}

func (t *TProxy) readStdout() {
	s := bufio.NewScanner(t.out)
	for s.Scan() {
		line := s.Text()

		log.Infof("[tproxy %s] %s", t.cname, line)
	}
	log.WithError(s.Err()).Info("TProxy stdout finished")
}

func (t *TProxy) readStderr() {
	defer ginkgo.GinkgoRecover()

	s := bufio.NewScanner(t.err)
	closedChan := false
	safeClose := func() {
		if !closedChan {
			close(t.listeningStarted)
			closedChan = true
		}
	}

	listening := false

	defer func() {
		Expect(listening).To(BeTrue())
		safeClose()
	}()

	for s.Scan() {
		line := s.Text()
		log.Infof("[tproxy %s] ERR: %s", t.cname, line)
		if !listening && strings.Contains(line, "Listening") {
			listening = true
			safeClose()
			continue
		}

		m := connRegexp.FindStringSubmatch(line)
		if len(m) == 4 {
			t.connections[ConnKey{ClientIP: m[1], PodIPPort: m[2], ServiceIPPort: m[3]}]++
		}
	}
	log.WithError(s.Err()).Info("TProxy stderr finished")
}

func (t *TProxy) ConnCount(client, pod, service string) int {
	return t.connections[ConnKey{ClientIP: client, PodIPPort: pod, ServiceIPPort: service}]
}

func (t *TProxy) Connections() map[ConnKey]int {
	return t.connections
}
