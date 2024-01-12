// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package utils

import (
	"io"

	"github.com/sirupsen/logrus"
)

// NewLogrusWriter returns an implementation of io.Writer that writes to the given logrus context logger.
func NewLogrusWriter(log *logrus.Entry) io.Writer {
	return &logrusWriter{log: log}
}

type logrusWriter struct {
	log *logrus.Entry
}

func (l *logrusWriter) Write(p []byte) (n int, err error) {
	l.log.Info(string(p))
	return len(p), nil
}
