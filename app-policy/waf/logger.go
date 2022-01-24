// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

const LogPath string = "/var/log/calico/waf"
const LogFile string = "waf.log"

var Logger logrus.Logger

func init() {
	logrus.Info("WAF logging initialization beginning.")
	defer logrus.Info("WAF logging initialization completed.")

	Logger = *logrus.New()
	Logger.Level = logrus.WarnLevel
	Logger.Formatter = &logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "@timestamp",
		},
	}

	if err := os.MkdirAll(LogPath, 0755); err == nil {
		if logfile, err := os.OpenFile(
			filepath.Join(LogPath, LogFile),
			os.O_CREATE|os.O_APPEND|os.O_RDWR,
			0755,
		); err == nil {
			Logger.SetOutput(io.MultiWriter(os.Stdout, logfile))
			return
		}
	}

	logrus.Error("Unable to create WAF log file, logging to stdout only. Elasticsearch logs for WAF may be unavailable.")
	Logger.SetOutput(os.Stdout)
}
