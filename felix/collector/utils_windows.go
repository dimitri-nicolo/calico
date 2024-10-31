// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package collector

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
)

// We need to use another package for Windows rotate logs.
func (c *collector) setupStatsDumping() {
	// TODO (doublek): This may not be the best place to put this. Consider
	// moving the signal handler and logging to file logic out of the collector
	// and simply out to appropriate sink on different messages.
	signal.Notify(
		c.sigChan,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)

	// path.Dir is used for slash-separated paths only.
	// filepath.Dir is os-specific.
	path := c.config.StatsDumpFilePath
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		log.WithError(err).Fatal("Failed to create log dir")
	}

	rotAwareFile, err := rotatelogs.New(
		path+"collector_log.%Y%m%d%H%M",
		rotatelogs.WithLinkName(path+"collector_log"),
		rotatelogs.WithMaxAge(24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)
	if err != nil {
		log.WithError(err).Fatal("Failed to open log file")
	}

	// Attributes have to be directly set for instantiated logger as opposed
	// to the module level log object.
	c.dumpLog.Formatter = &MessageOnlyFormatter{}
	c.dumpLog.Level = log.InfoLevel
	c.dumpLog.Out = rotAwareFile
}
