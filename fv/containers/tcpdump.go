// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package containers

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"

	"sync"

	. "github.com/onsi/gomega"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/sirupsen/logrus"
)

func AttachTCPDump(c *Container, iface string) *TCPDump {
	t := &TCPDump{
		containerID:   c.GetID(),
		containerName: c.Name,
		iface:         iface,
		matchers:      map[string]*tcpDumpMatcher{},
	}
	return t
}

type stringMatcher interface {
	MatchString(string) bool
}

type tcpDumpMatcher struct {
	regex stringMatcher
	count int
}

type TCPDump struct {
	lock sync.Mutex

	containerID   string
	containerName string
	iface         string
	cmd           *exec.Cmd
	out           io.ReadCloser

	matchers map[string]*tcpDumpMatcher
}

func (t *TCPDump) AddMatcher(name string, s stringMatcher) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.matchers[name] = &tcpDumpMatcher{
		regex: s,
	}
}

func (t *TCPDump) MatchCount(name string) int {
	t.lock.Lock()
	defer t.lock.Unlock()

	c := t.matchers[name].count
	logrus.Infof("Match count for %s is %v", name, c)
	return c
}

func (t *TCPDump) Start() {
	// docker run --rm --network=container:48b6c5f44d57 --privileged corfr/tcpdump -nli cali01

	t.cmd = utils.Command("docker", "run",
		"--rm",
		fmt.Sprintf("--network=container:%s", t.containerID),
		"--privileged",
		"corfr/tcpdump", "-nli", t.iface,
	)
	var err error
	t.out, err = t.cmd.StdoutPipe()
	Expect(err).NotTo(HaveOccurred())

	go t.readStdout()

	err = t.cmd.Start()
	Expect(err).NotTo(HaveOccurred())
}

func (t *TCPDump) Stop() {
	err := t.cmd.Process.Kill()
	if err != nil {
		logrus.WithError(err).Error("Failed to kill tcp dump; maybe it failed to start?")
	}
}

func (t *TCPDump) readStdout() {
	s := bufio.NewScanner(t.out)
	for s.Scan() {
		line := s.Text()
		logrus.Infof("tcpdump [%s] %s", t.containerName, line)

		t.lock.Lock()
		for _, m := range t.matchers {
			if m.regex.MatchString(line) {
				m.count++
			}
		}
		t.lock.Unlock()
	}
	logrus.WithError(s.Err()).Info("TCPDump stdout finished")
}
