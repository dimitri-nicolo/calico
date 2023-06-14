// Copyright (c) 2017-2023 Tigera, Inc. All rights reserved.

package reporter

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/collector/types/metric"
	logutil "github.com/projectcalico/calico/felix/logutils"
)

const (
	reporterChanBufferSize = 1000
)

type MetricsReporter interface {
	Start()
	Report(mu metric.Update) error
}

// rename reporter to dispatch
type LogDispatcher interface {
	Initialize() error
	Dispatch(logSlice interface{}) error
}

type Manager struct {
	ReportChan            chan metric.Update
	reporters             []MetricsReporter
	DisplayDebugTraceLogs bool
}

func NewManager(displayDebugTraceLogs bool) *Manager {
	return &Manager{
		DisplayDebugTraceLogs: displayDebugTraceLogs,
		ReportChan:            make(chan metric.Update, reporterChanBufferSize),
	}
}

func (m *Manager) RegisterMetricsReporter(mr MetricsReporter) {
	m.reporters = append(m.reporters, mr)
}

func (m *Manager) Start() {
	for _, reporter := range m.reporters {
		reporter.Start()
	}
	go m.start()
}

func (m *Manager) start() {
	log.Info("Starting ReporterManager")
	for {
		// TODO(doublek): Channel for stopping the reporter.
		select {
		case mu := <-m.ReportChan:
			logutil.Tracef(m.DisplayDebugTraceLogs, "Received metric update %v", mu)
			for _, reporter := range m.reporters {
				reporter.Report(mu)
			}
		}
	}
}
