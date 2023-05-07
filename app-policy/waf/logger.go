// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

const LogPath string = "/var/log/calico/waf"
const LogFile string = "waf.log"

const MaxLogCount = 5
const MaxLogCountDuration = 1 * time.Second

var Logger *logrus.Logger

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

func InitializeLogging(writers ...io.Writer) {
	logrus.Info("WAF logging initialization beginning.")
	defer logrus.Info("WAF logging initialization completed.")

	if Logger != nil {
		logrus.Warn("Log already intialized.. skipping")
		return
	}

	writers = append(writers, os.Stderr)
	if logFileWriter, err := ensureLogfile(); err != nil {
		logrus.WithError(err).Warn("failure with logging setup occurred. Stdout log only. Elasticsearch logs for WAF may be unavailable.")
	} else {
		writers = append(writers, logFileWriter...)
	}

	formatter := &logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "@timestamp",
		},
	}

	Logger = &logrus.Logger{
		Level:     logrus.WarnLevel,
		Formatter: NewRateLimitedFormatter(formatter, MaxLogCount, MaxLogCountDuration),
		Out:       io.MultiWriter(writers...),
	}
}

func NewRateLimitedFormatter(formatter logrus.Formatter, logCount uint, timeUnit time.Duration) logrus.Formatter {
	limiter := new(rateLimitedFormatter)
	limiter.queue = make(chan bool, logCount)
	limiter.queueSize = logCount
	limiter.queueTime = timeUnit
	limiter.formatter = formatter
	return limiter
}

type rateLimitedFormatter struct {
	queue     chan bool
	queueSize uint
	queueTime time.Duration
	formatter logrus.Formatter
	activated bool
}

func (r *rateLimitedFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	select {
	case r.queue <- true:
		r.activated = false
		go func() {
			<-time.NewTimer(r.queueTime).C
			<-r.queue
		}()
		return r.formatter.Format(entry)
	default:
		// exceeding set limits (# log entries / duration) will drop log entries
		if r.activated {
			return nil, nil
		}
		r.activated = true
		return nil, fmt.Errorf("reached log output limiting rate of %d per %s", r.queueSize, r.queueTime)
	}
}
