// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package collector

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// We need to use another package for Windows rotate logs.
func setupStatsDumping(sigChan chan os.Signal, filePath string, dumpLog *log.Logger) {
	// TODO (doublek): This may not be the best place to put this. Consider
	// moving the signal handler and logging to file logic out of the collector
	// and simply out to appropriate sink on different messages.
	signal.Notify(
		sigChan,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)

	// path.Dir is used for slash-separated paths only.
	// filepath.Dir is os-specific.
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		log.WithError(err).Fatal("Failed to create log dir")
	}

	rotAwareFile, err := rotatelogs.New(
		filePath+"collector_log.%Y%m%d%H%M",
		rotatelogs.WithLinkName(filePath+"collector_log"),
		rotatelogs.WithMaxAge(24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)
	if err != nil {
		log.WithError(err).Fatal("Failed to open log file")
	}

	// Attributes have to be directly set for instantiated logger as opposed
	// to the module level log object.
	dumpLog.Formatter = &MessageOnlyFormatter{}
	dumpLog.Level = log.InfoLevel
	dumpLog.Out = rotAwareFile
}
