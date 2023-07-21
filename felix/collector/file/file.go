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
	writeLog := func(b []byte) error {
		b = append(b, '\n')
		// It is an error to call Dispatch before Initialize, so it's safe to
		// assume d.logger is non-nil.
		_, err := f.logger.Write(b)
		if err != nil {
			// NOTE: the FlowLogsReporter ignores errors returned by Dispatch,
			// so log the error here.  We don't want to do anything more drastic
			// like retrying because we don't know if the error is even
			// recoverable.
			log.WithError(err).Error("unable to write flow log to file")
			return err
		}
		return nil
	}
	switch fl := logSlice.(type) {
	case []*flowlog.FlowLog:
		log.Debug("Dispatching flow logs to file")
		for _, l := range fl {
			o := flowlog.ToOutput(l)
			b, err := json.Marshal(o)
			if err != nil {
				// This indicates a bug, not a runtime error since we should always
				// be able to serialize.
				log.WithError(err).
					WithField("FlowLog", o).
					Panic("unable to serialize flow log to JSON")
			}
			if err = writeLog(b); err != nil {
				return err
			}
		}
	case []*v1.DNSLog:
		log.Debug("Dispatching DNS logs to file")
		for _, l := range fl {
			var b []byte
			var err error
			if l.Type == v1.DNSLogTypeUnlogged {
				b, err = json.Marshal(&dnslog.DNSExcessLog{
					StartTime: l.StartTime,
					EndTime:   l.EndTime,
					Type:      l.Type,
					Count:     l.Count,
				})
			} else {
				b, err = json.Marshal(l)
			}
			if err != nil {
				// This indicates a bug, not a runtime error since we should always
				// be able to serialize.
				log.WithError(err).
					WithField("DNSLog", l).
					Panic("unable to serialize DNS log to JSON")
			}
			if err = writeLog(b); err != nil {
				return err
			}
		}
	case []*l7log.L7Log:
		log.Debug("Dispatching L7 logs to file")
		for _, l := range fl {
			b, err := json.Marshal(l)
			if err != nil {
				// This indicates a bug, not a runtime error since we should always
				// be able to serialize.
				log.WithError(err).
					WithField("L7Log", l).
					Panic("unable to serialize L7 log to JSON")
			}
			if err = writeLog(b); err != nil {
				return err
			}
		}
	default:
		log.Panic("Unexpected kind of log in file dispatcher")
	}
	return nil
}
