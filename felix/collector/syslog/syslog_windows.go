// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package syslog

import (
	log "github.com/sirupsen/logrus"
)

type Syslog struct{}

// NewSyslogReporter configures and returns a SyslogReporter.
// Network and Address can be used to configure remote syslogging. Leaving both
// of these values empty implies using local syslog such as /dev/log.
func New(network, address string) *Syslog {
	return &Syslog{}
}

func (s *Syslog) Start() error {
	log.Info("syslog reporting is not supported on a windows node")
	return nil
}

func (s *Syslog) Report(any) error {
	return nil
}
