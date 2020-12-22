// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/mipearson/rfw"
	log "github.com/sirupsen/logrus"
)

func setupStatsDumping(sigChan chan os.Signal, filePath string, dumpLog *log.Logger) {
	// TODO (doublek): This may not be the best place to put this. Consider
	// moving the signal handler and logging to file logic out of the collector
	// and simply out to appropriate sink on different messages.
	signal.Notify(sigChan, syscall.SIGUSR2)

	err := os.MkdirAll(path.Dir(filePath), 0755)
	if err != nil {
		log.WithError(err).Fatal("Failed to create log dir")
	}

	rotAwareFile, err := rfw.Open(filePath, 0644)
	if err != nil {
		log.WithError(err).Fatal("Failed to open log file")
	}

	// Attributes have to be directly set for instantiated logger as opposed
	// to the module level log object.
	dumpLog.Formatter = &MessageOnlyFormatter{}
	dumpLog.Level = log.InfoLevel
	dumpLog.Out = rotAwareFile
}
