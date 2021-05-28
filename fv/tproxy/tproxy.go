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
	"sync"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/utils"
)

var proxiedRegexp = regexp.MustCompile(
	`Proxying from (\d+\.\d+\.\d+\.\d+):\d+ to (\d+\.\d+\.\d+\.\d+:\d+) orig dest (\d+\.\d+\.\d+\.\d+:\d+)`)

var acceptedRegexp = regexp.MustCompile(
	`Accepted connection from (\d+\.\d+\.\d+\.\d+):\d+ to (\d+\.\d+\.\d+\.\d+:\d+) orig dest (\d+\.\d+\.\d+\.\d+:\d+)`)

type TProxy struct {
	cmd              *exec.Cmd
	out              io.ReadCloser
	err              io.ReadCloser
	listeningStarted chan struct{}

	cname  string
	port   uint16
	portNp uint16

	proxied  map[ConnKey]int
	accepted map[ConnKey]int
	connLock sync.Mutex
}

type ConnKey struct {
	ClientIP      string
	ServiceIPPort string
	PodIPPort     string
}

func New(f *infrastructure.Felix, port, portNp uint16) *TProxy {
	f.EnsureBinary("tproxy")
	return &TProxy{
		cname:  f.Name,
		port:   port,
		portNp: portNp,

		listeningStarted: make(chan struct{}),

		proxied:  make(map[ConnKey]int),
		accepted: make(map[ConnKey]int),
	}
}

func (t *TProxy) Start() {
	t.cmd = utils.Command("docker", "exec", t.cname, "/tproxy",
		strconv.Itoa(int(t.port)), strconv.Itoa(int(t.portNp)))

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

		m := acceptedRegexp.FindStringSubmatch(line)
		if len(m) == 4 {
			t.acceptedAdd(m[1], m[2], m[3])
			continue
		}
		m = proxiedRegexp.FindStringSubmatch(line)
		if len(m) == 4 {
			t.proxiedAdd(m[1], m[2], m[3])
			continue
		}
	}
	log.WithError(s.Err()).Info("TProxy stderr finished")
}

func (t *TProxy) proxiedAdd(client, pod, service string) {
	t.connLock.Lock()
	t.proxied[ConnKey{ClientIP: client, PodIPPort: pod, ServiceIPPort: service}]++
	t.connLock.Unlock()
}

func (t *TProxy) ProxiedCount(client, pod, service string) int {
	t.connLock.Lock()
	defer t.connLock.Unlock()
	return t.proxied[ConnKey{ClientIP: client, PodIPPort: pod, ServiceIPPort: service}]
}

func (t *TProxy) acceptedAdd(client, pod, service string) {
	t.connLock.Lock()
	t.accepted[ConnKey{ClientIP: client, PodIPPort: pod, ServiceIPPort: service}]++
	t.connLock.Unlock()
}

func (t *TProxy) AcceptedCount(client, pod, service string) int {
	t.connLock.Lock()
	defer t.connLock.Unlock()
	return t.accepted[ConnKey{ClientIP: client, PodIPPort: pod, ServiceIPPort: service}]
}
