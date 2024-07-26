// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package file

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/projectcalico/calico/felix/collector/dnslog"
	"github.com/projectcalico/calico/felix/collector/flowlog"
	"github.com/projectcalico/calico/felix/collector/l7log"
	"github.com/projectcalico/calico/felix/collector/types"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const (
	FlowLogFilename = "flows.log"
	DNSLogFilename  = "dns.log"
	L7LogFilename   = "l7.log"
)

// fileReporter is a Reporter that writes logs to a local,
// auto-rotated log file. We write one JSON-encoded log per line.
type fileReporter struct {
	directory string
	fileName  string
	maxMB     int
	numFiles  int
	logger    io.WriteCloser
}

func NewReporter(directory, fileName string, maxMB, numFiles int) types.Reporter {
	return &fileReporter{directory: directory, fileName: fileName, maxMB: maxMB, numFiles: numFiles}
}

func (f *fileReporter) Start() error {
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
	f.logger = &lumberjack.Logger{
		Filename:   path.Join(f.directory, f.fileName),
		MaxSize:    f.maxMB,
		MaxBackups: f.numFiles,
	}
	return nil
}

func (f *fileReporter) Report(logSlice interface{}) error {
	enc := json.NewEncoder(f.logger)

	switch fl := logSlice.(type) {
	case []*flowlog.FlowLog:
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithField("num", len(fl)).Debug("Dispatching flow logs to file")
		}
		for _, l := range fl {
			o := flowlog.ToOutput(l)
			err := enc.Encode(o)
			if err != nil {
				log.WithError(err).
					WithField("flowLog", o).
					Error("Unable to serialize flow log to file.")
				return err
			}
		}
	case []*v1.DNSLog:
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithField("num", len(fl)).Debug("Dispatching DNS logs to file")
		}
		for _, l := range fl {
			var objToLog any = l
			if l.Type == v1.DNSLogTypeUnlogged {
				objToLog = &dnslog.DNSExcessLog{
					StartTime: l.StartTime,
					EndTime:   l.EndTime,
					Type:      l.Type,
					Count:     l.Count,
				}
			}
			err := enc.Encode(objToLog)
			if err != nil {
				log.WithError(err).
					WithField("dnsLog", l).
					Error("Unable to serialize DNS log to JSON")
				return err
			}
		}
	case []*l7log.L7Log:
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithField("num", len(fl)).Debug("Dispatching L7 logs to file")
		}
		for _, l := range fl {
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
