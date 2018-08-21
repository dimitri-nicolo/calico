// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"io"

	"path"

	"encoding/json"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

const FlowLogFilename = "flows.log"

type fileDispatcher struct {
	directory string
	maxMB     int
	numFiles  int
	logger    io.WriteCloser
}

func NewFileDispatcher(directory string, maxMB, numFiles int) FlowLogDispatcher {
	return &fileDispatcher{directory: directory, maxMB: maxMB, numFiles: numFiles}
}

func (d *fileDispatcher) Initialize() error {
	if d.logger != nil {
		// Already initialized; no-op
		return nil
	}
	d.logger = &lumberjack.Logger{
		Filename:   path.Join(d.directory, FlowLogFilename),
		MaxSize:    d.maxMB,
		MaxBackups: d.numFiles,
	}
	return nil
}

func (d *fileDispatcher) Dispatch(fl []*FlowLog) error {
	log.Debug("Dispatching flow logs to file")
	for _, l := range fl {
		o := toOutput(l)
		b, err := json.Marshal(o)
		if err != nil {
			// This indicates a bug, not a runtime error since we should always
			// be able to serialize.
			log.WithError(err).
				WithField("FlowLog", o).
				Panic("unable to serialize flow log to JSON")
		}
		b = append(b, '\n')
		// It is an error to call Dispatch before Initialize, so it's safe to
		// assume d.logger is non-nil.
		_, err = d.logger.Write(b)
		if err != nil {
			// NOTE: the FlowLogsReporter ignores errors returned by Dispatch,
			// so log the error here.  We don't want to do anything more drastic
			// like retrying because we don't know if the error is even
			// recoverable.
			log.WithError(err).Error("unable to write flow log to file")
			return err
		}
	}
	return nil
}
