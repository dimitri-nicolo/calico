// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package reporter

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/collector/types/metric"
)

type Syslog struct{}

// NewSyslogReporter configures and returns a SyslogReporter.
// Network and Address can be used to configure remote syslogging. Leaving both
// of these values empty implies using local syslog such as /dev/log.
func NewSyslog(network, address string) *Syslog {
	return &Syslog{}
}

func (sr *Syslog) Start() {
	log.Info("syslog reporting is not supported on a windows node")
}

func (sr *Syslog) Report(mu metric.Update) error {
	return nil
}
