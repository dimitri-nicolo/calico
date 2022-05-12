// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package forwarder

import (
	"fmt"
	"io"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogDispatcher is the external interface for dispatchers. For now there is only the file dispatcher.
type LogDispatcher interface {
	Initialize() error
	Dispatch(data []byte) error
}

// fileDispatcher is a LogDispatcher that writes logs to a local,
// auto-rotated log file.  We write one JSON-encoded log per line.
type fileDispatcher struct {
	directory string
	fileName  string
	maxMB     int
	numFiles  int
	logger    io.WriteCloser
}

// NewFileDispatcher returns a new LogDispatcher of type file dispatcher
func NewFileDispatcher(directory, fileName string, maxMB, numFiles int) LogDispatcher {
	return &fileDispatcher{
		directory: directory,
		fileName:  fileName,
		maxMB:     maxMB,
		numFiles:  numFiles,
	}
}

// Initialize the given file dispatcher
func (d *fileDispatcher) Initialize() error {
	if d.logger != nil {
		// Already initialized; no-op
		return nil
	}
	// Create the log directory before creating the logger.  If the logger creates it, it will do so
	// with permission 0755, meaning that non-root users won't be able to "see" files in the
	// directory, since "execute" permission on a directory needs to be granted.
	err := os.MkdirAll(d.directory, 0755)
	if err != nil {
		return fmt.Errorf("can't make directories for new logfile: %s", err)
	}
	d.logger = &lumberjack.Logger{
		Filename:   path.Join(d.directory, d.fileName),
		MaxSize:    d.maxMB,
		MaxBackups: d.numFiles,
	}
	return nil
}

// Dispatch takes in serialized data and writes it to the pre-configured destination file.
func (d *fileDispatcher) Dispatch(data []byte) error {
	writeLog := func(b []byte) (int, error) {
		b = append(b, '\n')
		// It is an error to call Dispatch before Initialize, so it's safe to
		// assume d.logger is non-nil.
		n, err := d.logger.Write(b)
		if err != nil {
			log.WithError(err).Error("unable to dispatch data to file")
			return n, err
		}
		return n, nil
	}
	bytesWritten, err := writeLog(data)
	if err != nil {
		return err
	}
	log.Debugf("Dispatcher wrote %d bytes", bytesWritten)
	return nil
}
