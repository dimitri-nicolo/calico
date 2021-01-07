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

func (c *collector) setupStatsDumping() {
	// TODO (doublek): This may not be the best place to put this. Consider
	// moving the signal handler and logging to file logic out of the collector
	// and simply out to appropriate sink on different messages.
	signal.Notify(c.sigChan, syscall.SIGUSR2)

	err := os.MkdirAll(path.Dir(c.config.StatsDumpFilePath), 0755)
	if err != nil {
		log.WithError(err).Fatal("Failed to create log dir")
	}

	rotAwareFile, err := rfw.Open(c.config.StatsDumpFilePath, 0644)
	if err != nil {
		log.WithError(err).Fatal("Failed to open log file")
	}

	// Attributes have to be directly set for instantiated logger as opposed
	// to the module level log object.
	c.dumpLog.Formatter = &MessageOnlyFormatter{}
	c.dumpLog.Level = log.InfoLevel
	c.dumpLog.Out = rotAwareFile
}
