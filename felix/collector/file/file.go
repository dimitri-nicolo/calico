// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/projectcalico/calico/felix/collector/dnslog"
	"github.com/projectcalico/calico/felix/collector/flowlog"
	"github.com/projectcalico/calico/felix/collector/l7log"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const (
	FlowLogFilename = "flows.log"
	DNSLogFilename  = "dns.log"
	L7LogFilename   = "l7.log"
)

// FileReporter is a Reporter that writes logs to a local,
// auto-rotated log file. We write one JSON-encoded log per line.
type FileReporter struct {
	directory string
	fileName  string
	maxMB     int
	numFiles  int
	logger    *bufio.Writer
}

func NewReporter(directory, fileName string, maxMB, numFiles int) *FileReporter {
	return &FileReporter{directory: directory, fileName: fileName, maxMB: maxMB, numFiles: numFiles}
}

func (f *FileReporter) Start() error {
	if f.logger != nil {
		// Already initialized; no-op
		return nil
	}
	// Create the log directory before creating the logger.  If the logger creates it, it will do so
	// with permission 0744, meaning that non-root users won't be able to "see" files in the
	// directory, since "execute" permission on a directory needs to be granted.
	err := os.MkdirAll(f.directory, 0o755)
	if err != nil {
		return fmt.Errorf("can't make directories for new logfile: %s", err)
	}
	logger := &lumberjack.Logger{
		Filename:   path.Join(f.directory, f.fileName),
		MaxSize:    f.maxMB,
		MaxBackups: f.numFiles,
	}
	f.logger = bufio.NewWriterSize(logger, 1<<16)
	return nil
}

func (f *FileReporter) Report(logSlice interface{}) (err error) {
	enc := json.NewEncoder(f.logger)

	defer func() {
		flushErr := f.logger.Flush()
		if flushErr != nil {
			log.WithError(flushErr).Error("Failed to flush log file.")
			if err == nil {
				err = flushErr
			}
		}
	}()

	switch logs := logSlice.(type) {
	case []*flowlog.FlowLog:
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithField("num", len(logs)).Debug("Dispatching flow logs to file")
		}
		// Optimisation: we re-use the same output object for each log to avoid
		// a (large) allocation per log.
		var output flowlog.JSONOutput
		for _, l := range logs {
			output.FillFrom(l)
			err := enc.Encode(&output)
			if err != nil {
				log.WithError(err).
					WithField("flowLog", output).
					Error("Unable to serialize flow log to file.")
				return err
			}
		}
	case []*v1.DNSLog:
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithField("num", len(logs)).Debug("Dispatching DNS logs to file")
		}
		// Optimisation: put this outside the loop to avoid an allocation per
		// excess log.
		var excessLog dnslog.DNSExcessLog
		for _, l := range logs {
			var err error
			if l.Type == v1.DNSLogTypeUnlogged {
				excessLog = dnslog.DNSExcessLog{
					StartTime: l.StartTime,
					EndTime:   l.EndTime,
					Type:      l.Type,
					Count:     l.Count,
				}
				err = enc.Encode(&excessLog)
			} else {
				err = enc.Encode(l)
			}
			if err != nil {
				log.WithError(err).
					WithField("dnsLog", l).
					Error("Unable to serialize DNS log to JSON")
				return err
			}
		}
	case []*l7log.L7Log:
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithField("num", len(logs)).Debug("Dispatching L7 logs to file")
		}
		for _, l := range logs {
			err := enc.Encode(l)
			if err != nil {
				log.WithError(err).
					WithField("l7Log", l).
					Error("Unable to serialize L7 log to JSON")
				return err
			}
		}
	default:
		log.Panic("Unexpected kind of log in file dispatcher")
	}
	return nil
}
