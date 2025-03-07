// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package config

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

// ConfigureLogging configures the logging framework. The logging level that will
// be used is passed in as a parameter. Otherwise, it will default to WARN
// The output will be set to STDOUT and the format is TextFormat
func ConfigureLogging(logLevel string) {
	// Install a hook that adds file/line number information.
	logutils.ConfigureFormatter("oiler")
	logrus.SetOutput(os.Stdout)

	// Override with desired log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Error("Invalid logging level passed in. Will use default level set to WARN")
		// Setting default to WARN
		level = logrus.WarnLevel
	}

	logrus.SetLevel(level)
}
