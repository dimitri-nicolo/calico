// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

const LogPath string = "/var/log/calico/waf"
const LogFile string = "waf.log"

const MaxLogCount = 5
const MaxLogCountDuration = 1 * time.Second

var Logger *logutils.RateLimitedLogger

func ensureLogfile() ([]io.Writer, error) {
	var res []io.Writer
	if err := os.MkdirAll(LogPath, 0755); err != nil {
		err = fmt.Errorf("cannot create log path %s", LogPath)
		return nil, err
	}
	logFilePath := filepath.Join(LogPath, LogFile)
	logfile, err := os.OpenFile(
		logFilePath,
		os.O_CREATE|os.O_APPEND|os.O_RDWR,
		0755,
	)
	if err != nil {
		err = fmt.Errorf("cannot create, append or r/w log path %s", logFilePath)
		return nil, err
	}
	res = append(res, logfile)
	return res, nil
}

func InitializeLogging() {
	logrus.Info("WAF logging initialization beginning.")
	defer logrus.Info("WAF logging initialization completed.")

	if Logger != nil {
		logrus.Warn("Log already intialized.. skipping")
		return
	}

	writers := []io.Writer{os.Stderr}
	if extraWriters, err := ensureLogfile(); err != nil {
		logrus.WithError(err).Error("failure with logging setup occurred. Stdout log only. Elasticsearch logs for WAF may be unavailable.")
	} else {
		writers = append(writers, extraWriters...)
	}

	Logger = logutils.NewRateLimitedLogger(logutils.OptLogger(
		&logrus.Logger{
			Level: logrus.WarnLevel,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: time.RFC3339Nano,
				FieldMap: logrus.FieldMap{
					logrus.FieldKeyTime: "@timestamp",
				},
			},
			Out: io.MultiWriter(writers...),
		},
	))
}
