// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package exec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

type Exec interface {
	Start() error
	Wait() error
	Stop()
}

type Snort func(podName string, iface string, namespace string, dpiName string, alertFileBasePath string,
	alertFileSize int, communityRulesFile string) (Exec, error)

func NewExec(podName string,
	iface string,
	namespace string,
	dpiName string,
	alertFileBasePath string,
	alertFileSize int,
	communityRulesFile string,
) (Exec, error) {
	s := &snort{}

	// -c <config path>		: configuration
	// -q					: quiet mode
	// -y					: include year in output
	// -k none				: checksum level
	// -y					: show year in timestamp
	// -i <iface>			: WEP interface
	// -l <path>			: alert output directory
	// --daq afpacket		: packet acquisition module
	// --lua <alert type>	: type/level of alert
	// -R <rules path>		: path to the rules
	logPath := fmt.Sprintf("%s/%s/%s/%s", alertFileBasePath, namespace, dpiName, podName)
	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	s.cmd = exec.Command(
		"snort",
		"-c",
		"/usr/local/etc/snort/snort.lua",
		"-q",
		"-y",
		"-k", "none",
		"-i", iface,
		"-l", logPath,
		"--daq", "afpacket",
		"--lua", fmt.Sprintf("alert_fast={ file = true, limit = %d }", alertFileSize),
		"-R", communityRulesFile,
	)

	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	return s, nil
}

// snort implements the Exec interface
type snort struct {
	cmd *exec.Cmd
}

func (s *snort) Start() error {
	return s.cmd.Start()
}

func (s *snort) Wait() error {
	return s.cmd.Wait()
}

func (s *snort) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		// Shutdown snort normally by sending SIGTERM
		err := s.cmd.Process.Signal(syscall.SIGTERM)
		if err != nil && !strings.Contains(err.Error(), "process already finished") {
			log.WithError(err).Errorf("failed to kill process snort")
		}
	}
}
