// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	log "github.com/sirupsen/logrus"
)

type SyslogReporter struct{}

// NewSyslogReporter configures and returns a SyslogReporter.
// Network and Address can be used to configure remote syslogging. Leaving both
// of these values empty implies using local syslog such as /dev/log.
func NewSyslogReporter(network, address string) *SyslogReporter {
	return &SyslogReporter{}
}

func (sr *SyslogReporter) Start() {
	log.Info("syslog reporting is not supported on a windows node")
}

func (sr *SyslogReporter) Report(mu MetricUpdate) error {
	return nil
}
